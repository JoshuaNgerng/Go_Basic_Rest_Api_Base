package server

import (
	"encoding/json"
	"log"
	"net/http"
)

// writeError is used by middleware and error handlers, which sit outside the
// generated typed responses.
func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// badRequest is the generated handlers' RequestErrorHandlerFunc (bad JSON etc).
func badRequest(w http.ResponseWriter, _ *http.Request, _ error) {
	writeError(w, http.StatusBadRequest, "invalid request body")
}

// internalError is the generated handlers' ResponseErrorHandlerFunc. It never
// leaks internal error details to clients.
func internalError(w http.ResponseWriter, _ *http.Request, err error) {
	log.Printf("handler error: %v", err)
	writeError(w, http.StatusInternalServerError, "internal error")
}
