package aws_test

import "errors"

var (
	errAccessDenied         = errors.New("access denied")
	errAPIError             = errors.New("API error")
	errAuthorizationPending = errors.New("AuthorizationPendingException: authorization pending")
	errDecryptionFailed     = errors.New("decryption failed")
	errKeyIDNotSet          = errors.New("KeyId not set for SecureString")
	errPersistentError      = errors.New("persistent error")
	errValidationError      = errors.New("validation error")
)
