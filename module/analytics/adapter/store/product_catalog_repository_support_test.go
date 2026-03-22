package store

import "testing"

// TestParseScopedURLAttributeKey verifies variation-scoped URL attribute key parsing behavior.
func TestParseScopedURLAttributeKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		key       string
		wantScope string
		wantOK    bool
	}{
		{name: "valid", key: "var-1.url", wantScope: "var-1", wantOK: true},
		{name: "valid uppercase suffix", key: "VAR-2.URL", wantScope: "var-2", wantOK: true},
		{name: "invalid suffix", key: "var-1.link", wantOK: false},
		{name: "missing scope", key: ".url", wantOK: false},
		{name: "missing separator", key: "url", wantOK: false},
		{name: "multi-dot suffix", key: "var-1.url.extra", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scope, ok := parseScopedURLAttributeKey(tt.key)
			if ok != tt.wantOK {
				t.Fatalf("parseScopedURLAttributeKey(%q) ok = %v, want %v", tt.key, ok, tt.wantOK)
			}
			if ok && scope != tt.wantScope {
				t.Fatalf("parseScopedURLAttributeKey(%q) scope = %q, want %q", tt.key, scope, tt.wantScope)
			}
		})
	}
}
