package utils

import (
	"fmt"
	"net/http"
)

// HTTPError defines a custom error structure that includes an HTTP status code and message
type HTTPError struct {
	Code    int    `json:"-"`
	Message string `json:"message"`
}

// Implement the Error() method to satisfy the error interface
func (e *HTTPError) Error() string {
	return e.Message
}

// New creates a new HTTPError instance with a custom status code and message
func NewHTTPError(code int, message string) error {
	return &HTTPError{
		Code:    code,
		Message: message,
	}
}

// BadRequest creates a 400 Bad Request error
func BadRequest(message string) error {
	return NewHTTPError(http.StatusBadRequest, message)
}

// Unauthorized creates a 401 Unauthorized error
func Unauthorized(message string) error {
	return NewHTTPError(http.StatusUnauthorized, message)
}

// Forbidden creates a 403 Forbidden error
func Forbidden(message string) error {
	return NewHTTPError(http.StatusForbidden, message)
}

// NotFound creates a 404 Not Found error
func NotFound(message string) error {
	return NewHTTPError(http.StatusNotFound, message)
}

// InternalServerError creates a 500 Internal Server Error
func InternalServerError(message string) error {
	return NewHTTPError(http.StatusInternalServerError, message)
}

// ServiceUnavailable creates a 503 Service Unavailable error
func ServiceUnavailable(message string) error {
	return NewHTTPError(http.StatusServiceUnavailable, message)
}

// WriteError is a helper function to send the error response as JSON
func WriteError(w http.ResponseWriter, err error) {
	// Check if the error is an instance of HTTPError
	httpErr, ok := err.(*HTTPError)
	if !ok {
		// If not, default to an internal server error
		httpErr = &HTTPError{
			Code:    http.StatusInternalServerError,
			Message: "Internal Server Error",
		}
	}

	// Write the HTTP error response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpErr.Code)
	fmt.Fprintf(w, `{"error": "%s"}`, httpErr.Message)
}
