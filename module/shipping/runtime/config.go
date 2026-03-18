package runtime

// Config defines shipping integration configuration values.
type Config struct {
	// Enabled defines whether shipping quote behavior is enabled.
	Enabled bool `mapstructure:"SHIPPING_ENABLED" default:"true"`
	// TCCBaseURL defines TCC base URL values.
	TCCBaseURL string `mapstructure:"SHIPPING_TCC_BASE_URL" default:"https://testsomos.tcc.com.co"`
	// TCCAccessToken defines TCC access token values.
	TCCAccessToken string `mapstructure:"SHIPPING_TCC_ACCESS_TOKEN" default:""`
	// TCCAccount defines TCC quote account values.
	TCCAccount string `mapstructure:"SHIPPING_TCC_ACCOUNT" default:"7000880"`
	// TCCIdentifier defines TCC quote identifier values.
	TCCIdentifier string `mapstructure:"SHIPPING_TCC_IDENTIFIER" default:""`
	// TCCLegalName defines TCC legal-name values.
	TCCLegalName string `mapstructure:"SHIPPING_TCC_LEGAL_NAME" default:""`
	// TCCRequestTimeoutMS defines TCC request timeout values in milliseconds.
	TCCRequestTimeoutMS int `mapstructure:"SHIPPING_TCC_REQUEST_TIMEOUT_MS" default:"5000"`
	// TCCCircuitBreakerEnabled defines whether TCC quote requests are circuit-breaker protected.
	TCCCircuitBreakerEnabled bool `mapstructure:"SHIPPING_TCC_CIRCUIT_BREAKER_ENABLED" default:"true"`
	// TCCCircuitBreakerMaxRequests defines half-open max concurrent requests.
	TCCCircuitBreakerMaxRequests uint32 `mapstructure:"SHIPPING_TCC_CIRCUIT_BREAKER_MAX_REQUESTS" default:"1"`
	// TCCCircuitBreakerIntervalMS defines closed-state counter reset intervals in milliseconds.
	TCCCircuitBreakerIntervalMS int `mapstructure:"SHIPPING_TCC_CIRCUIT_BREAKER_INTERVAL_MS" default:"60000"`
	// TCCCircuitBreakerTimeoutMS defines open-state timeout windows in milliseconds.
	TCCCircuitBreakerTimeoutMS int `mapstructure:"SHIPPING_TCC_CIRCUIT_BREAKER_TIMEOUT_MS" default:"30000"`
	// TCCCircuitBreakerFailureThreshold defines consecutive failure count that opens the breaker.
	TCCCircuitBreakerFailureThreshold uint32 `mapstructure:"SHIPPING_TCC_CIRCUIT_BREAKER_FAILURE_THRESHOLD" default:"3"`
}
