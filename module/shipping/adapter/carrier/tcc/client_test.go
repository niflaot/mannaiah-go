package tcc

import "testing"

// TestNewClient validates required constructor fields.
func TestNewClient(t *testing.T) {
	if _, err := NewClient(ClientConfig{}); err == nil {
		t.Fatalf("expected NewClient() error")
	}
	if _, err := NewClient(ClientConfig{BaseURL: "https://example.com", AccessToken: "token"}); err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
}
