package http

import "testing"

// TestVerifyWebhookSignature verifies Shopify-compatible HMAC validation.
func TestVerifyWebhookSignature(t *testing.T) {
	secret := "secret"
	body := []byte(`{"id":123}`)
	signature := ComputeWebhookSignature(secret, body)

	if !VerifyWebhookSignature(secret, body, signature) {
		t.Fatalf("VerifyWebhookSignature() should accept valid signatures")
	}
	if VerifyWebhookSignature(secret, body, "invalid") {
		t.Fatalf("VerifyWebhookSignature() should reject invalid signatures")
	}
}
