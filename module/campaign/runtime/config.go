package runtime

import "strings"

// Config defines campaign runtime configuration values.
type Config struct {
	// Enabled defines whether campaign module wiring should be active.
	Enabled bool `mapstructure:"CAMPAIGN_ENABLED" default:"true"`
	// SendWorkers defines bounded asynchronous fan-out worker values.
	SendWorkers int `mapstructure:"CAMPAIGN_SEND_WORKERS" default:"8"`
	// SendBatchSize defines send batch-size values used by external orchestrators.
	SendBatchSize int `mapstructure:"CAMPAIGN_SEND_BATCH_SIZE" default:"100"`
	// SendRateLimitPerSecond defines outbound send rate-limit values.
	SendRateLimitPerSecond int `mapstructure:"CAMPAIGN_SEND_RATE_LIMIT_PER_SECOND" default:"10"`
	// UnsubscribeBaseURL defines the public frontend base URL used for unsubscribe links.
	UnsubscribeBaseURL string `mapstructure:"UNSUBSCRIBE_BASE_URL" default:""`
	// MarketingOptOutSecret defines HMAC secret values used to sign unsubscribe tokens.
	MarketingOptOutSecret string `mapstructure:"MARKETING_OPTOUT_SECRET" default:""`
	// MarketingOptOutTokenTTLHours defines unsubscribe token expiration windows in hours.
	MarketingOptOutTokenTTLHours int `mapstructure:"MN_MARKETING_OPTOUT_TOKEN_TTL_HOURS" default:"720"`
}

// ResolvedMarketingOptOutSecret returns the effective opt-out token secret value.
func (c Config) ResolvedMarketingOptOutSecret() string {
	return strings.TrimSpace(c.MarketingOptOutSecret)
}
