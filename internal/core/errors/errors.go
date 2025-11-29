package errors

import (
	"fmt"
	"net/http"
)

// ErrorType represents the type of error
type ErrorType string

const (
	ErrTypeValidation   ErrorType = "VALIDATION_ERROR"
	ErrTypeNotFound     ErrorType = "NOT_FOUND"
	ErrTypeConflict     ErrorType = "CONFLICT"
	ErrTypeInternal     ErrorType = "INTERNAL_ERROR"
	ErrTypeDeployment   ErrorType = "DEPLOYMENT_ERROR"
	ErrTypeContainer    ErrorType = "CONTAINER_ERROR"
	ErrTypeDatabase     ErrorType = "DATABASE_ERROR"
	ErrTypeUnauthorized ErrorType = "UNAUTHORIZED"
	ErrTypeProxy        ErrorType = "PROXY_ERROR"
)

// AppError is the application-level error type
type AppError struct {
	Type       ErrorType
	Message    string
	Cause      error
	Details    map[string]interface{}
	StatusCode int
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

// NewValidationError creates a validation error
func NewValidationError(message string, details map[string]interface{}) *AppError {
	return &AppError{
		Type:       ErrTypeValidation,
		Message:    message,
		Details:    details,
		StatusCode: http.StatusBadRequest,
	}
}

// NewNotFoundError creates a not found error
func NewNotFoundError(resource, id string) *AppError {
	return &AppError{
		Type:       ErrTypeNotFound,
		Message:    fmt.Sprintf("%s with id '%s' not found", resource, id),
		StatusCode: http.StatusNotFound,
	}
}

// NewConflictError creates a conflict error
func NewConflictError(message string) *AppError {
	return &AppError{
		Type:       ErrTypeConflict,
		Message:    message,
		StatusCode: http.StatusConflict,
	}
}

// NewInternalError creates an internal error
func NewInternalError(message string, cause error) *AppError {
	return &AppError{
		Type:       ErrTypeInternal,
		Message:    message,
		Cause:      cause,
		StatusCode: http.StatusInternalServerError,
	}
}

// NewDeploymentError creates a deployment error
func NewDeploymentError(message string, cause error) *AppError {
	return &AppError{
		Type:       ErrTypeDeployment,
		Message:    message,
		Cause:      cause,
		StatusCode: http.StatusInternalServerError,
	}
}

// NewContainerError creates a container error
func NewContainerError(message string, cause error) *AppError {
	return &AppError{
		Type:       ErrTypeContainer,
		Message:    message,
		Cause:      cause,
		StatusCode: http.StatusInternalServerError,
	}
}

// NewUnauthorizedError creates an unauthorized error
func NewUnauthorizedError(message string) *AppError {
	return &AppError{
		Type:       ErrTypeUnauthorized,
		Message:    message,
		StatusCode: http.StatusUnauthorized,
	}
}

// NewProxyError creates a proxy error
func NewProxyError(message string, cause error) *AppError {
	return &AppError{
		Type:       ErrTypeProxy,
		Message:    message,
		Cause:      cause,
		StatusCode: http.StatusInternalServerError,
	}
}

// IsNotFound checks if error is a not found error
func IsNotFound(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type == ErrTypeNotFound
	}
	return false
}

// IsValidation checks if error is a validation error
func IsValidation(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type == ErrTypeValidation
	}
	return false
}

// New creates a new AppError with the given type and message
func New(errType ErrorType, message string) *AppError {
	statusCode := http.StatusInternalServerError
	switch errType {
	case ErrTypeValidation:
		statusCode = http.StatusBadRequest
	case ErrTypeNotFound:
		statusCode = http.StatusNotFound
	case ErrTypeConflict:
		statusCode = http.StatusConflict
	case ErrTypeUnauthorized:
		statusCode = http.StatusUnauthorized
	}
	return &AppError{
		Type:       errType,
		Message:    message,
		StatusCode: statusCode,
	}
}
