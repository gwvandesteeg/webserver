package main

import (
	"fmt"
	"net/http"
)

const (
	contentTypeTextPlain       = "text/plain; charset=utf-8" // the MIME type and charset for plain text
	contentTypeApplicationJSON = "application/json"
)

func readinessCheckEndpoint() http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", contentTypeTextPlain)
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "OK\n")
		})
}

func livenessCheckEndpoint() http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", contentTypeTextPlain)
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "OK\n")
		})
}
