// Package ui serves the embedded browser UI for trying the TTS API.
package ui

import (
	_ "embed"
	"net/http"
)

//go:embed index.html
var indexHTML []byte

// Handler serves the embedded UI at /ui/.
type Handler struct{}

// NewHandler creates a new UI handler.
func NewHandler() *Handler { return &Handler{} }

// ServeHTTP writes the embedded HTML page.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(indexHTML)
}
