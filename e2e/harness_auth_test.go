package e2e_test

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

// SignToken creates a signed JWT token for E2E requests.
func (h *contactsE2EHarness) SignToken(t *testing.T, scopes string) string {
	t.Helper()

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, jwtlib.MapClaims{
		"sub":   "e2e-user",
		"iss":   strings.TrimSuffix(h.jwksServer.URL, e2eIssuerSuffix),
		"aud":   e2eAudience,
		"scope": scopes,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(10 * time.Minute).Unix(),
	})
	token.Header["kid"] = e2eTokenKid

	signed, err := token.SignedString(h.key)
	if err != nil {
		t.Fatalf("token.SignedString() error = %v", err)
	}

	return signed
}

// newJWKSServer creates an HTTP JWKS endpoint server for token verification tests.
func newJWKSServer(t *testing.T, publicKey rsa.PublicKey) *httptest.Server {
	t.Helper()

	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != e2eIssuerSuffix {
			writer.WriteHeader(http.StatusNotFound)
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"keys": []any{map[string]any{
				"kty": "RSA",
				"kid": e2eTokenKid,
				"alg": "RS256",
				"use": "sig",
				"n":   encodeBigInt(publicKey.N),
				"e":   encodeBigInt(big.NewInt(int64(publicKey.E))),
			}},
		})
	})

	return httptest.NewServer(handler)
}

// encodeBigInt encodes big-int values into base64url strings.
func encodeBigInt(value *big.Int) string {
	if value == nil {
		return ""
	}

	return base64.RawURLEncoding.EncodeToString(value.Bytes())
}
