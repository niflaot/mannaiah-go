package platform

import (
	"errors"
	"testing"
)

// TestConfigNormalizedAppliesDefaults verifies zero-values are normalized to safe defaults.
func TestConfigNormalizedAppliesDefaults(t *testing.T) {
	normalized := Config{}.Normalized()

	if normalized.GoChannelBuffer != 100 {
		t.Fatalf("GoChannelBuffer = %d, want %d", normalized.GoChannelBuffer, 100)
	}
	if normalized.RetryMaxRetries != 0 {
		t.Fatalf("RetryMaxRetries = %d, want %d", normalized.RetryMaxRetries, 0)
	}
	if normalized.RetryInitialIntervalMS != 100 {
		t.Fatalf("RetryInitialIntervalMS = %d, want %d", normalized.RetryInitialIntervalMS, 100)
	}
	if normalized.RetryMaxIntervalMS != 2000 {
		t.Fatalf("RetryMaxIntervalMS = %d, want %d", normalized.RetryMaxIntervalMS, 2000)
	}
	if normalized.RetryMultiplier != 2.0 {
		t.Fatalf("RetryMultiplier = %f, want %f", normalized.RetryMultiplier, 2.0)
	}
	if normalized.DLQSuffix != ".dlq" {
		t.Fatalf("DLQSuffix = %q, want %q", normalized.DLQSuffix, ".dlq")
	}
}

// TestConfigNormalizedPreservesValues verifies valid values are preserved by normalization.
func TestConfigNormalizedPreservesValues(t *testing.T) {
	normalized := Config{
		GoChannelBuffer:        321,
		RetryMaxRetries:        4,
		RetryInitialIntervalMS: 10,
		RetryMaxIntervalMS:     50,
		RetryMultiplier:        1.5,
		DLQSuffix:              ".dead",
	}.Normalized()

	if normalized.GoChannelBuffer != 321 {
		t.Fatalf("GoChannelBuffer = %d, want %d", normalized.GoChannelBuffer, 321)
	}
	if normalized.RetryMaxRetries != 4 {
		t.Fatalf("RetryMaxRetries = %d, want %d", normalized.RetryMaxRetries, 4)
	}
	if normalized.RetryInitialIntervalMS != 10 {
		t.Fatalf("RetryInitialIntervalMS = %d, want %d", normalized.RetryInitialIntervalMS, 10)
	}
	if normalized.RetryMaxIntervalMS != 50 {
		t.Fatalf("RetryMaxIntervalMS = %d, want %d", normalized.RetryMaxIntervalMS, 50)
	}
	if normalized.RetryMultiplier != 1.5 {
		t.Fatalf("RetryMultiplier = %f, want %f", normalized.RetryMultiplier, 1.5)
	}
	if normalized.DLQSuffix != ".dead" {
		t.Fatalf("DLQSuffix = %q, want %q", normalized.DLQSuffix, ".dead")
	}
}

// TestNonRetriableHelpers verifies non-retriable marker wrapping and classification.
func TestNonRetriableHelpers(t *testing.T) {
	wrapped := NonRetriable(errors.New("validation"))
	if !IsNonRetriable(wrapped) {
		t.Fatalf("expected wrapped error to be classified as non-retriable")
	}

	nilWrapped := NonRetriable(nil)
	if !IsNonRetriable(nilWrapped) {
		t.Fatalf("expected nil-wrapped error to be classified as non-retriable")
	}
}
