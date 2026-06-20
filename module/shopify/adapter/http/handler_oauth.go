package http

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	corehttp "mannaiah/module/core/http"
	shopifyport "mannaiah/module/shopify/port"
)

const (
	oauthStateTTL      = 10 * time.Minute
	shopifyOAuthScopes = "read_orders,write_orders,read_customers,write_customers,read_products,read_metaobjects"
)

var (
	// ErrOAuthUnavailable is returned when Shopify OAuth dependencies are unavailable.
	ErrOAuthUnavailable = errors.New("shopify oauth is unavailable")
	// ErrOAuthCodeRequired is returned when Shopify OAuth callback codes are missing.
	ErrOAuthCodeRequired = errors.New("shopify oauth code is required")
	// ErrOAuthStateInvalid is returned when Shopify OAuth callback state values are invalid.
	ErrOAuthStateInvalid = errors.New("shopify oauth state is invalid")
	// ErrOAuthStateExpired is returned when Shopify OAuth callback state values are expired.
	ErrOAuthStateExpired = errors.New("shopify oauth state is expired")
	// ErrOAuthHMACInvalid is returned when Shopify OAuth callback signatures are invalid.
	ErrOAuthHMACInvalid = errors.New("shopify oauth callback signature is invalid")
	// ErrPublicBaseURLRequired is returned when externally reachable base URLs cannot be resolved.
	ErrPublicBaseURLRequired = errors.New("shopify public base url is required")
)

// newSignedState builds a stateless OAuth state token: "{timestamp}.{HMAC(shopDomain|timestamp, secret)}".
// No server-side storage needed — the state self-verifies on callback using the client secret.
func newSignedState(shopDomain string, secret string, now time.Time) (string, error) {
	ts := strconv.FormatInt(now.UTC().Unix(), 10)
	payload := shopifyport.NormalizeShopDomain(shopDomain) + "|" + ts
	mac := hmac.New(sha256.New, []byte(strings.TrimSpace(secret)))
	if _, err := rand.Read(make([]byte, 0)); err != nil {
		return "", err
	}
	_, _ = mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))
	return ts + "." + sig, nil
}

// verifySignedState validates a state token produced by newSignedState.
func verifySignedState(state string, shopDomain string, secret string, now time.Time, maxAge time.Duration) error {
	parts := strings.SplitN(strings.TrimSpace(state), ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return ErrOAuthStateInvalid
	}
	ts, sig := parts[0], parts[1]
	seconds, err := strconv.ParseInt(ts, 10, 64)
	if err != nil || seconds <= 0 {
		return ErrOAuthStateInvalid
	}
	issued := time.Unix(seconds, 0).UTC()
	if now.Sub(issued) > maxAge || issued.After(now.Add(maxAge)) {
		return ErrOAuthStateExpired
	}
	payload := shopifyport.NormalizeShopDomain(shopDomain) + "|" + ts
	mac := hmac.New(sha256.New, []byte(strings.TrimSpace(secret)))
	_, _ = mac.Write([]byte(payload))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(strings.ToLower(sig))) {
		return ErrOAuthStateInvalid
	}
	return nil
}

func (h *Handler) installOAuth(ctx corehttp.Context) error {
	if h == nil || strings.TrimSpace(h.clientID) == "" || strings.TrimSpace(h.clientSecret) == "" || h.oauthClient == nil {
		return h.mapError(ErrOAuthUnavailable)
	}

	shopDomain := shopifyport.NormalizeShopDomain(ctx.Query("shop", ""))
	if !isValidShopDomain(shopDomain) {
		return h.mapError(ErrInvalidShopDomain)
	}

	baseURL, err := resolveExternalBaseURL(ctx)
	if err != nil {
		return h.mapError(err)
	}
	state, err := newSignedState(shopDomain, h.clientSecret, time.Now().UTC())
	if err != nil {
		return h.mapError(err)
	}

	redirectURL, err := buildOAuthInstallURL(shopDomain, h.clientID, baseURL+"/shopify/oauth/callback", state)
	if err != nil {
		return h.mapError(err)
	}

	ctx.SetHeader("Location", redirectURL)
	return ctx.Status(302).SendString("")
}

