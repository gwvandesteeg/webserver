package main

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// handleHello returns a handler for the hello endpoint
func handleHello() http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			name := req.PathValue("name")
			w.Header().Set("Content-Type", contentTypeTextPlain)
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "Hello %s\n", name)
		})
}

// generatePanic demonstrates what happens when a panic occurs during the
func generatePanic() http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			panic("example")
		})
}

// neverCompletingResponse is used to demonstrate a part of the graceful
// request shutdown, all it does is just sits there counting seconds.
// Because this request never completes except through the context
// cancellation no data written gets
func neverCompletingResponse(logger Logger) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", contentTypeTextPlain)
			// This status is still counted by the request logger
			w.WriteHeader(http.StatusRequestTimeout)
			ctx, cancel := context.WithTimeout(req.Context(), 2*gracePeriod)
			defer cancel()
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					logger.Info("context done")
					return
				case t := <-ticker.C:
					logger.Info("tick")
					// Nothing we write here gets through to the client,
					// however it does get counted as bytes written by the logger
					fmt.Fprintf(w, "item: %s\n", t.Format(time.RFC3339Nano))
				}
			}
		})
}

// slowCompletingResponse is used to demonstrate a part of the graceful request
// shutdown, it writes some partial output to the response, requests the user
// presses Ctrl-C on the server which then writes the another line after a
// short delay.
func slowCompletingResponse(logger Logger) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", contentTypeTextPlain)
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "first line\n")
			logger.Info("press Ctrl-C now on this server to trigger the shutdown")
			time.Sleep(httpWriteTimeout / 2)
			fmt.Fprintf(w, "last line\n")
			logger.Info("handler completed before httpWriteTimeout completed")
		})
}
