package model

import "fmt"

// AppError is a structured application error that carries an HTTP status code
// and a machine-readable error code for the API response.
type AppError struct {
	StatusCode int    `json:"-"`
	Code       string `json:"code"`
	Message    string `json:"message"`
}

// Error implements the error interface.
func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// NewAppError creates a new AppError.
func NewAppError(statusCode int, code, message string) *AppError {
	return &AppError{
		StatusCode: statusCode,
		Code:       code,
		Message:    message,
	}
}

// Common application errors.
var (
	ErrUnauthorized   = NewAppError(401, "UNAUTHORIZED", "authentication required")
	ErrForbidden      = NewAppError(403, "PLAN_LIMIT_EXCEEDED", "plan limit exceeded")
	ErrNotFound       = NewAppError(404, "NOT_FOUND", "resource not found")
	ErrConflict       = NewAppError(409, "CONFLICT", "resource already exists")
	ErrInternalServer = NewAppError(500, "INTERNAL_ERROR", "internal server error")
)

// NewValidationError creates a 400 validation error.
func NewValidationError(message string) *AppError {
	return NewAppError(400, "VALIDATION_ERROR", message)
}
