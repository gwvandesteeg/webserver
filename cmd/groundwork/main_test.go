package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

// Ensure that os.Getenv works for our EnvGetter type
func TestEnvGetter(t *testing.T) {
	var _ EnvGetter = os.Getenv
}

// mockServer allows us to test the shutdown behaviour of an http server
type mockServer struct {
	Returns error
}

func (m *mockServer) SetKeepAlivesEnabled(_ bool)        {}
func (m *mockServer) Shutdown(ctx context.Context) error { return m.Returns }

// TestShutdown tests the behaviour for our graceful shutdown procedure
func TestShutdown(t *testing.T) {
	// shutdown with no error, this is the normal behaviour we expect
	t.Run("shutdown no error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.TODO())
		errChan := make(chan error)
		m := &mockServer{Returns: nil}

		go shutdown(ctx, m, errChan, time.Second)()
		cancel()
		if err := <-errChan; err != nil {
			t.Errorf("Unexpected error encountered during shutdown, got %s", err)
		}
	})

	// the shutdown now returns an error for some reason, so we make sure we
	// can get it and is the correct error
	t.Run("shutdown with error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.TODO())
		errChan := make(chan error)
		m := &mockServer{Returns: http.ErrServerClosed}

		go shutdown(ctx, m, errChan, time.Second)()
		cancel()
		if err := <-errChan; err == nil {
			t.Errorf("Missing expected error encountered during shutdown want %s, got %s", m.Returns, err)
		}
	})
}

// waitForReady waits for the specified endpoint to become available checking
// every interval for a maximum of timeout in duration
func waitForReady(ctx context.Context, endpoint string, timeout, interval time.Duration) error {
	startTime := time.Now()
	client := &http.Client{Timeout: timeout} // always set a timeout on a HTTP client
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request, %w", err)
	}
	for {
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("error making request, got %s\n", err.Error())
			continue
		}
		if resp.StatusCode == http.StatusOK {
			fmt.Printf("successfull response, got %d\n", resp.StatusCode)
			resp.Body.Close()
			return nil
		}
		resp.Body.Close()
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if time.Since(startTime) >= timeout {
				return fmt.Errorf("server did not reply after %v", timeout)
			}
			time.Sleep(interval)
		}
	}
}

// TestRun tests the run() method
func TestRun(t *testing.T) {
	// setup
	ctx, cancel := context.WithCancel(context.TODO())
	t.Cleanup(cancel)
	// provide our own environment lookup function
	getenv := func(env string) string {
		if env == envVarAddress {
			return ":4001" // this is not the default hostport value
		}
		return ""
	}

	// start the server in the background
	go func() {
		// since the server returns an error we need to check it
		err := run(ctx, getenv, os.Stdout, os.Stderr)
		if err != nil {
			t.Errorf("error during program execution, got %s", err)
		}
	}()

	const (
		interval = 100 * time.Millisecond // interval between failed steps
		timeout  = 5 * time.Second        // how long to try for
	)
	// wait for the server to be ready
	if err := waitForReady(ctx, "http://localhost:4001/hello/name", timeout, interval); err != nil {
		t.Errorf("server never became available in the allocated time, got %s", err)
	}
	// shut down the server
	cancel()
}
