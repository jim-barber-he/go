package aws

import (
	"errors"
	"fmt"
)

var (
	errCacheFileRead  = errors.New("failed to read cache file")
	errCacheFileWrite = errors.New("failed to write cache file")

	errDirCreate = errors.New("failed to create directory")

	errGetCachePath       = errors.New("failed to get cache file path")
	errGetToken           = errors.New("failed to get token")
	errGoRoutine          = errors.New("goroutine error")
	errMarshalJSON        = errors.New("failed to marshal cache data to JSON")
	errOpenBrowser        = errors.New("failed to open browser for authentication")
	errParameterDelete    = errors.New("failed to delete parameter")
	errParameterDescribe  = errors.New("failed to describe parameter")
	errParameterGet       = errors.New("failed to get parameter")
	errParameterGetByPath = errors.New("failed to get parameters by path")
	errParameterPut       = errors.New("failed to put parameter")
	errParametersValidate = errors.New("failed to validate parameters")

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

// NewCreateDirError creates a new error for directory creation failure.
func NewCreateDirError(directory string) error {
	return fmt.Errorf("%w: %s", errDirCreate, directory)
}

// NewOneParameterError creates a new error for invalid parameter count.
func NewOneParameterError(numParameters int) error {
	return fmt.Errorf("%w: expected 1 parameter, got %d", errParametersValidate, numParameters)
}

// NewParameterDeleteError creates a new error for parameter deletion failure.
func NewParameterDeleteError(parameter string) error {
	return fmt.Errorf("%w: %s", errParameterDelete, parameter)
}

// NewParameterDescribeError creates a new error for parameter description failure.
func NewParameterDescribeError(parameter string) error {
	return fmt.Errorf("%w: %s", errParameterDescribe, parameter)
}

// NewParameterGetError creates a new error for parameter retrieval failure.
func NewParameterGetError(parameter string) error {
	return fmt.Errorf("%w: %s", errParameterGet, parameter)
}

// NewParameterPutError creates a new error for parameter storage failure.
func NewParameterPutError(parameter string) error {
	return fmt.Errorf("%w: %s", errParameterPut, parameter)
}

// NewWriteCacheFileError creates a new error for failure to write to the cache file.
func NewWriteCacheFileError(file string) error {
	return fmt.Errorf("%w: %s", errCacheFileWrite, file)
}
