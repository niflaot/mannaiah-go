package runtime

// Config defines shipping runtime configuration values.
type Config struct {
	// Enabled defines whether shipping module wiring should be active.
	Enabled bool `mapstructure:"SHIPPING_ENABLED" default:"true"`
	// TrackingCacheTTLSeconds defines tracking cache TTL values in seconds.
	TrackingCacheTTLSeconds int `mapstructure:"SHIPPING_TRACKING_CACHE_TTL_SECONDS" default:"300"`
	// TCC defines TCC carrier configuration values.
	TCC TCCConfig `mapstructure:",squash"`
	// DefaultSender defines fallback sender information used for mark generation.
	DefaultSender DefaultSenderConfig `mapstructure:",squash"`
}

// TCCConfig defines TCC carrier configuration values.
type TCCConfig struct {
	// Enabled defines whether TCC provider wiring should be active.
	Enabled bool `mapstructure:"SHIPPING_TCC_ENABLED" default:"false"`
	// BaseURL defines TCC API base URL values.
	BaseURL string `mapstructure:"SHIPPING_TCC_BASE_URL" default:""`
	// AccessToken defines TCC access-token values.
	AccessToken string `mapstructure:"SHIPPING_TCC_ACCESS_TOKEN" default:""`
	// AccountNumber defines TCC account number values.
	AccountNumber string `mapstructure:"SHIPPING_TCC_ACCOUNT_NUMBER" default:""`
	// BusinessUnit defines TCC business-unit values.
	BusinessUnit int `mapstructure:"SHIPPING_TCC_BUSINESS_UNIT" default:"1"`
	// PaymentForm defines TCC payment-form values.
	PaymentForm int `mapstructure:"SHIPPING_TCC_PAYMENT_FORM" default:"1"`
	// RequestTimeoutMS defines outbound TCC request timeout values in milliseconds.
	RequestTimeoutMS int `mapstructure:"SHIPPING_TCC_REQUEST_TIMEOUT_MS" default:"10000"`
}

// DefaultSenderConfig defines fallback sender information for mark generation.
type DefaultSenderConfig struct {
	// Name defines sender name values.
	Name string `mapstructure:"SHIPPING_DEFAULT_SENDER_NAME" default:""`
	// ID defines sender identification values.
	ID string `mapstructure:"SHIPPING_DEFAULT_SENDER_ID" default:""`
	// IDType defines sender identification-type values.
	IDType string `mapstructure:"SHIPPING_DEFAULT_SENDER_ID_TYPE" default:"NIT"`
	// Address defines sender address values.
	Address string `mapstructure:"SHIPPING_DEFAULT_SENDER_ADDRESS" default:""`
	// CityCode defines sender city-code values.
	CityCode string `mapstructure:"SHIPPING_DEFAULT_SENDER_CITY_CODE" default:""`
	// Phone defines sender phone values.
	Phone string `mapstructure:"SHIPPING_DEFAULT_SENDER_PHONE" default:""`
	// Email defines sender email values.
	Email string `mapstructure:"SHIPPING_DEFAULT_SENDER_EMAIL" default:""`
}
