package api

import (
	"encoding/json"
	"net/http"
)

// ErrorBody is the JSON error response shape for the HTTP API.
// Spec: specs/07-web-ui.md — HTTP status + {"error":"..."}.
type ErrorBody struct {
	Error string `json:"error"`
}

// WriteError writes a JSON error response with the given HTTP status.
// The body is always {"error":"<message>"}.
func WriteError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorBody{Error: message})
}
