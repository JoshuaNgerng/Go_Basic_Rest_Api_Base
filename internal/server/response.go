package server

import (
	"encoding/json"
	"net/http"
)

// writeError is used by middleware and error handlers, which sit outside the
// generated typed responses.
func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
