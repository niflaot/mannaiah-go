package domain

import "errors"

var (
	// ErrInvalidEmail is returned when recipient email values are invalid.
	ErrInvalidEmail = errors.New("email recipient is required")
	// ErrInvalidSubject is returned when subject values are invalid.
	ErrInvalidSubject = errors.New("email subject is required")
	// ErrNotFound is returned when delivery rows are missing.
	ErrNotFound = errors.New("email delivery not found")
)
