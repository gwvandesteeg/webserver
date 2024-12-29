package main

/*
  Minimal is a very minimal program that demonstrates how to safely bootstrap
  a web server as well as shows how to do the maximum amount of testing for a
  web server and the handlers without resorting to external resources beyond
  the standard library.
*/

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/gwvandesteeg/webserver/internal/buildversion"
)

// EnvGetter defines a function type used to do lookups from a process environment
type EnvGetter func(key string) string

// httpServer is a simple interface we can use for the shutdown function to
// make it easier to test in isolation
type httpServer interface {
	SetKeepAlivesEnabled(v bool)
	Shutdown(ctx context.Context) error
}

// shutdown returns a function that deals with the graceful termination of
// a http.Server to ensure in-flight requests are terminated gracefully
// and be allowed to complete if able
func shutdown(ctx context.Context, httpServer httpServer, errChan chan<- error, gracePeriod time.Duration) func() {
	return func() {
		// wait for the context to be cancelled/terminated
		<-ctx.Done()
		// set graceful shutdown time limit
		shutdownCtx, cancel := context.WithTimeout(context.Background(), gracePeriod)
		defer cancel()
		// disable keepalives to ensure long waiting requests timeout
		httpServer.SetKeepAlivesEnabled(false)
		// call graceful shutdown
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			// write the error to the channel
			errChan <- err
		}
		close(errChan)
	}
}

const (
	httpReadTimeout    = 10 * time.Second // maximum amount of time the server has to read the request including the body
	httpWriteTimeout   = 30 * time.Second // maximum amount of time the server has to write the response
	httpMaxHeaderBytes = 1 << 20          // maximum number of bytes accepted in the headers for received requests
	defaultHostPort    = ":4000"          // the default hostport address to listen on
	envVarAddress      = "ADDRESS"        // the environment variable name
	gracePeriod        = 29 * time.Second // the default graceperiod for kubernetes is 30 seconds, so give us one less
)

// run sets up and executes our program in a testable manner
func run(ctx context.Context, getenv EnvGetter, stdout, stderr io.Writer) error {
	// setup the signal handlers to interrupt correctly on Ctrl-C as done by
	// a user, or by SIGTERM done by process managers like Kubernetes
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	logger := slog.New(slog.NewTextHandler(stdout, nil))
	var goVersion string
	if buildinfo, ok := debug.ReadBuildInfo(); ok {
		goVersion = buildinfo.GoVersion
	}
	logger.Info("Starting groundwork",
		// slog.String("build", buildversion.Build),
		slog.String("branch", buildversion.GitBranch),
		slog.String("commit", buildversion.GitCommit),
		slog.String("goVersion", goVersion))

	// figure out the address we're going to listen on
	hostport := getenv(envVarAddress)
	if strings.TrimSpace(hostport) == "" {
		hostport = defaultHostPort
	}
	// setup the routes
	mux := http.NewServeMux()
	if err := setupRoutes(mux, logger); err != nil {
		logger.Error("Unable to setup routes")
		return err
	}
	// configure the server
	srv := &http.Server{
		Addr:    hostport,
		Handler: mux,
		// The timeouts are related to the graceful shutdown of your request handlers,
		// if a handler does not complete in this time the client will get an "Empty reply from server" type error
		// and the request logger will likely log that the request completed succesfully according to it.
		ReadTimeout:    httpReadTimeout,
		WriteTimeout:   httpWriteTimeout,
		MaxHeaderBytes: httpMaxHeaderBytes,
		// create a simple handler for errors establishing connections
		ErrorLog: slog.NewLogLogger(slog.NewTextHandler(stderr, nil), slog.LevelError),
	}

	// setup the graceful shutdown mechanism
	errChan := make(chan error)
	go shutdown(ctx, srv, errChan, 30*time.Second)()

	logger.Info("Listening on", slog.String("hostport", hostport))
	// start the webserver until it is terminated
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	logger.Info("Stopped serving new connections.")
	// pull the error from the graceful shutdown if any
	err := <-errChan

	logger.Info("Shutdown completed.")
	return err
}

const (
	errorExitCode = 1 // the exit code to use if there was an error returned from running the main code
)

func main() {
	ctx := context.Background()
	// start the main code execution
	if err := run(ctx, os.Getenv, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(errorExitCode)
	}
}
