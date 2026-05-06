package http

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	corehttp "mannaiah/module/core/http"
	shopifyport "mannaiah/module/shopify/port"
)

const (
	oauthStateCookieName = "shopify_oauth_state"
	oauthStateTTL        = 10 * time.Minute
	shopifyOAuthScopes   = "read_orders,write_orders,read_customers,write_customers"
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

type oauthStateCookie struct {
	ShopDomain string
	State      string
	ExpiresAt  time.Time
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
	state, err := newOAuthState()
	if err != nil {
		return h.mapError(err)
	}
	expiresAt := time.Now().UTC().Add(oauthStateTTL)
	if err := writeOAuthStateCookie(ctx, oauthStateCookie{
		ShopDomain: shopDomain,
		State:      state,
		ExpiresAt:  expiresAt,
	}, h.clientSecret); err != nil {
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
	storedState, err := readOAuthStateCookie(ctx, h.clientSecret)
	if err != nil {
		return h.mapError(err)
	}
	if storedState.State != state || storedState.ShopDomain != shopDomain {
		return h.mapError(ErrOAuthStateInvalid)
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

	clearOAuthStateCookie(ctx)
	return ctx.Status(200).JSON(map[string]any{
		"shopDomain": installation.ShopDomain,
		"scopes": installation.Scopes,
		"installedAt": installation.InstalledAt,
		"webhooksRegistered": true,
	})
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

func newOAuthState() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

func writeOAuthStateCookie(ctx corehttp.Context, state oauthStateCookie, secret string) error {
	value, err := signOAuthStateCookie(state, secret)
	if err != nil {
		return err
	}
	ctx.SetHeader("Set-Cookie", (&http.Cookie{
		Name:     oauthStateCookieName,
		Value:    value,
		Path:     "/shopify/oauth",
		MaxAge:   int(time.Until(state.ExpiresAt).Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}).String())

	return nil
}

func clearOAuthStateCookie(ctx corehttp.Context) {
	ctx.SetHeader("Set-Cookie", (&http.Cookie{
		Name:     oauthStateCookieName,
		Value:    "",
		Path:     "/shopify/oauth",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}).String())
}

func readOAuthStateCookie(ctx corehttp.Context, secret string) (oauthStateCookie, error) {
	request := &http.Request{Header: http.Header{"Cookie": []string{ctx.GetHeader("Cookie", "")}}}
	cookie, err := request.Cookie(oauthStateCookieName)
	if err != nil {
		return oauthStateCookie{}, ErrOAuthStateInvalid
	}

	return parseOAuthStateCookie(cookie.Value, secret)
}

func signOAuthStateCookie(state oauthStateCookie, secret string) (string, error) {
	if !isValidShopDomain(state.ShopDomain) || strings.TrimSpace(state.State) == "" || state.ExpiresAt.IsZero() {
		return "", ErrOAuthStateInvalid
	}
	payload := strings.Join([]string{
		shopifyport.NormalizeShopDomain(state.ShopDomain),
		strings.TrimSpace(state.State),
		strconv.FormatInt(state.ExpiresAt.UTC().Unix(), 10),
	}, "|")
	signature := computeStateSignature(payload, secret)
	return base64.RawURLEncoding.EncodeToString([]byte(payload + "." + signature)), nil
}

func parseOAuthStateCookie(value string, secret string) (oauthStateCookie, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return oauthStateCookie{}, ErrOAuthStateInvalid
	}
	parts := strings.SplitN(string(decoded), ".", 2)
	if len(parts) != 2 {
		return oauthStateCookie{}, ErrOAuthStateInvalid
	}
	payload := strings.TrimSpace(parts[0])
	received := strings.TrimSpace(parts[1])
	if payload == "" || received == "" {
		return oauthStateCookie{}, ErrOAuthStateInvalid
	}
	expected := computeStateSignature(payload, secret)
	if !hmac.Equal([]byte(expected), []byte(received)) {
		return oauthStateCookie{}, ErrOAuthStateInvalid
	}
	values := strings.Split(payload, "|")
	if len(values) != 3 {
		return oauthStateCookie{}, ErrOAuthStateInvalid
	}
	expiresUnix, err := strconv.ParseInt(values[2], 10, 64)
	if err != nil {
		return oauthStateCookie{}, ErrOAuthStateInvalid
	}
	state := oauthStateCookie{
		ShopDomain: shopifyport.NormalizeShopDomain(values[0]),
		State:      strings.TrimSpace(values[1]),
		ExpiresAt:  time.Unix(expiresUnix, 0).UTC(),
	}
	if state.ExpiresAt.Before(time.Now().UTC()) {
		return oauthStateCookie{}, ErrOAuthStateExpired
	}

	return state, nil
}

func computeStateSignature(payload string, secret string) string {
	mac := hmac.New(sha256.New, []byte(strings.TrimSpace(secret)))
	_, _ = mac.Write([]byte(strings.TrimSpace(payload)))
	return hex.EncodeToString(mac.Sum(nil))
}