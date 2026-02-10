package domain

import "testing"

// TestClaimsScopes verifies scope parsing behavior.
func TestClaimsScopes(t *testing.T) {
	claims := &Claims{Scope: "contacts:read  contacts:create"}
	scopes := claims.Scopes()
	if len(scopes) != 2 {
		t.Fatalf("len(scopes) = %d, want %d", len(scopes), 2)
	}
	if scopes[0] != "contacts:read" || scopes[1] != "contacts:create" {
		t.Fatalf("scopes = %#v", scopes)
	}
}

// TestClaimsScopesNil verifies nil-claims behavior.
func TestClaimsScopesNil(t *testing.T) {
	var claims *Claims
	scopes := claims.Scopes()
	if len(scopes) != 0 {
		t.Fatalf("len(scopes) = %d, want %d", len(scopes), 0)
	}
}
