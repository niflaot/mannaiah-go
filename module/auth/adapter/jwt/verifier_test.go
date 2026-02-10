package jwt

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

// TestNewVerifierRejectsInvalidConfig verifies constructor validation behavior.
func TestNewVerifierRejectsInvalidConfig(t *testing.T) {
	_, err := NewVerifier(Config{})
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("NewVerifier() error = %v, want ErrInvalidConfig", err)
	}
}

// TestVerifyRS256WithCachedJWKS verifies RS256 verification and JWKS cache behavior.
func TestVerifyRS256WithCachedJWKS(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey() error = %v", err)
	}

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{rsaJWK("rsa-1", &privateKey.PublicKey)}})
	}))
	defer server.Close()

	verifier := newVerifierForTest(t, Config{
		Issuer:             server.URL,
		Audience:           "https://api.mannaiah.test",
		JWKSURL:            server.URL,
		RateLimitPerMinute: 5,
		CacheTTL:           time.Hour,
		HTTPTimeout:        2 * time.Second,
	})

	token := signRS256Token(t, privateKey, "rsa-1", server.URL, "https://api.mannaiah.test", "contacts:read")

	claims, verifyErr := verifier.Verify(context.Background(), token)
	if verifyErr != nil {
		t.Fatalf("Verify() error = %v", verifyErr)
	}
	if claims.Subject != "user-1" {
		t.Fatalf("claims.Subject = %q, want %q", claims.Subject, "user-1")
	}
	if claims.Scope != "contacts:read" {
		t.Fatalf("claims.Scope = %q, want %q", claims.Scope, "contacts:read")
	}

	claims, verifyErr = verifier.Verify(context.Background(), token)
	if verifyErr != nil {
		t.Fatalf("Verify() second error = %v", verifyErr)
	}
	if claims.Subject != "user-1" {
		t.Fatalf("claims.Subject = %q, want %q", claims.Subject, "user-1")
	}
	if calls.Load() < 1 {
		t.Fatalf("jwks calls = %d, want at least one call", calls.Load())
	}
}

// TestVerifyES384 verifies ES384 signature validation behavior.
func TestVerifyES384(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Fatalf("ecdsa.GenerateKey() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{ecJWK("ec-1", privateKey.PublicKey)}})
	}))
	defer server.Close()

	verifier := newVerifierForTest(t, Config{
		Issuer:             server.URL,
		Audience:           "https://api.mannaiah.test",
		JWKSURL:            server.URL,
		RateLimitPerMinute: 5,
		CacheTTL:           time.Hour,
		HTTPTimeout:        2 * time.Second,
	})

	token := signES384Token(t, privateKey, "ec-1", server.URL, "https://api.mannaiah.test", "contacts:update")

	claims, verifyErr := verifier.Verify(context.Background(), token)
	if verifyErr != nil {
		t.Fatalf("Verify() error = %v", verifyErr)
	}
	if claims.Scope != "contacts:update" {
		t.Fatalf("claims.Scope = %q, want %q", claims.Scope, "contacts:update")
	}
}

// TestVerifyRejectsInvalidAudience verifies audience claim validation behavior.
func TestVerifyRejectsInvalidAudience(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{rsaJWK("rsa-1", &privateKey.PublicKey)}})
	}))
	defer server.Close()

	verifier := newVerifierForTest(t, Config{
		Issuer:             server.URL,
		Audience:           "https://api.expected",
		JWKSURL:            server.URL,
		RateLimitPerMinute: 5,
		CacheTTL:           time.Hour,
		HTTPTimeout:        2 * time.Second,
	})

	token := signRS256Token(t, privateKey, "rsa-1", server.URL, "https://api.other", "contacts:read")
	_, verifyErr := verifier.Verify(context.Background(), token)
	if !errors.Is(verifyErr, ErrInvalidToken) {
		t.Fatalf("Verify() error = %v, want ErrInvalidToken", verifyErr)
	}
}

