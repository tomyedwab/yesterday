package yesterdaygo

import (
	"fmt"
	"net/http"
)

// ErrorType represents different categories of errors
type ErrorType int

const (
	// ErrorTypeUnknown represents an unknown error
	ErrorTypeUnknown ErrorType = iota
	// ErrorTypeNetwork represents network-related errors
	ErrorTypeNetwork
	// ErrorTypeAuthentication represents authentication-related errors
	ErrorTypeAuthentication
	// ErrorTypeAPI represents API-specific errors
	ErrorTypeAPI
	// ErrorTypeValidation represents validation errors
	ErrorTypeValidation
)

// Error represents a structured error with type information
type Error struct {
	Type       ErrorType
	Message    string
	StatusCode int
	Cause      error
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying cause error
func (e *Error) Unwrap() error {
	return e.Cause
}

// IsType checks if the error is of a specific type
func (e *Error) IsType(errorType ErrorType) bool {
	return e.Type == errorType
}

// NewError creates a new Error with the specified type and message
func NewError(errorType ErrorType, message string) *Error {
	return &Error{
		Type:    errorType,
		Message: message,
	}
}

// NewErrorWithCause creates a new Error with the specified type, message, and underlying cause
func NewErrorWithCause(errorType ErrorType, message string, cause error) *Error {
	return &Error{
		Type:    errorType,
		Message: message,
		Cause:   cause,
	}
}

// NewNetworkError creates a network-related error
func NewNetworkError(message string, cause error) *Error {
	return NewErrorWithCause(ErrorTypeNetwork, message, cause)
}

// NewAuthenticationError creates an authentication-related error
func NewAuthenticationError(message string) *Error {
	return NewError(ErrorTypeAuthentication, message)
}

// NewAPIError creates an API-related error with status code
func NewAPIError(message string, statusCode int) *Error {
	return &Error{
		Type:       ErrorTypeAPI,
		Message:    message,
		StatusCode: statusCode,
	}
}

// NewValidationError creates a validation-related error
func NewValidationError(message string) *Error {
	return NewError(ErrorTypeValidation, message)
}

// IsNetworkError checks if an error is network-related
func IsNetworkError(err error) bool {
	if yErr, ok := err.(*Error); ok {
		return yErr.IsType(ErrorTypeNetwork)
	}
	return false
}

// IsAuthenticationError checks if an error is authentication-related
func IsAuthenticationError(err error) bool {
	if yErr, ok := err.(*Error); ok {
		return yErr.IsType(ErrorTypeAuthentication)
	}
	return false
}

// IsAPIError checks if an error is API-related
func IsAPIError(err error) bool {
	if yErr, ok := err.(*Error); ok {
		return yErr.IsType(ErrorTypeAPI)
	}
	return false
}

// IsValidationError checks if an error is validation-related
func IsValidationError(err error) bool {
	if yErr, ok := err.(*Error); ok {
		return yErr.IsType(ErrorTypeValidation)
	}
	return false
}

// WrapHTTPError wraps an HTTP response into an appropriate Error type
func WrapHTTPError(resp *http.Response, message string) *Error {
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return NewAuthenticationError(fmt.Sprintf("%s: %s", message, resp.Status))
	case http.StatusBadRequest:
		return NewValidationError(fmt.Sprintf("%s: %s", message, resp.Status))
	default:
		return NewAPIError(fmt.Sprintf("%s: %s", message, resp.Status), resp.StatusCode)
	}
}
