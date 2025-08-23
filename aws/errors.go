package aws

import (
	"errors"
	"fmt"

	"github.com/jim-barber-he/go/util"
)

// NewCreateDirError creates a new error for directory creation failure.
func NewCreateDirError(directory string) error {
	return util.NewError("failed to create directory", directory)
}

// NewOneParameterError creates a new error for invalid parameter count.
func NewOneParameterError(numParameters int) error {
	return util.NewError("failed to validate parameters", fmt.Sprintf("expected 1 parameter, got %d", numParameters))
}

// NewParameterDeleteError creates a new error for parameter deletion failure.
func NewParameterDeleteError(parameter string) error {
	return util.NewError("failed to delete parameter", parameter)
}

// NewParameterDescribeError creates a new error for parameter description failure.
func NewParameterDescribeError(parameter string) error {
	return util.NewError("failed to describe parameter", parameter)
}

// NewParameterGetError creates a new error for parameter retrieval failure.
func NewParameterGetError(parameter string) error {
	return util.NewError("failed to get parameter", parameter)
}

// NewParameterPutError creates a new error for parameter storage failure.
func NewParameterPutError(parameter string) error {
	return util.NewError("failed to put parameter", parameter)
}

// NewWriteCacheFileError creates a new error for failure to write to the cache file.
func NewWriteCacheFileError(file string) error {
	return util.NewError("failed to write cache file", file)
}

var (
	errGetCachePath       = errors.New("failed to get cache file path")
	errGetToken           = errors.New("failed to get token")
	errGoRoutine          = errors.New("goroutine error")
	errMarshalJSON        = errors.New("failed to marshal cache data to JSON")
	errOpenBrowser        = errors.New("failed to open browser for authentication")
	errParameterGetByPath = errors.New("failed to get parameters by path")
	errReadCacheFile      = errors.New("failed to read cache file")
	errRegisterClient     = errors.New("failed to register client")
	errSSOTimeout         = errors.New("SSO login attempt timed out")
	errStartDeviceAuth    = errors.New("failed to start device authorisation")
	errUnmarshalCacheFile = errors.New("failed to unmarshal cache file data")
	errWriteCacheFile     = errors.New("failed to write cache file")
)

// Errors used in the tests.
var (
	errAccessDenied         = errors.New("access denied")
	errAPIError             = errors.New("API error")
	errAuthorizationPending = errors.New("AuthorizationPendingException: authorization pending")
	errDecryptionFailed     = errors.New("decryption failed")
	errKeyIDNotSet          = errors.New("KeyId not set for SecureString")
	errPersistentError      = errors.New("persistent error")
	errValidationError      = errors.New("validation error")
)
