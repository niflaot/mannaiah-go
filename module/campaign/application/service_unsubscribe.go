package application

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
	"time"
)

const (
	// defaultUnsubscribeTokenTTL defines default opt-out token expiration windows.
	defaultUnsubscribeTokenTTL = 30 * 24 * time.Hour
	// defaultUnsubscribePath defines unsubscribe path values under public frontend URLs.
	defaultUnsubscribePath = "/public/marketing/optout"
)

// marketingOptOutTokenPayload defines unsubscribe-token payload values.
type marketingOptOutTokenPayload struct {
	// Email defines recipient email values.
	Email string `json:"email"`
	// Name defines optional recipient display name values.
	Name *string `json:"name"`
	// CampaignID defines optional campaign identifier values.
	CampaignID *string `json:"campaignId"`
	// IssuedAt defines unix issuance timestamps.
	IssuedAt int64 `json:"iat"`
	// ExpiresAt defines unix expiration timestamps.
	ExpiresAt int64 `json:"exp"`
}

// cloneTemplateVars returns a shallow copy of campaign template variables.
func cloneTemplateVars(values map[string]string) map[string]string {
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}

	return result
}

// buildUnsubscribeURL builds the personalized unsubscribe URL for one recipient.
func (s *CampaignService) buildUnsubscribeURL(email string, name string, campaignID string) string {
	if s == nil {
		return ""
	}
	baseURL := normalizeUnsubscribeBaseURL(s.unsubscribeBaseURL)
	if baseURL == "" {
		return ""
	}
	secret := strings.TrimSpace(s.unsubscribeTokenSecret)
	if secret == "" {
		return ""
	}
	trimmedEmail := strings.TrimSpace(email)
	if trimmedEmail == "" {
		return ""
	}

	token, err := buildMarketingOptOutToken(marketingOptOutTokenPayload{
		Email:      trimmedEmail,
		Name:       optionalStringPointer(name),
		CampaignID: optionalStringPointer(campaignID),
	}, secret, time.Now().UTC(), s.unsubscribeTokenTTL)
	if err != nil {
		return ""
	}

	return baseURL + defaultUnsubscribePath + "/" + url.PathEscape(token)
}

// buildMarketingOptOutToken builds signed opt-out token values as "<payload>.<signature>".
func buildMarketingOptOutToken(payload marketingOptOutTokenPayload, secret string, now time.Time, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		ttl = defaultUnsubscribeTokenTTL
	}

	payload.IssuedAt = now.UTC().Unix()
	payload.ExpiresAt = now.UTC().Add(ttl).Unix()
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	signature := signMarketingOptOutToken(encodedPayload, secret)

	return encodedPayload + "." + signature, nil
}

// signMarketingOptOutToken signs base64url payload values with HMAC SHA256.
func signMarketingOptOutToken(encodedPayload string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(encodedPayload))

	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// optionalStringPointer returns nil for empty strings and pointer values otherwise.
func optionalStringPointer(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

// normalizeUnsubscribeBaseURL normalizes trailing slashes from public URL values.
func normalizeUnsubscribeBaseURL(value string) string {
	return strings.TrimRight(strings.TrimSpace(value), "/")
}
