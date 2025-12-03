package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/pako-tts/server/internal/domain"
)

// WriteError writes an API error response.
func WriteError(w http.ResponseWriter, err *domain.APIError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.StatusCode)
	json.NewEncoder(w).Encode(domain.NewErrorResponse(err))
}

// WriteJSON writes a JSON response.
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
