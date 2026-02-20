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
	if signature != "9f6de2578b90e1280016e42176691285813aede3a0630e9695d5bb8201901ddb" {
		t.Fatalf("signature mismatch = %q", signature)
	}
}

// TestCanonicalQuerySellerCenterExample verifies documented canonical query behavior.
func TestCanonicalQuerySellerCenterExample(t *testing.T) {
	canonical := canonicalQuery(map[string]string{
		"Action":    "GetBrands",
		"Format":    "XML",
		"Timestamp": "2018-02-02T10:46:17+00:00",
		"UserID":    "test@example.com",
		"Version":   "1.0",
	})

	const expected = "Action=GetBrands&Format=XML&Timestamp=2018-02-02T10%3A46%3A17%2B00%3A00&UserID=test%40example.com&Version=1.0"
	if canonical != expected {
		t.Fatalf("canonical query mismatch = %q", canonical)
	}
}

// TestCanonicalQueryRawSellerCenterExample verifies raw canonical query behavior.
func TestCanonicalQueryRawSellerCenterExample(t *testing.T) {
	canonical := canonicalQueryRaw(map[string]string{
		"Action":    "GetBrands",
		"Format":    "XML",
		"Timestamp": "2018-02-02T10:46:17+0000",
		"UserID":    "test@example.com",
		"Version":   "1.0",
	})

	const expected = "Action=GetBrands&Format=XML&Timestamp=2018-02-02T10:46:17+0000&UserID=test@example.com&Version=1.0"
	if canonical != expected {
		t.Fatalf("raw canonical query mismatch = %q", canonical)
	}
}
