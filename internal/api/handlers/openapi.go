package handlers

import (
	"encoding/json"
	"net/http"

	"gopkg.in/yaml.v3"
)

// OpenAPIHandler serves the OpenAPI specification.
type OpenAPIHandler struct {
	specJSON []byte
	specYAML []byte
}

// NewOpenAPIHandler creates a new OpenAPI handler with embedded spec.
func NewOpenAPIHandler(yamlSpec []byte) (*OpenAPIHandler, error) {
	// Parse YAML
	var spec map[string]interface{}
	if err := yaml.Unmarshal(yamlSpec, &spec); err != nil {
		return nil, err
	}

	// Convert to JSON
	jsonSpec, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}

	return &OpenAPIHandler{
		specJSON: jsonSpec,
		specYAML: yamlSpec,
	}, nil
}

// ServeSpecJSON handles GET /openapi.json and /api/v1/openapi.json.
func (h *OpenAPIHandler) ServeSpecJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	w.Write(h.specJSON)
}

// ServeSpecYAML handles GET /openapi.yaml and /api/v1/openapi.yaml.
func (h *OpenAPIHandler) ServeSpecYAML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	w.Write(h.specYAML)
}
