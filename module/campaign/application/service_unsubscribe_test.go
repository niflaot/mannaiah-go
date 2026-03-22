package application

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestBuildMarketingOptOutTokenSignsPayload verifies token payload/signature formats and exp/iat values.
func TestBuildMarketingOptOutTokenSignsPayload(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 22, 18, 0, 0, 0, time.UTC)
	payload := marketingOptOutTokenPayload{
		Email:      "jane@example.com",
		Name:       optionalStringPointer("Jane Doe"),
		CampaignID: optionalStringPointer("cmp-1"),
	}
	token, err := buildMarketingOptOutToken(payload, "secret", now, 2*time.Hour)
	if err != nil {
		t.Fatalf("buildMarketingOptOutToken() error = %v", err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		t.Fatalf("token = %q, want payload.signature", token)
	}
	if got, want := parts[1], signMarketingOptOutToken(parts[0], "secret"); got != want {
		t.Fatalf("signature = %q, want %q", got, want)
	}

	decodedPayload, decodeErr := base64.RawURLEncoding.DecodeString(parts[0])
	if decodeErr != nil {
		t.Fatalf("DecodeString() error = %v", decodeErr)
	}
	var parsed marketingOptOutTokenPayload
	if unmarshalErr := json.Unmarshal(decodedPayload, &parsed); unmarshalErr != nil {
		t.Fatalf("Unmarshal() error = %v", unmarshalErr)
	}
	if parsed.Email != "jane@example.com" {
		t.Fatalf("parsed.Email = %q, want jane@example.com", parsed.Email)
	}
	if parsed.Name == nil || *parsed.Name != "Jane Doe" {
		t.Fatalf("parsed.Name = %v, want Jane Doe", parsed.Name)
	}
	if parsed.CampaignID == nil || *parsed.CampaignID != "cmp-1" {
		t.Fatalf("parsed.CampaignID = %v, want cmp-1", parsed.CampaignID)
	}
	if parsed.IssuedAt != now.Unix() {
		t.Fatalf("parsed.IssuedAt = %d, want %d", parsed.IssuedAt, now.Unix())
	}
	if parsed.ExpiresAt != now.Add(2*time.Hour).Unix() {
		t.Fatalf("parsed.ExpiresAt = %d, want %d", parsed.ExpiresAt, now.Add(2*time.Hour).Unix())
	}
}

// TestNormalizeUnsubscribeBaseURLTrimsTrailingSlash verifies base URL normalization behavior.
func TestNormalizeUnsubscribeBaseURLTrimsTrailingSlash(t *testing.T) {
	t.Parallel()

	if got, want := normalizeUnsubscribeBaseURL(" https://mannaiah.flockstore.co/ "), "https://mannaiah.flockstore.co"; got != want {
		t.Fatalf("normalizeUnsubscribeBaseURL() = %q, want %q", got, want)
	}
}
