package http

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strings"
)

// ComputeWebhookSignature computes one Shopify-compatible webhook signature.
func ComputeWebhookSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(strings.TrimSpace(secret)))
	_, _ = mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// VerifyWebhookSignature verifies one Shopify webhook signature.
func VerifyWebhookSignature(secret string, body []byte, signature string) bool {
	expected := ComputeWebhookSignature(secret, body)
	received := strings.TrimSpace(signature)
	if expected == "" || received == "" {
		return false
	}

	return hmac.Equal([]byte(expected), []byte(received))
}
