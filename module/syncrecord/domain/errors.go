package domain

import "errors"

var (
	// ErrInvalidRunID is returned when run id values are invalid.
	ErrInvalidRunID = errors.New("sync run id is required")
	// ErrInvalidKind is returned when run kind values are invalid.
	ErrInvalidKind = errors.New("sync run kind is invalid")
	// ErrInvalidTrigger is returned when run trigger values are invalid.
	ErrInvalidTrigger = errors.New("sync run trigger is invalid")
	// ErrRunNotFound is returned when run rows are missing.
	ErrRunNotFound = errors.New("sync run not found")
	// ErrRunAlreadyFinished is returned when completed runs are modified.
	ErrRunAlreadyFinished = errors.New("sync run already finished")
)
