package domain

import "errors"

var (
	// ErrInvalidID is returned when campaign id values are invalid.
	ErrInvalidID = errors.New("campaign id is required")
	// ErrInvalidName is returned when campaign name values are invalid.
	ErrInvalidName = errors.New("campaign name is required")
	// ErrInvalidSlug is returned when campaign slug values are invalid.
	ErrInvalidSlug = errors.New("campaign slug is required")
	// ErrNotFound is returned when campaign rows are missing.
	ErrNotFound = errors.New("campaign not found")
	// ErrSendConflict is returned when send operations are called after processing/sent states.
	ErrSendConflict = errors.New("campaign cannot be sent in current status")
	// ErrInvalidTestEmail is returned when a test send is requested without a valid recipient email.
	ErrInvalidTestEmail = errors.New("test recipient email is required")
	// ErrSenderNotConfigured is returned when a test send is attempted without an email sender.
	ErrSenderNotConfigured = errors.New("email sender is not configured")
	// ErrSenderUnavailable is returned when email provider dependencies reject delivery due configuration/outage constraints.
	ErrSenderUnavailable = errors.New("email sender is unavailable")
	// ErrInvalidTemplate is returned when campaign template parsing or execution fails.
	ErrInvalidTemplate = errors.New("campaign template is invalid")
	// ErrContactPersonalization is returned when contact personalization cannot be resolved for an explicit contact id.
	ErrContactPersonalization = errors.New("campaign contact personalization failed")
)