// TestVerifyRateLimitUnknownKID verifies unknown-kid refresh rate-limiting behavior.
func TestVerifyRateLimitUnknownKID(t *testing.T) {
	primaryKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey(primary) error = %v", err)
	}
	secondaryKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey(secondary) error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{rsaJWK("rsa-1", &primaryKey.PublicKey)}})
	}))
	defer server.Close()

	verifier := newVerifierForTest(t, Config{
		Issuer:             server.URL,
		Audience:           "https://api.mannaiah.test",
		JWKSURL:            server.URL,
		RateLimitPerMinute: 1,
		CacheTTL:           time.Hour,
		HTTPTimeout:        50 * time.Millisecond,
	})

	firstToken := signRS256Token(t, primaryKey, "rsa-1", server.URL, "https://api.mannaiah.test", "contacts:read")
	if _, verifyErr := verifier.Verify(context.Background(), firstToken); verifyErr != nil {
		t.Fatalf("Verify() first token error = %v", verifyErr)
	}

	unknownKidToken := signRS256Token(t, secondaryKey, "rsa-2", server.URL, "https://api.mannaiah.test", "contacts:read")
	if _, verifyErr := verifier.Verify(context.Background(), unknownKidToken); !errors.Is(verifyErr, ErrInvalidToken) {
		t.Fatalf("Verify() unknown kid error = %v, want ErrInvalidToken", verifyErr)
	}

	_, verifyErr := verifier.Verify(context.Background(), unknownKidToken)
	if !errors.Is(verifyErr, ErrInvalidToken) {
		t.Fatalf("Verify() unknown kid retry error = %v, want ErrInvalidToken", verifyErr)
	}
	if !strings.Contains(strings.ToLower(verifyErr.Error()), "rate") {
		t.Fatalf("Verify() error = %v, want rate-limit context", verifyErr)
	}
}

// TestMapClaimsHelpers verifies claims helper mapping behavior.
func TestMapClaimsHelpers(t *testing.T) {
	claims := mapClaims(jwtlib.MapClaims{
		"sub":   " user-1 ",
		"iss":   "issuer",
		"aud":   []any{"api", ""},
		"scope": "contacts:read",
	})
	if claims.Subject != "user-1" {
		t.Fatalf("claims.Subject = %q, want %q", claims.Subject, "user-1")
	}
	if len(claims.Audience) != 1 || claims.Audience[0] != "api" {
		t.Fatalf("claims.Audience = %#v, want [api]", claims.Audience)
	}

	audience := readAudienceClaim(123)
	if len(audience) != 0 {
		t.Fatalf("len(audience) = %d, want %d", len(audience), 0)
	}
}

// newVerifierForTest creates a verifier and fails tests on constructor errors.
func newVerifierForTest(t *testing.T, cfg Config) *Verifier {
	t.Helper()

	verifier, err := NewVerifier(cfg)
	if err != nil {
		t.Fatalf("NewVerifier() error = %v", err)
	}

	return verifier
}

// rsaJWK maps RSA public keys to JWK values.
func rsaJWK(kid string, key *rsa.PublicKey) map[string]any {
	return map[string]any{
		"kty": "RSA",
		"kid": kid,
		"n":   encodeBigInt(key.N),
		"e":   encodeBigInt(big.NewInt(int64(key.E))),
	}
}

// ecJWK maps ECDSA public keys to JWK values.
func ecJWK(kid string, key ecdsa.PublicKey) map[string]any {
	return map[string]any{
		"kty": "EC",
		"kid": kid,
		"crv": "P-384",
		"x":   encodeBigInt(key.X),
		"y":   encodeBigInt(key.Y),
	}
}

// encodeBigInt encodes big-int values as base64url strings.
func encodeBigInt(value *big.Int) string {
	if value == nil {
		return ""
	}

	return base64.RawURLEncoding.EncodeToString(value.Bytes())
}

// signRS256Token signs RS256 JWT tokens for verifier tests.
func signRS256Token(t *testing.T, key *rsa.PrivateKey, kid string, issuer string, audience string, scope string) string {
	t.Helper()

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, jwtlib.MapClaims{
		"sub":   "user-1",
		"iss":   issuer,
		"aud":   audience,
		"scope": scope,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	token.Header["kid"] = kid

	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}

	return signed
}

// signES384Token signs ES384 JWT tokens for verifier tests.
func signES384Token(t *testing.T, key *ecdsa.PrivateKey, kid string, issuer string, audience string, scope string) string {
	t.Helper()

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodES384, jwtlib.MapClaims{
		"sub":   "user-1",
		"iss":   issuer,
		"aud":   audience,
		"scope": scope,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	token.Header["kid"] = kid

	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}

	return signed
}
