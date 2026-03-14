package service

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
	"mannaiah/module/woocommerce/port"
)

// membershipStamperMock defines membership stamper behavior for tests.
type membershipStamperMock struct {
	calls []membershipStampCall
}

// membershipStampCall defines captured stamp invocation values.
type membershipStampCall struct {
	email      string
	channel    string
	action     port.MembershipAction
	source     string
	occurredAt *time.Time
}

// StampByEmail captures stamp payload values.
func (m *membershipStamperMock) StampByEmail(ctx context.Context, email string, channel string, action port.MembershipAction, source string, occurredAt *time.Time) error {
	m.calls = append(m.calls, membershipStampCall{
		email:      email,
		channel:    channel,
		action:     action,
		source:     source,
		occurredAt: occurredAt,
	})
	return nil
}

// TestResolveMembershipAction verifies circle checker decisions map to membership actions.
func TestResolveMembershipAction(t *testing.T) {
	action, occurredAt, ok := resolveMembershipAction(map[string]string{
		"flock_checker_circle_optin":                 "yes",
		"flock_checker_circle_optin_accepted_at_utc": "2026-03-13T18:05:22Z",
	})
	if !ok {
		t.Fatalf("resolveMembershipAction() ok = false, want true")
	}
	if action != port.MembershipActionOptIn {
		t.Fatalf("action = %q, want %q", action, port.MembershipActionOptIn)
	}
	if occurredAt == nil || occurredAt.UTC().Format(time.RFC3339) != "2026-03-13T18:05:22Z" {
		t.Fatalf("occurredAt = %v, want %q", occurredAt, "2026-03-13T18:05:22Z")
	}

	action, occurredAt, ok = resolveMembershipAction(map[string]string{
		"flock_checker_circle_optin":                 "no",
		"flock_checker_circle_optin_rejected_at_utc": "2026-03-14T18:05:22Z",
	})
	if !ok {
		t.Fatalf("resolveMembershipAction(optout) ok = false, want true")
	}
	if action != port.MembershipActionOptOut {
		t.Fatalf("action = %q, want %q", action, port.MembershipActionOptOut)
	}
	if occurredAt == nil || occurredAt.UTC().Format(time.RFC3339) != "2026-03-14T18:05:22Z" {
		t.Fatalf("occurredAt = %v, want %q", occurredAt, "2026-03-14T18:05:22Z")
	}
}

// TestStampMembershipUsesAllChannel verifies Woo sync writes all-channel membership stamps.
func TestStampMembershipUsesAllChannel(t *testing.T) {
	stamper := &membershipStamperMock{}
	service := &ContactSyncService{membershipStamper: stamper, logger: zap.NewNop()}

	service.stampMembership(context.Background(), port.ContactSyncCommand{
		Email: "john@example.com",
		Metadata: map[string]string{
			"flock_checker_circle_optin":                 "yes",
			"flock_checker_circle_optin_accepted_at_utc": "2026-03-13T18:05:22Z",
		},
	})

	if len(stamper.calls) != 1 {
		t.Fatalf("len(calls) = %d, want 1", len(stamper.calls))
	}
	if stamper.calls[0].channel != "all" {
		t.Fatalf("channel = %q, want %q", stamper.calls[0].channel, "all")
	}
	if stamper.calls[0].action != port.MembershipActionOptIn {
		t.Fatalf("action = %q, want %q", stamper.calls[0].action, port.MembershipActionOptIn)
	}
}
