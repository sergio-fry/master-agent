package api

import (
	"encoding/json"
	"net/http"
)

// ErrorBody is the JSON error response shape for the HTTP API.
// Spec: specs/07-web-ui.md — HTTP status + {"error":"..."}.
type ErrorBody struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

// WriteError writes a JSON error response with the given HTTP status.
// The body is always {"error":"<message>"} with optional code.
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteErrorCode(w, status, message, "")
}

// WriteErrorCode writes a JSON error response with an optional machine-readable code.
func WriteErrorCode(w http.ResponseWriter, status int, message, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	body := ErrorBody{Error: message}
	if code != "" {
		body.Code = code
	}
	_ = json.NewEncoder(w).Encode(body)
}
