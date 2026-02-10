package application

import "errors"

var (
	// ErrUnauthorized is returned when request authentication fails.
	ErrUnauthorized = errors.New("unauthorized")
	// ErrForbidden is returned when request authorization fails.
	ErrForbidden = errors.New("forbidden")
	// ErrNilVerifier is returned when verifier dependencies are missing.
	ErrNilVerifier = errors.New("token verifier must not be nil")
)
