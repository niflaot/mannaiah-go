package http

import (
	"net/url"
	"testing"
)

// TestDecodeWebhookRequestJSON verifies JSON webhook envelopes are decoded.
func TestDecodeWebhookRequestJSON(t *testing.T) {
	payload := []byte(`{"Type":"SubscriptionConfirmation","TopicArn":"arn:aws:sns:us-east-2:123456789012:ses-events","Token":"abc","SubscribeURL":"https://sns.us-east-2.amazonaws.com/?Action=ConfirmSubscription"}`)

	request, err := decodeWebhookRequest(payload, nil)
	if err != nil {
		t.Fatalf("decodeWebhookRequest() error = %v", err)
	}
	if request.MessageType != "SubscriptionConfirmation" {
		t.Fatalf("decodeWebhookRequest() MessageType = %q, want %q", request.MessageType, "SubscriptionConfirmation")
	}
	if request.SubscribeURL == "" {
		t.Fatalf("decodeWebhookRequest() SubscribeURL must not be empty")
	}
}

// TestDecodeWebhookRequestEmbeddedJSON verifies string-encoded JSON payloads are decoded.
func TestDecodeWebhookRequestEmbeddedJSON(t *testing.T) {
	payload := []byte(`"{\"Type\":\"SubscriptionConfirmation\",\"TopicArn\":\"arn:aws:sns:us-east-2:123456789012:ses-events\",\"Token\":\"abc\",\"SubscribeURL\":\"https://sns.us-east-2.amazonaws.com/?Action=ConfirmSubscription\"}"`)

	request, err := decodeWebhookRequest(payload, nil)
	if err != nil {
		t.Fatalf("decodeWebhookRequest() error = %v", err)
	}
	if request.MessageType != "SubscriptionConfirmation" {
		t.Fatalf("decodeWebhookRequest() MessageType = %q, want %q", request.MessageType, "SubscriptionConfirmation")
	}
	if request.TopicARN == "" {
		t.Fatalf("decodeWebhookRequest() TopicARN must not be empty")
	}
}

// TestDecodeWebhookRequestFormEncoded verifies form-encoded webhook payloads are decoded.
func TestDecodeWebhookRequestFormEncoded(t *testing.T) {
	subscribeURL := "https://sns.us-east-2.amazonaws.com/?Action=ConfirmSubscription"
	payload := []byte("Type=SubscriptionConfirmation&TopicArn=arn%3Aaws%3Asns%3Aus-east-2%3A123456789012%3Ases-events&Token=abc&SubscribeURL=" + url.QueryEscape(subscribeURL))

	request, err := decodeWebhookRequest(payload, nil)
	if err != nil {
		t.Fatalf("decodeWebhookRequest() error = %v", err)
	}
	if request.MessageType != "SubscriptionConfirmation" {
		t.Fatalf("decodeWebhookRequest() MessageType = %q, want %q", request.MessageType, "SubscriptionConfirmation")
	}
	if request.SubscribeURL != subscribeURL {
		t.Fatalf("decodeWebhookRequest() SubscribeURL = %q, want %q", request.SubscribeURL, subscribeURL)
	}
}

// TestDecodeWebhookRequestBodyParserFallback verifies fallback parser behavior.
func TestDecodeWebhookRequestBodyParserFallback(t *testing.T) {
	parser := func(out any) error {
		request := out.(*webhookRequest)
		request.ProviderMessageID = "provider-message"
		request.Status = "delivered"
		return nil
	}

	request, err := decodeWebhookRequest(nil, parser)
	if err != nil {
		t.Fatalf("decodeWebhookRequest() error = %v", err)
	}
	if request.ProviderMessageID != "provider-message" {
		t.Fatalf("decodeWebhookRequest() ProviderMessageID = %q, want %q", request.ProviderMessageID, "provider-message")
	}
}

// TestDecodeWebhookRequestInvalid verifies invalid payloads return errors.
func TestDecodeWebhookRequestInvalid(t *testing.T) {
	_, err := decodeWebhookRequest([]byte("not-json-or-form"), nil)
	if err == nil {
		t.Fatalf("decodeWebhookRequest() error = nil, want non-nil")
	}
}
