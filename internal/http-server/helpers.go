package httpserver

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
)

// APIError represents the structure of error responses
type APIError struct {
	Error string `json:"error"`
}

// WriteJSON sends data as JSON with the specified HTTP status code.
// Automatically sets the correct Content-Type header.
func WriteJSON(w http.ResponseWriter, statusCode int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	// For 204 No Content or 202 Accepted with no data, don't encode anything
	if data == nil && (statusCode == http.StatusNoContent || statusCode == http.StatusAccepted) {
		return nil
	}
	// If data is nil for other status codes, send an empty object
	if data == nil && statusCode != http.StatusNoContent && statusCode != http.StatusAccepted {
		data = map[string]any{}
	}

	return json.NewEncoder(w).Encode(data)
}

// apiFunc defines the signature for API handler functions
// that return an error for centralized error handling
type apiFunc func(w http.ResponseWriter, r *http.Request) error

// makeHTTPHandlerFunc converts an apiFunc to a standard http.HandlerFunc,
// adding unified error handling
func makeHTTPHandlerFunc(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			handleError(w, r, err)
		}
	}
}

// handleError processes errors and sends appropriate HTTP responses
func handleError(w http.ResponseWriter, r *http.Request, err error) {
	var validationErr *ValidationErr
	if errors.As(err, &validationErr) {
		WriteJSON(w, http.StatusBadRequest, APIError{Error: validationErr.Error()})
		return
	}

	var notFoundErr *NotFoundErr
	if errors.As(err, &notFoundErr) {
		WriteJSON(w, http.StatusNotFound, APIError{Error: notFoundErr.Error()})
		return
	}

	var unaouthorizedErr *UnaouthorizedError
	if errors.As(err, &unaouthorizedErr) {
		WriteJSON(w, http.StatusUnauthorized, APIError{Error: unaouthorizedErr.Error()})
	}

	errString := strings.ToLower(err.Error())

	if strings.Contains(errString, "required") ||
		strings.Contains(errString, "invalid") ||
		strings.Contains(errString, "format") {
		WriteJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
		return
	}

	if strings.Contains(errString, "unaouthorized") ||
		strings.Contains(errString, "unauthenticated") {
		WriteJSON(w, http.StatusUnauthorized, APIError{Error: "Unaouthorized"})
	}

	if strings.Contains(errString, "not found") {
		WriteJSON(w, http.StatusNotFound, APIError{Error: err.Error()})
		return
	}

	log.Printf("HTTP handler error", "error", err, "path", r.URL.Path)
	WriteJSON(w, http.StatusInternalServerError, APIError{Error: "An unexpected error occurred"})
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
