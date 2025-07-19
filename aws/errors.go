package aws

import (
	"errors"
	"fmt"

	"github.com/jim-barber-he/go/util"
)

// NewCreateDirError creates a new error for directory creation failure.
func NewCreateDirError(directory string) error {
	return &util.Error{
		Msg:   "failed to create directory: ",
		Param: directory,
	}
}

// NewOneParameterError creates a new error for invalid parameter count.
func NewOneParameterError(numParameters int) error {
	return &util.Error{
		Msg:   "failed to validate parameters: ",
		Param: fmt.Sprintf("expected 1 parameter, got %d", numParameters),
	}
}

// NewParameterDeleteError creates a new error for parameter deletion failure.
func NewParameterDeleteError(parameter string) error {
	return &util.Error{
		Msg:   "failed to delete parameter: ",
		Param: parameter,
	}
}

// NewParameterDescribeError creates a new error for parameter description failure.
func NewParameterDescribeError(parameter string) error {
	return &util.Error{
		Msg:   "failed to describe parameter: ",
		Param: parameter,
	}
}

// NewParameterGetError creates a new error for parameter retrieval failure.
func NewParameterGetError(parameter string) error {
	return &util.Error{
		Msg:   "failed to get parameter: ",
		Param: parameter,
	}
}

// NewParameterPutError creates a new error for parameter storage failure.
func NewParameterPutError(parameter string) error {
	return &util.Error{
		Msg:   "failed to put parameter: ",
		Param: parameter,
	}
}

// NewWriteCacheFileError creates a new error for failure to write to the cache file.
func NewWriteCacheFileError(file string) error {
	return &util.Error{
		Msg:   "failed to write cache file: ",
		Param: file,
	}
}

var (
	errGetCachePath       = errors.New("failed to get cache file path")
	errGetToken           = errors.New("failed to get token")
	errMarshalJSON        = errors.New("failed to marshal cache data to JSON")
	errOpenBrowser        = errors.New("failed to open browser for authentication")
	errParameterGetByPath = errors.New("failed to get parameters by path")
	errRegisterClient     = errors.New("failed to register client")
	errSSOTimeout         = errors.New("SSO login attempt timed out")
	errStartDeviceAuth    = errors.New("failed to start device authorisation")
	errWriteCacheFile     = errors.New("failed to write cache file")
)
