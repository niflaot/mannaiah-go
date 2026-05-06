package http

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"

	shopifyport "mannaiah/module/shopify/port"
)

var (
	// ErrSessionTokenMissing is returned when extension requests do not provide bearer tokens.
	ErrSessionTokenMissing = errors.New("shopify session token is required")
	// ErrSessionTokenInvalid is returned when extension session tokens cannot be verified.
	ErrSessionTokenInvalid = errors.New("shopify session token is invalid")
	// ErrSessionTokenExpired is returned when extension session tokens are expired.
	ErrSessionTokenExpired = errors.New("shopify session token is expired")
	// ErrSessionTokenNotYetValid is returned when extension session tokens are not yet valid.
	ErrSessionTokenNotYetValid = errors.New("shopify session token is not yet valid")
)

type sessionTokenHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type sessionTokenClaims struct {
	Iss string `json:"iss"`
	Dest string `json:"dest"`
	Exp int64 `json:"exp"`
	Nbf int64 `json:"nbf"`
}

func authenticateSessionToken(ctx context.Context, authorizationHeader string, clientSecret string, resolver shopifyport.InstallationResolver) (string, error) {
	rawToken := extractBearerToken(authorizationHeader)
	if rawToken == "" {
		return "", ErrSessionTokenMissing
	}
	claims, err := verifySessionToken(rawToken, clientSecret, time.Now().UTC())
	if err != nil {
		return "", err
	}
	shopDomain, err := resolveSessionShopDomain(claims)
	if err != nil {
		return "", err
	}
	if resolver == nil {
		return "", ErrOAuthUnavailable
	}
	installation, err := resolver.ResolveInstallation(ctx, shopDomain)
	if err != nil || installation == nil {
		return "", ErrSessionTokenInvalid
	}

	return shopDomain, nil
}

func verifySessionToken(rawToken string, clientSecret string, now time.Time) (sessionTokenClaims, error) {
	parts := strings.Split(strings.TrimSpace(rawToken), ".")
	if len(parts) != 3 {
		return sessionTokenClaims{}, ErrSessionTokenInvalid
	}

	var header sessionTokenHeader
	if err := decodeJWTPart(parts[0], &header); err != nil {
		return sessionTokenClaims{}, ErrSessionTokenInvalid
	}
	if !strings.EqualFold(strings.TrimSpace(header.Alg), "HS256") {
		return sessionTokenClaims{}, ErrSessionTokenInvalid
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return sessionTokenClaims{}, ErrSessionTokenInvalid
	}
	mac := hmac.New(sha256.New, []byte(strings.TrimSpace(clientSecret)))
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return sessionTokenClaims{}, ErrSessionTokenInvalid
	}

	var claims sessionTokenClaims
	if err := decodeJWTPart(parts[1], &claims); err != nil {
		return sessionTokenClaims{}, ErrSessionTokenInvalid
	}
	if claims.Exp > 0 && now.Unix() >= claims.Exp {
		return sessionTokenClaims{}, ErrSessionTokenExpired
	}
	if claims.Nbf > 0 && now.Unix() < claims.Nbf {
		return sessionTokenClaims{}, ErrSessionTokenNotYetValid
	}

	return claims, nil
}

func resolveSessionShopDomain(claims sessionTokenClaims) (string, error) {
	destURL, err := url.Parse(strings.TrimSpace(claims.Dest))
	if err != nil {
		return "", ErrSessionTokenInvalid
	}
	shopDomain := shopifyport.NormalizeShopDomain(destURL.Host)
	if !isValidShopDomain(shopDomain) {
		return "", ErrSessionTokenInvalid
	}
	issURL, err := url.Parse(strings.TrimSpace(claims.Iss))
	if err != nil {
		return "", ErrSessionTokenInvalid
	}
	if shopifyport.NormalizeShopDomain(issURL.Host) != shopDomain || !strings.HasPrefix(strings.TrimSpace(issURL.Path), "/admin") {
		return "", ErrSessionTokenInvalid
	}

	return shopDomain, nil
}

func decodeJWTPart(value string, output any) error {
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return err
	}

	return json.Unmarshal(decoded, output)
}

func extractBearerToken(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	parts := strings.SplitN(trimmed, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	return strings.TrimSpace(parts[1])
}