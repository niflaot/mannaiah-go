package application

import (
	"errors"
	"testing"

	"mannaiah/module/campaign/domain"
)

// TestNormalizeSenderErrorMapsSESUnverifiedIdentity verifies SES unverified-identity errors map to sender-unavailable.
func TestNormalizeSenderErrorMapsSESUnverifiedIdentity(t *testing.T) {
	t.Parallel()

	raw := errors.New("operation error SESv2: SendEmail, https response error StatusCode: 400, MessageRejected: Email address is not verified. The following identities failed the check in region US-EAST-1: to@example.com, from@example.com")
	mapped := normalizeSenderError(raw)
	if !errors.Is(mapped, domain.ErrSenderUnavailable) {
		t.Fatalf("expected ErrSenderUnavailable, got %v", mapped)
	}
}

// TestNormalizeSenderErrorPassesThroughUnknownErrors verifies unrelated provider errors are preserved.
func TestNormalizeSenderErrorPassesThroughUnknownErrors(t *testing.T) {
	t.Parallel()

	raw := errors.New("operation error SESv2: SendEmail, network timeout")
	mapped := normalizeSenderError(raw)
	if mapped != raw {
		t.Fatalf("expected original error instance to be returned, got %v", mapped)
	}
}

// TestIsSenderUnavailableError verifies provider-availability pattern matching behavior.
func TestIsSenderUnavailableError(t *testing.T) {
	t.Parallel()

	if !isSenderUnavailableError("MessageRejected: Email address is not verified") {
		t.Fatalf("expected unverified-identity message to match sender-unavailable")
	}
	if isSenderUnavailableError("operation canceled: context deadline exceeded") {
		t.Fatalf("expected timeout message to not match sender-unavailable")
	}
}
