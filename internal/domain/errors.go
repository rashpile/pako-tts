package domain

import (
	"fmt"
	"net/http"
)

// APIError represents an API error with HTTP status code.
type APIError struct {
	StatusCode int            `json:"-"`
	Code       string         `json:"code"`
	Message    string         `json:"message"`
	Details    map[string]any `json:"details,omitempty"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// WithDetails returns a new error with additional details.
func (e *APIError) WithDetails(details map[string]any) *APIError {
	return &APIError{
		StatusCode: e.StatusCode,
		Code:       e.Code,
		Message:    e.Message,
		Details:    details,
	}
}

// WithMessage returns a new error with a custom message.
func (e *APIError) WithMessage(msg string) *APIError {
	return &APIError{
		StatusCode: e.StatusCode,
		Code:       e.Code,
		Message:    msg,
		Details:    e.Details,
	}
}

// Standard API errors
var (
	// ErrJobNotFound indicates the requested job does not exist.
	ErrJobNotFound = &APIError{
		StatusCode: http.StatusNotFound,
		Code:       "JOB_NOT_FOUND",
		Message:    "Job not found",
	}

	// ErrResultExpired indicates the job result has expired.
	ErrResultExpired = &APIError{
		StatusCode: http.StatusGone,
		Code:       "RESULT_EXPIRED",
		Message:    "Result has expired. Results are retained for 24 hours.",
	}

	// ErrJobNotComplete indicates the job is not yet complete.
	ErrJobNotComplete = &APIError{
		StatusCode: http.StatusTooEarly,
		Code:       "JOB_NOT_COMPLETE",
		Message:    "Job not yet completed",
	}

	// ErrValidation indicates a validation error.
	ErrValidation = &APIError{
		StatusCode: http.StatusUnprocessableEntity,
		Code:       "VALIDATION_ERROR",
		Message:    "Validation failed",
	}

	// ErrTextTooLong indicates the text exceeds the sync endpoint limit.
	ErrTextTooLong = &APIError{
		StatusCode: http.StatusRequestEntityTooLarge,
		Code:       "TEXT_TOO_LONG",
		Message:    "Text exceeds 5000 character limit. Use POST /api/v1/jobs for longer texts.",
	}

	// ErrProviderUnavailable indicates the TTS provider is not available.
	ErrProviderUnavailable = &APIError{
		StatusCode: http.StatusServiceUnavailable,
		Code:       "PROVIDER_UNAVAILABLE",
		Message:    "TTS provider unavailable",
	}

	// ErrInternalServer indicates an internal server error.
	ErrInternalServer = &APIError{
		StatusCode: http.StatusInternalServerError,
		Code:       "INTERNAL_ERROR",
		Message:    "Internal server error",
	}

	// ErrInvalidVoice indicates an invalid voice ID.
	ErrInvalidVoice = &APIError{
		StatusCode: http.StatusUnprocessableEntity,
		Code:       "INVALID_VOICE",
		Message:    "Invalid voice_id",
	}

	// ErrInvalidFormat indicates an invalid output format.
	ErrInvalidFormat = &APIError{
		StatusCode: http.StatusUnprocessableEntity,
		Code:       "INVALID_FORMAT",
		Message:    "Invalid output_format. Must be 'mp3' or 'wav'.",
	}
)

// ErrorResponse wraps an API error for JSON response.
type ErrorResponse struct {
	Error *APIError `json:"error"`
}

// NewErrorResponse creates a new error response.
func NewErrorResponse(err *APIError) *ErrorResponse {
	return &ErrorResponse{Error: err}
}