func (h *Handler) oauthCallback(ctx corehttp.Context) error {
	if h == nil || strings.TrimSpace(h.clientSecret) == "" || h.oauthClient == nil {
		return h.mapError(ErrOAuthUnavailable)
	}

	params := ctx.Queries()
	if !VerifyOAuthCallbackSignature(params, h.clientSecret) {
		return h.mapError(ErrOAuthHMACInvalid)
	}

	shopDomain := shopifyport.NormalizeShopDomain(params["shop"])
	if !isValidShopDomain(shopDomain) {
		return h.mapError(ErrInvalidShopDomain)
	}
	if err := verifyOAuthTimestamp(params["timestamp"], time.Now().UTC(), oauthStateTTL); err != nil {
		return h.mapError(err)
	}
	code := strings.TrimSpace(params["code"])
	if code == "" {
		return h.mapError(ErrOAuthCodeRequired)
	}
	state := strings.TrimSpace(params["state"])
	if err := verifySignedState(state, shopDomain, h.clientSecret, time.Now().UTC(), oauthStateTTL); err != nil {
		return h.mapError(err)
	}

	accessToken, scopes, err := h.oauthClient.ExchangeAuthorizationCode(ctx.Context(), shopDomain, code)
	if err != nil {
		return h.mapError(err)
	}
	installation, err := h.installations.UpsertInstallation(ctx.Context(), shopifyport.UpsertInstallationInput{
		ShopDomain:  shopDomain,
		AccessToken: accessToken,
		Scopes:      scopes,
		InstalledAt: time.Now().UTC(),
	})
	if err != nil {
		return h.mapError(err)
	}
	if h.installationResolver != nil {
		if err := h.installationResolver.Refresh(ctx.Context()); err != nil {
			return h.mapError(err)
		}
	}

	baseURL, err := resolveExternalBaseURL(ctx)
	if err != nil {
		return h.mapError(err)
	}
	if err := h.oauthClient.RegisterWebhooks(ctx.Context(), shopDomain, accessToken, baseURL+"/shopify/webhooks"); err != nil {
		return h.mapError(err)
	}

	payload := map[string]any{
		"shopDomain":         installation.ShopDomain,
		"scopes":             installation.Scopes,
		"installedAt":        installation.InstalledAt,
		"webhooksRegistered": true,
	}
	if prefersJSONOAuthCallbackResponse(ctx.GetHeader("Accept", "")) {
		return ctx.Status(200).JSON(payload)
	}

	launchURL, err := buildAppLaunchURL(baseURL, installation.ShopDomain, true)
	if err != nil {
		return h.mapError(err)
	}
	ctx.SetHeader("Location", launchURL)
	return ctx.Status(302).SendString("")
}

// VerifyOAuthCallbackSignature verifies the callback query-string HMAC using the Shopify client secret.
func VerifyOAuthCallbackSignature(params map[string]string, secret string) bool {
	received := strings.TrimSpace(params["hmac"])
	if received == "" || strings.TrimSpace(secret) == "" {
		return false
	}

	message := buildOAuthSignatureMessage(params)
	mac := hmac.New(sha256.New, []byte(strings.TrimSpace(secret)))
	_, _ = mac.Write([]byte(message))
	computed := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(strings.ToLower(computed)), []byte(strings.ToLower(received)))
}

func buildOAuthSignatureMessage(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for key, value := range params {
		if strings.EqualFold(key, "hmac") || strings.EqualFold(key, "signature") {
			continue
		}
		if strings.TrimSpace(value) == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, params[key]))
	}

	return strings.Join(parts, "&")
}

func verifyOAuthTimestamp(value string, now time.Time, maxAge time.Duration) error {
	seconds, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || seconds <= 0 {
		return ErrOAuthStateInvalid
	}
	timestamp := time.Unix(seconds, 0).UTC()
	if timestamp.Before(now.Add(-maxAge)) || timestamp.After(now.Add(maxAge)) {
		return ErrOAuthStateExpired
	}

	return nil
}

func buildOAuthInstallURL(shopDomain string, clientID string, redirectURI string, state string) (string, error) {
	if !isValidShopDomain(shopDomain) || strings.TrimSpace(clientID) == "" || strings.TrimSpace(redirectURI) == "" || strings.TrimSpace(state) == "" {
		return "", ErrOAuthStateInvalid
	}

	endpoint := url.URL{
		Scheme: "https",
		Host:   shopDomain,
		Path:   "/admin/oauth/authorize",
	}
	query := endpoint.Query()
	query.Set("client_id", strings.TrimSpace(clientID))
	query.Set("scope", shopifyOAuthScopes)
	query.Set("redirect_uri", strings.TrimSpace(redirectURI))
	query.Set("state", strings.TrimSpace(state))
	endpoint.RawQuery = query.Encode()
	return endpoint.String(), nil
}

func buildAppLaunchURL(baseURL string, shopDomain string, installed bool) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", ErrPublicBaseURLRequired
	}

	endpoint := url.URL{
		Scheme: parsed.Scheme,
		Host:   parsed.Host,
		Path:   "/shopify/app",
	}
	query := endpoint.Query()
	if normalizedShopDomain := shopifyport.NormalizeShopDomain(shopDomain); isValidShopDomain(normalizedShopDomain) {
		query.Set("shop", normalizedShopDomain)
	}
	if installed {
		query.Set("installed", "1")
	}
	endpoint.RawQuery = query.Encode()

	return endpoint.String(), nil
}

func prefersJSONOAuthCallbackResponse(acceptHeader string) bool {
	acceptHeader = strings.ToLower(strings.TrimSpace(acceptHeader))
	if acceptHeader == "" {
		return false
	}

	return strings.Contains(acceptHeader, "application/json") && !strings.Contains(acceptHeader, "text/html")
}

func resolveExternalBaseURL(ctx corehttp.Context) (string, error) {
	host := strings.TrimSpace(ctx.GetHeader("X-Forwarded-Host", ""))
	if host == "" {
		host = strings.TrimSpace(ctx.GetHeader("Host", ""))
	}
	host = strings.TrimSpace(strings.Split(host, ",")[0])
	if host == "" {
		return "", ErrPublicBaseURLRequired
	}

	proto := strings.TrimSpace(ctx.GetHeader("X-Forwarded-Proto", ""))
	if proto == "" {
		proto = "https"
	}

	return proto + "://" + host, nil
}

func computeStateSignature(payload string, secret string) string {
	mac := hmac.New(sha256.New, []byte(strings.TrimSpace(secret)))
	_, _ = mac.Write([]byte(strings.TrimSpace(payload)))
	return hex.EncodeToString(mac.Sum(nil))
}
