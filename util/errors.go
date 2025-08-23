package util

import (
	"errors"
	"fmt"
)

// Error is a generic type for errors that take a parameter.
type Error struct {
	Msg   string
	Param string
}

// Error implements the Error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Msg, e.Param)
}

// NewError creates a new Error with the given message and parameter.
func NewError(msg, param string) *Error {
	return &Error{Msg: msg, Param: param}
}

// Exported error constants for better reusability
var (
	ErrCommandTimedOut = errors.New("command timed out")
	ErrTerminalSize    = errors.New("failed to get terminal size")
)

// Keep legacy unexported versions for backward compatibility
var (
	errCommandTimedOut = ErrCommandTimedOut
	errTerminalSize    = ErrTerminalSize
)
