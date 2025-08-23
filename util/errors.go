package util

import (
	"errors"
	"fmt"
)

// ErrorType represents the category of error.
type ErrorType string

const (
	// ErrorTypeSystem represents system-level errors (file system, network, etc.)
	ErrorTypeSystem ErrorType = "system"
	// ErrorTypeValidation represents input validation errors
	ErrorTypeValidation ErrorType = "validation"
	// ErrorTypeTimeout represents timeout-related errors
	ErrorTypeTimeout ErrorType = "timeout"
	// ErrorTypeConfiguration represents configuration-related errors
	ErrorTypeConfiguration ErrorType = "configuration"
	// ErrorTypeOperation represents general operation errors
	ErrorTypeOperation ErrorType = "operation"
)

// Error is a generic type for errors that take a parameter.
// Deprecated: Use AppError for new code.
type Error struct {
	Msg   string
	Param string
}

// Error implements the Error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("%s%s", e.Msg, e.Param)
}

// AppError provides structured error handling with categorization and wrapping.
type AppError struct {
	Type    ErrorType
	Message string
	Err     error
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// Unwrap returns the wrapped error.
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewSystemError creates a new system error.
func NewSystemError(message string, err error) *AppError {
	return &AppError{
		Type:    ErrorTypeSystem,
		Message: message,
		Err:     err,
	}
}

// NewValidationError creates a new validation error.
func NewValidationError(message string) *AppError {
	return &AppError{
		Type:    ErrorTypeValidation,
		Message: message,
	}
}

// NewTimeoutError creates a new timeout error.
func NewTimeoutError(message string, err error) *AppError {
	return &AppError{
		Type:    ErrorTypeTimeout,
		Message: message,
		Err:     err,
	}
}

// NewConfigurationError creates a new configuration error.
func NewConfigurationError(message string, err error) *AppError {
	return &AppError{
		Type:    ErrorTypeConfiguration,
		Message: message,
		Err:     err,
	}
}

// NewOperationError creates a new operation error.
func NewOperationError(message string, err error) *AppError {
	return &AppError{
		Type:    ErrorTypeOperation,
		Message: message,
		Err:     err,
	}
}

var (
	// ErrCommandTimedOut indicates a command exceeded its timeout
	ErrCommandTimedOut = NewTimeoutError("command timed out", nil)
	// ErrTerminalSize indicates failure to get terminal size
	ErrTerminalSize = NewSystemError("failed to get terminal size", nil)

	// Deprecated error variables - use structured errors above
	errCommandTimedOut = errors.New("command timed out")
	errTerminalSize    = errors.New("failed to get terminal size")
)
