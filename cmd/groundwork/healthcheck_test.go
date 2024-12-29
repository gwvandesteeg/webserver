package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReadinessCheckEndpoint(t *testing.T) {
	// setup the request
	ctx, cancel := context.WithCancel(context.TODO())
	t.Cleanup(cancel)
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "https://localhost:4000/ready", nil)
	// setup the response recorder
	w := httptest.NewRecorder()
	// do the request
	readinessCheckEndpoint().ServeHTTP(w, req)
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
	if string(body) != "OK\n" {
		t.Errorf("wrong response detected expected, got %s", string(body))
	}
}

func TestLivenessCheckEndpoint(t *testing.T) {
	// setup the request
	ctx, cancel := context.WithCancel(context.TODO())
	t.Cleanup(cancel)
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "https://localhost:4000/live", nil)
	// setup the response recorder
	w := httptest.NewRecorder()
	// do the request
	livenessCheckEndpoint().ServeHTTP(w, req)
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
	if string(body) != "OK\n" {
		t.Errorf("wrong response detected expected, got %s", string(body))
	}
}
