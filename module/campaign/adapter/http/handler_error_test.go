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
