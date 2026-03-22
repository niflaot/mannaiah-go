package http

import (
	"errors"
	"testing"

	"mannaiah/module/campaign/domain"
	corehttp "mannaiah/module/core/http"
)

// TestMapErrorSenderUnavailable verifies sender-unavailable domain errors return controlled 503 responses.
func TestMapErrorSenderUnavailable(t *testing.T) {
	t.Parallel()

	handler := &Handler{}
	err := handler.mapError(domain.ErrSenderUnavailable)

	var appErr *corehttp.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected core AppError, got %T", err)
	}
	if appErr.Status != 503 {
		t.Fatalf("status = %d, want 503", appErr.Status)
	}
	if appErr.Message != "email_sender_unavailable" {
		t.Fatalf("message = %q, want %q", appErr.Message, "email_sender_unavailable")
	}
}

// TestMapErrorInvalidTemplate verifies invalid-template domain errors return controlled 400 responses.
func TestMapErrorInvalidTemplate(t *testing.T) {
	t.Parallel()

	handler := &Handler{}
	err := handler.mapError(domain.ErrInvalidTemplate)

	var appErr *corehttp.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected core AppError, got %T", err)
	}
	if appErr.Status != 400 {
		t.Fatalf("status = %d, want 400", appErr.Status)
	}
	if appErr.Message != "invalid_template" {
		t.Fatalf("message = %q, want %q", appErr.Message, "invalid_template")
	}
}

// TestMapErrorInvalidContactPersonalization verifies contact-personalization errors return controlled 400 responses.
func TestMapErrorInvalidContactPersonalization(t *testing.T) {
	t.Parallel()

	handler := &Handler{}
	err := handler.mapError(domain.ErrContactPersonalization)

	var appErr *corehttp.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected core AppError, got %T", err)
	}
	if appErr.Status != 400 {
		t.Fatalf("status = %d, want 400", appErr.Status)
	}
	if appErr.Message != "invalid_contact_personalization" {
		t.Fatalf("message = %q, want %q", appErr.Message, "invalid_contact_personalization")
	}
}
