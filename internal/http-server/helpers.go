package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

// APIError represents the structure of error responses
type APIError struct {
	Error string `json:"error"`
}

// respondJSON sends a JSON response with the given status code
func (s *Server) respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			s.log.Error("Failed to encode JSON response", "error", err)
		}
	}
}

// respondError sends an error response with appropriate status code
func (s *Server) respondError(w http.ResponseWriter, status int, message string) {
	s.respondJSON(w, status, map[string]string{
		"error": message,
	})
}

// handleError processes an error and sends the appropriate HTTP response
// This centralizes your error handling logic
func (s *Server) handleError(w http.ResponseWriter, err error) {
	// Check for custom error types
	var validationErr *ValidationErr
	if errors.As(err, &validationErr) {
		s.respondError(w, http.StatusBadRequest, validationErr.Error())
		return
	}

	var notFoundErr *NotFoundErr
	if errors.As(err, &notFoundErr) {
		s.respondError(w, http.StatusNotFound, notFoundErr.Error())
		return
	}

	var unauthorizedErr *UnaouthorizedError
	if errors.As(err, &unauthorizedErr) {
		s.respondError(w, http.StatusUnauthorized, unauthorizedErr.Error())
		return
	}

	// Check error message for common patterns
	errMsg := strings.ToLower(err.Error())

	if strings.Contains(errMsg, "required") ||
		strings.Contains(errMsg, "invalid") ||
		strings.Contains(errMsg, "format") {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if strings.Contains(errMsg, "not found") {
		s.respondError(w, http.StatusNotFound, err.Error())
		return
	}

	if strings.Contains(errMsg, "unauthorized") ||
		strings.Contains(errMsg, "unauthenticated") {
		s.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Default to 500 for unknown errors
	s.log.Error("Internal server error", "error", err)
	s.respondError(w, http.StatusInternalServerError, "An unexpected error occurred")
}

type ValidationErr struct {
	Message string
}

func (e *ValidationErr) Error() string {
	return e.Message
}

func NewValidationError(message string) error {
	return &ValidationErr{
		Message: message,
	}
}

type NotFoundErr struct {
	Message string
}

func (e *NotFoundErr) Error() string {
	return e.Message
}

func NewNotFoundError(message string) error {
	return &NotFoundErr{
		Message: message,
	}
}

type UnaouthorizedError struct {
	Message string
}

func (e *UnaouthorizedError) Error() string {
	return e.Message
}

func NewUnaouthorizedError(message string) error {
	return &UnaouthorizedError{
		Message: message,
	}
}
