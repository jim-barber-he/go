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

// NewError creates a new Error with the given message and parameter.
func NewError(msg, param string) *Error {
	return &Error{Msg: msg, Param: param}
}

// Error implements the Error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Msg, e.Param)
}

var (
	errCommandTimedOut = errors.New("command timed out")
	errTerminalSize    = errors.New("failed to get terminal size")
)
