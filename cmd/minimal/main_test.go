package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
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

// TestHandleHello_direct is where we do an explicit test of the handler in
// isolation without a routing mux, this approach is needed here due to the
// use of req.PathValue in the code base which requires the value to be set
// to a meaningful value.
func TestHandleHello_direct(t *testing.T) {
	// setup the request
	ctx, cancel := context.WithCancel(context.TODO())
	t.Cleanup(cancel)
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "https://localhost:4000/hello/Testing", nil)
	req.SetPathValue("name", "Testing")
	// setup the response recorder
	w := httptest.NewRecorder()
	// do the request
	handleHello().ServeHTTP(w, req)
	// get the result and process it
	resp := w.Result()

	defer func() {
		// always close the response body as a client
		if err := resp.Body.Close(); err != nil {
			t.Errorf("error closing body, got %s", err)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("incorrect HTTP status code returned want %d, got %d", http.StatusOK, resp.StatusCode)
	}
	expectedContentType := "text/plain; charset=utf-8"
	if contentType := resp.Header.Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("incorrect Content-Type header returned expected %s, got %s", expectedContentType, contentType)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("error reading body, got %s", err)
	}
	if string(body) != "Hello Testing\n" {
		t.Errorf("wrong response detected expected, got %s", string(body))
	}
}

// TestHandleHello_mux is a different appraoch to testing the handler by
// setting up the mux router where the path interpolation will work using
// the same rules as in the implementation of the routes.
func TestHandleHello_mux(t *testing.T) {
	// set up the server
	mux := http.NewServeMux()
	mux.Handle("GET /hello/{name}", handleHello())
	ts := httptest.NewServer(mux)
	// set up the request
	ctx, cancel := context.WithCancel(context.TODO())
	t.Cleanup(cancel)
	reqUri, err := url.JoinPath(ts.URL, "/hello/Testing")
	if err != nil {
		t.Errorf("error forming URI, got %s", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqUri, nil)
	if err != nil {
		t.Errorf("error forming request, got %s", err)
	}
	// issue the request to the client
	client := ts.Client()
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("error processing request, got %s", err)
	}
	// process the response
	defer func() {
		// always close the response body as a client
		if err := resp.Body.Close(); err != nil {
			t.Errorf("error closing body, got %s", err)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("incorrect HTTP status code returned want %d, got %d", http.StatusOK, resp.StatusCode)
	}
	expectedContentType := "text/plain; charset=utf-8"
	if contentType := resp.Header.Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("incorrect Content-Type header returned expected %s, got %s", expectedContentType, contentType)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("error reading body, got %s", err)
	}
	if string(body) != "Hello Testing\n" {
		t.Errorf("wrong response detected expected, got %s", string(body))
	}
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
		err := run(ctx, getenv)
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
