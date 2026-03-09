package runtime

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

// TestResolveContext verifies context resolution behavior.
func TestResolveContext(t *testing.T) {
	if resolveContext(context.Background()) == nil {
		t.Fatalf("resolveContext(background) should not be nil")
	}
	var nilCtx context.Context
	if resolveContext(nilCtx) == nil {
		t.Fatalf("resolveContext(nil) should not be nil")
	}
}

// TestResolveLogger verifies logger resolution behavior.
func TestResolveLogger(t *testing.T) {
	if resolveLogger(nil) == nil {
		t.Fatalf("resolveLogger(nil) should not be nil")
	}
	logger := zap.NewNop()
	if resolveLogger(logger) != logger {
		t.Fatalf("resolveLogger(logger) should return same instance")
	}
}

// TestResolveTimeouts verifies timeout fallback behavior.
func TestResolveTimeouts(t *testing.T) {
	if resolveValidationTimeout(0) != 3*time.Second {
		t.Fatalf("resolveValidationTimeout(0) should use fallback")
	}
	if resolveRequestTimeout(0) != 5*time.Second {
		t.Fatalf("resolveRequestTimeout(0) should use fallback")
	}
	if resolveImageTranscodeTimeout(0) != 15*time.Second {
		t.Fatalf("resolveImageTranscodeTimeout(0) should use fallback")
	}
}

// TestResolveImageTranscodeAllowedPrefixes verifies image-transcode allowed-prefix resolution behavior.
func TestResolveImageTranscodeAllowedPrefixes(t *testing.T) {
	prefixes := resolveImageTranscodeAllowedPrefixes(Config{
		ProductImageTranscodeAllowedPrefixes: " https://cdn.example.com/assets/ , https://img.example.com ",
	})
	if len(prefixes) != 2 {
		t.Fatalf("len(prefixes) = %d, want 2", len(prefixes))
	}
	if prefixes[0] != "https://cdn.example.com/assets" {
		t.Fatalf("prefixes[0] = %q, want %q", prefixes[0], "https://cdn.example.com/assets")
	}
	if prefixes[1] != "https://img.example.com" {
		t.Fatalf("prefixes[1] = %q, want %q", prefixes[1], "https://img.example.com")
	}

	fallback := resolveImageTranscodeAllowedPrefixes(Config{ProductImageBaseURL: "https://cdn.example.com/"})
	if len(fallback) != 1 || fallback[0] != "https://cdn.example.com" {
		t.Fatalf("fallback prefixes = %#v, want [https://cdn.example.com]", fallback)
	}
}

// TestResolveImageTranscodeConfig verifies image-transcode runtime config mapping behavior.
func TestResolveImageTranscodeConfig(t *testing.T) {
	resolved := resolveImageTranscodeConfig(Config{
		ProductImageTranscodeEnabled:         true,
		ProductImageTranscodeAllowedPrefixes: "https://cdn.example.com",
		ProductImageTranscodeTimeoutMS:       7000,
	})
	if !resolved.Enabled {
		t.Fatalf("resolved.Enabled = false, want true")
	}
	if len(resolved.AllowedSourcePrefixes) != 1 || resolved.AllowedSourcePrefixes[0] != "https://cdn.example.com" {
		t.Fatalf("resolved.AllowedSourcePrefixes = %#v, want [https://cdn.example.com]", resolved.AllowedSourcePrefixes)
	}
	if resolved.RequestTimeout != 7*time.Second {
		t.Fatalf("resolved.RequestTimeout = %s, want %s", resolved.RequestTimeout, 7*time.Second)
	}
}
