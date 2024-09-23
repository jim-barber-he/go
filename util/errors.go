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
	return fmt.Sprintf("%s%s", e.Msg, e.Param)
}

var (
	errCommandTimedOut = errors.New("command timed out")
	errTerminalSize    = errors.New("failed to get terminal size")
)
