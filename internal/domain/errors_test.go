package domain

import (
	"net/http"
	"testing"
)

func TestAPIError_Error(t *testing.T) {
	err := &APIError{
		StatusCode: http.StatusNotFound,
		Code:       "NOT_FOUND",
		Message:    "Resource not found",
	}

	expected := "NOT_FOUND: Resource not found"
	if err.Error() != expected {
		t.Errorf("Expected error string %q, got %q", expected, err.Error())
	}
}

func TestAPIError_WithDetails(t *testing.T) {
	original := &APIError{
		StatusCode: http.StatusBadRequest,
		Code:       "VALIDATION_ERROR",
		Message:    "Validation failed",
	}

	details := map[string]any{
		"field":   "email",
		"message": "invalid format",
	}

	withDetails := original.WithDetails(details)

	if withDetails.StatusCode != original.StatusCode {
		t.Errorf("Expected StatusCode %d, got %d", original.StatusCode, withDetails.StatusCode)
	}
	if withDetails.Code != original.Code {
		t.Errorf("Expected Code %s, got %s", original.Code, withDetails.Code)
	}
	if withDetails.Message != original.Message {
		t.Errorf("Expected Message %s, got %s", original.Message, withDetails.Message)
	}
	if withDetails.Details["field"] != "email" {
		t.Errorf("Expected Details[field] to be 'email', got %v", withDetails.Details["field"])
	}

	// Original should not be modified
	if original.Details != nil {
		t.Error("Original error Details should remain nil")
	}
}

func TestAPIError_WithMessage(t *testing.T) {
	original := &APIError{
		StatusCode: http.StatusBadRequest,
		Code:       "VALIDATION_ERROR",
		Message:    "Validation failed",
	}

	customMessage := "Custom validation message"
	withMessage := original.WithMessage(customMessage)

	if withMessage.StatusCode != original.StatusCode {
		t.Errorf("Expected StatusCode %d, got %d", original.StatusCode, withMessage.StatusCode)
	}
	if withMessage.Code != original.Code {
		t.Errorf("Expected Code %s, got %s", original.Code, withMessage.Code)
	}
	if withMessage.Message != customMessage {
		t.Errorf("Expected Message %s, got %s", customMessage, withMessage.Message)
	}

	// Original should not be modified
	if original.Message != "Validation failed" {
		t.Error("Original error Message should not be modified")
	}
}

func TestNewErrorResponse(t *testing.T) {
	apiErr := ErrJobNotFound

	response := NewErrorResponse(apiErr)

	if response.Error != apiErr {
		t.Error("Expected response.Error to be the same as input error")
	}
}

func TestStandardErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        *APIError
		statusCode int
		code       string
	}{
		{"ErrJobNotFound", ErrJobNotFound, http.StatusNotFound, "JOB_NOT_FOUND"},
		{"ErrResultExpired", ErrResultExpired, http.StatusGone, "RESULT_EXPIRED"},
		{"ErrJobNotComplete", ErrJobNotComplete, http.StatusTooEarly, "JOB_NOT_COMPLETE"},
		{"ErrValidation", ErrValidation, http.StatusUnprocessableEntity, "VALIDATION_ERROR"},
		{"ErrTextTooLong", ErrTextTooLong, http.StatusRequestEntityTooLarge, "TEXT_TOO_LONG"},
		{"ErrProviderUnavailable", ErrProviderUnavailable, http.StatusServiceUnavailable, "PROVIDER_UNAVAILABLE"},
		{"ErrInternalServer", ErrInternalServer, http.StatusInternalServerError, "INTERNAL_ERROR"},
		{"ErrInvalidVoice", ErrInvalidVoice, http.StatusUnprocessableEntity, "INVALID_VOICE"},
		{"ErrInvalidFormat", ErrInvalidFormat, http.StatusUnprocessableEntity, "INVALID_FORMAT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.StatusCode != tt.statusCode {
				t.Errorf("Expected StatusCode %d, got %d", tt.statusCode, tt.err.StatusCode)
			}
			if tt.err.Code != tt.code {
				t.Errorf("Expected Code %s, got %s", tt.code, tt.err.Code)
			}
			if tt.err.Message == "" {
				t.Error("Expected Message to be set")
			}
		})
	}
}
