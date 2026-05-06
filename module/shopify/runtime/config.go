package runtime

// Config defines Shopify integration configuration values.
type Config struct {
	// ClientID defines Shopify OAuth client identifier values.
	ClientID string `mapstructure:"SHOPIFY_CLIENT_ID" default:""`
	// ClientSecret defines Shopify OAuth client secret values and webhook/session HMAC verification keys.
	ClientSecret string `mapstructure:"SHOPIFY_CLIENT_SECRET" default:""`
	// SyncOrders enables order sync behavior.
	SyncOrders bool `mapstructure:"SHOPIFY_SYNC_ORDERS" default:"false"`
	// SyncContacts enables contact sync behavior.
	SyncContacts bool `mapstructure:"SHOPIFY_SYNC_CONTACTS" default:"false"`
	// SyncWorkers defines webhook worker counts.
	SyncWorkers int `mapstructure:"SHOPIFY_SYNC_WORKERS" default:"4"`
	// SyncTimeoutMS defines sync timeout values in milliseconds.
	SyncTimeoutMS int `mapstructure:"SHOPIFY_SYNC_TIMEOUT_MS" default:"600000"`
	// RequestTimeoutMS defines Shopify API timeout values in milliseconds.
	RequestTimeoutMS int `mapstructure:"SHOPIFY_REQUEST_TIMEOUT_MS" default:"10000"`
	// AdminRateLimitIntervalMS defines minimum spacing between Shopify Admin API calls.
	AdminRateLimitIntervalMS int `mapstructure:"SHOPIFY_ADMIN_RATE_LIMIT_INTERVAL_MS" default:"600"`
	// TooManyRequestsRetryDelayMS defines fallback wait time after Shopify 429 responses.
	TooManyRequestsRetryDelayMS int `mapstructure:"SHOPIFY_429_RETRY_DELAY_MS" default:"1100"`
	// CircuitBreakerEnabled defines whether Shopify source/destination requests use circuit breakers.
	CircuitBreakerEnabled bool `mapstructure:"SHOPIFY_CIRCUIT_BREAKER_ENABLED" default:"true"`
	// CircuitBreakerMaxRequests defines half-open max concurrent requests.
	CircuitBreakerMaxRequests uint32 `mapstructure:"SHOPIFY_CIRCUIT_BREAKER_MAX_REQUESTS" default:"1"`
	// CircuitBreakerIntervalMS defines closed-state counter reset intervals in milliseconds.
	CircuitBreakerIntervalMS int `mapstructure:"SHOPIFY_CIRCUIT_BREAKER_INTERVAL_MS" default:"60000"`
	// CircuitBreakerTimeoutMS defines open-state timeout windows in milliseconds.
	CircuitBreakerTimeoutMS int `mapstructure:"SHOPIFY_CIRCUIT_BREAKER_TIMEOUT_MS" default:"30000"`
	// CircuitBreakerFailureThreshold defines consecutive failure count that opens Shopify breakers.
	CircuitBreakerFailureThreshold uint32 `mapstructure:"SHOPIFY_CIRCUIT_BREAKER_FAILURE_THRESHOLD" default:"3"`
}
