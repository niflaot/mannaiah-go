package falabella

import "testing"

// TestSignParamsDeterministic verifies deterministic signature behavior.
func TestSignParamsDeterministic(t *testing.T) {
	signature := signParams("secret", map[string]string{
		"Version": "1.0",
		"Action":  "GetBrands",
		"UserID":  "user",
	})

	if signature == "" {
		t.Fatalf("signature should not be empty")
	}
	if signature != "7DC015A4ECDE81630036978797CEF0D3C648F94C02A3EA30B3C744B14510B5E8" {
		t.Fatalf("signature mismatch = %q", signature)
	}
}
