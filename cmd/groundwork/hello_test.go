package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

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
