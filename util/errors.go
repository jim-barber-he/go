package util

import (
	"errors"
)

var (
	errCommandTimedOut = errors.New("command timed out")
	errTerminalSize    = errors.New("failed to get terminal size")
)
