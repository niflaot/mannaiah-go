package ses

import (
	"strings"
	"testing"

	"mannaiah/module/email/port"
)

// TestBuildSNSStringToSignNotification verifies canonical field ordering for notification messages.
func TestBuildSNSStringToSignNotification(t *testing.T) {
	t.Parallel()

	value, err := buildSNSStringToSign(port.SNSMessage{
		Type:      "Notification",
		Message:   "{\"notificationType\":\"Delivery\"}",
		MessageID: "msg-1",
		Subject:   "Amazon SES Email Event Notification",
		Timestamp: "2026-03-22T20:20:20.000Z",
		TopicARN:  "arn:aws:sns:us-east-1:123456789012:ses-events",
	})
	if err != nil {
		t.Fatalf("buildSNSStringToSign() error = %v", err)
	}
	if !strings.Contains(value, "Message\n{\"notificationType\":\"Delivery\"}\n") {
		t.Fatalf("stringToSign missing Message block: %q", value)
	}
	if !strings.Contains(value, "Subject\nAmazon SES Email Event Notification\n") {
		t.Fatalf("stringToSign missing Subject block: %q", value)
	}
	if !strings.HasSuffix(value, "Type\nNotification\n") {
		t.Fatalf("stringToSign suffix invalid: %q", value)
	}
}

// TestBuildSNSStringToSignSubscriptionConfirmation verifies canonical field ordering for subscription confirmation messages.
func TestBuildSNSStringToSignSubscriptionConfirmation(t *testing.T) {
	t.Parallel()

	value, err := buildSNSStringToSign(port.SNSMessage{
		Type:         "SubscriptionConfirmation",
		Message:      "You have chosen to subscribe.",
		MessageID:    "msg-2",
		SubscribeURL: "https://sns.us-east-1.amazonaws.com/?Action=ConfirmSubscription",
		Token:        "token-1",
		Timestamp:    "2026-03-22T20:20:20.000Z",
		TopicARN:     "arn:aws:sns:us-east-1:123456789012:ses-events",
	})
	if err != nil {
		t.Fatalf("buildSNSStringToSign() error = %v", err)
	}
	if !strings.Contains(value, "SubscribeURL\nhttps://sns.us-east-1.amazonaws.com/?Action=ConfirmSubscription\n") {
		t.Fatalf("stringToSign missing SubscribeURL block: %q", value)
	}
	if !strings.Contains(value, "Timestamp\n2026-03-22T20:20:20.000Z\n") {
		t.Fatalf("stringToSign missing Timestamp block: %q", value)
	}
	if !strings.Contains(value, "Token\ntoken-1\n") {
		t.Fatalf("stringToSign missing Token block: %q", value)
	}
	timestampIndex := strings.Index(value, "Timestamp\n2026-03-22T20:20:20.000Z\n")
	tokenIndex := strings.Index(value, "Token\ntoken-1\n")
	if timestampIndex < 0 || tokenIndex < 0 || timestampIndex > tokenIndex {
		t.Fatalf("stringToSign order invalid, expected Timestamp before Token: %q", value)
	}
}

// TestDigestSNSStringToSignUnsupportedVersion verifies unsupported signature-version errors.
func TestDigestSNSStringToSignUnsupportedVersion(t *testing.T) {
	t.Parallel()

	if _, _, err := digestSNSStringToSign("3", "payload"); err == nil {
		t.Fatalf("digestSNSStringToSign() error = nil, want unsupported-version error")
	}
}
