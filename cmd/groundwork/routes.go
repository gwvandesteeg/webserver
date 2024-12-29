package main

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/felixge/httpsnoop"
)

type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

// RecoveryHandler is a minimal and incomplete system for capturing panics
// in handlers and having them recover and serve a meaningful response back
// to the client.
func RecoveryHandler(logger Logger, wrapped http.Handler) http.Handler {
	// Endpoints wrap with this handler will change the response of

	// $ curl -v http://localhost:4000/example/panic
	// * Host localhost:4000 was resolved.
	// * IPv6: ::1
	// * IPv4: 127.0.0.1
	// *   Trying [::1]:4000...
	// * Connected to localhost (::1) port 4000
	// > GET /example/panic HTTP/1.1
	// > Host: localhost:4000
	// > User-Agent: curl/8.5.0
	// > Accept: */*
	// >
	// * Empty reply from server
	// * Closing connection
	// curl: (52) Empty reply from server
	//
	// 	Into a more meaningful
	//
	// $ curl -v http://localhost:4000/example/panic
	// * Host localhost:4000 was resolved.
	// * IPv6: ::1
	// * IPv4: 127.0.0.1
	// *   Trying [::1]:4000...
	// * Connected to localhost (::1) port 4000
	// > GET /example/panic HTTP/1.1
	// > Host: localhost:4000
	// > User-Agent: curl/8.5.0
	// > Accept: */*
	// >
	// < HTTP/1.1 500 Internal Server Error
	// < Date: Tue, 26 Nov 2024 10:43:24 GMT
	// < Content-Length: 0
	// <
	// * Connection #0 to host localhost left intact

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				logger.Error("panic", slog.Any("reason", err))
			}
		}()
		wrapped.ServeHTTP(w, req)
	})
}

// WrapWithLogger is a minimal and incomplete request logger, a more complete
// explanation behind the problems with a request logger can be found in the
// httpsnoop documentation https://pkg.go.dev/github.com/felixge/httpsnoop.
// The logging done here utilises the slog functionality and expects an slog
// compattible logger to be provided.
func WrapWithLogger(logger Logger, wrapped http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		t := time.Now()
		url := *req.URL
		metrics := httpsnoop.CaptureMetrics(wrapped, w, req)

		logger.Info("served",
			slog.String("host", req.Host),
			slog.String("username", url.User.Username()),
			slog.Time("received", t),
			slog.String("method", req.Method),
			slog.String("uri", url.RequestURI()),
			slog.String("proto", req.Proto),
			slog.Int("status", metrics.Code),
			slog.Int64("size", metrics.Written),
			slog.Duration("duration", metrics.Duration),
		)
	})
}

// setupRoutes sets up all the routes and relevant middlewares
func setupRoutes(mux *http.ServeMux, logger Logger) error {
	// This format requires go1.22 or later
	mux.Handle("GET /hello/{name}", WrapWithLogger(logger, handleHello()))
	// The /healthz endpoint is synonymous to the /readyz endpoint and as such
	// it exposes the same handler
	// Refer to https://kubernetes.io/docs/reference/using-api/health-checks/
	// for naming convention information and deprecation of /healthz endpoint.
	mux.Handle("GET /healthz", readinessCheckEndpoint())
	// A service that is not ready does not get traffic routed to it.
	mux.Handle("GET /readyz", readinessCheckEndpoint())
	// A service that is not live gets restarted.
	mux.Handle("GET /livez", livenessCheckEndpoint())
	// this route generates a panic without a recovery wrapper which causes it to not log the request
	mux.Handle("GET /example/panic", WrapWithLogger(logger, generatePanic()))
	// this route generates a panic with a recovery wrapper, however this then needs logging and handling appropiately for action by the developers
	mux.Handle("GET /example/recoveredpanic", WrapWithLogger(logger, RecoveryHandler(logger, generatePanic())))
	// this route never completes
	mux.Handle("GET /example/never", WrapWithLogger(logger, neverCompletingResponse(logger)))
	// this route is very slow to allow for demonstrating the graceful shutdown behaviour
	mux.Handle("GET /example/slow", WrapWithLogger(logger, slowCompletingResponse(logger)))

	return nil
}
