package domain

import "errors"

var (
	// ErrInvalidEmail is returned when recipient email values are invalid.
	ErrInvalidEmail = errors.New("email recipient is required")
	// ErrInvalidSubject is returned when subject values are invalid.
	ErrInvalidSubject = errors.New("email subject is required")
	// ErrInvalidWebhookPayload is returned when webhook payload values are invalid.
	ErrInvalidWebhookPayload = errors.New("email webhook payload is invalid")
	// ErrInvalidWebhookSignature is returned when webhook signature verification fails.
	ErrInvalidWebhookSignature = errors.New("email webhook signature is invalid")
	// ErrWebhookTopicMismatch is returned when webhook topic arn values do not match expected configuration.
	ErrWebhookTopicMismatch = errors.New("email webhook topic arn mismatch")
	// ErrWebhookSubscriptionConfirmationFailed is returned when sns subscription confirmation fails.
	ErrWebhookSubscriptionConfirmationFailed = errors.New("email webhook subscription confirmation failed")
	// ErrNotFound is returned when delivery rows are missing.
	ErrNotFound = errors.New("email delivery not found")
)
