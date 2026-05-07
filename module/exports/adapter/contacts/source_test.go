package contacts

import (
	"testing"
	"time"

	"mannaiah/module/exports/port"
)

// TestMembershipFromStatusesPrefersGlobalStatus verifies status rows drive membership export values.
func TestMembershipFromStatusesPrefersGlobalStatus(t *testing.T) {
	occurredAt := time.Date(2026, 5, 6, 13, 0, 0, 0, time.FixedZone("COT", -5*60*60))

	optIn, optInAt, err := membershipFromStatuses([]port.ContactConsentStatus{
		{Channel: "email", Action: "opt_out", OccurredAt: occurredAt.Add(time.Hour)},
		{Channel: "all", Action: "opt_in", OccurredAt: occurredAt},
	})
	if err != nil {
		t.Fatalf("membershipFromStatuses() error = %v", err)
	}

	if !optIn {
		t.Fatalf("optIn = %v, want true", optIn)
	}
	if optInAt.Format(time.RFC3339) != "2026-05-06T18:00:00Z" {
		t.Fatalf("optInAt = %s", optInAt.Format(time.RFC3339))
	}
}

// TestMetadataConsentResolvers verifies checker metadata fallback values are exportable.
func TestMetadataConsentResolvers(t *testing.T) {
	metadata := map[string]string{
		"flock_checker_circle_optin":                 "yes",
		"flock_checker_circle_optin_accepted_at_utc": "2026-05-06T13:00:00Z",
		"flock_checker_privacy_accept":               "yes",
		"flock_checker_privacy_accept_accepted_at":   "2026-05-06 09:30:00",
	}

	membershipOptIn, membershipAt, err := membershipFromMetadata(metadata)
	if err != nil {
		t.Fatalf("membershipFromMetadata() error = %v", err)
	}
	privacyAccepted, privacyAt := resolvePrivacy(metadata)

	if !membershipOptIn || membershipAt.Format(time.RFC3339) != "2026-05-06T13:00:00Z" {
		t.Fatalf("membership = %v %s", membershipOptIn, membershipAt.Format(time.RFC3339))
	}
	if !privacyAccepted || privacyAt.Format(time.RFC3339) != "2026-05-06T14:30:00Z" {
		t.Fatalf("privacy = %v %s", privacyAccepted, privacyAt.Format(time.RFC3339))
	}
}
