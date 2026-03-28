package domain

import "errors"

var (
	// ErrInvalidID is returned when segment id values are invalid.
	ErrInvalidID = errors.New("segment id is required")
	// ErrInvalidName is returned when segment name values are invalid.
	ErrInvalidName = errors.New("segment name is required")
	// ErrInvalidSlug is returned when segment slug values are invalid.
	ErrInvalidSlug = errors.New("segment slug is required")
	// ErrInvalidFilter is returned when segment filter definitions are invalid.
	ErrInvalidFilter = errors.New("segment filter is invalid")
	// ErrNotFound is returned when segment rows are missing.
	ErrNotFound = errors.New("segment not found")
	// ErrParentNotFound is returned when a referenced parent segment does not exist.
	ErrParentNotFound = errors.New("parent segment not found")
	// ErrCircularReference is returned when a sub-segment references itself or creates a cycle.
	ErrCircularReference = errors.New("circular segment reference detected")
)
