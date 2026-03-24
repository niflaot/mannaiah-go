package runtime

// Config defines shipping runtime configuration values.
type Config struct {
	// Enabled defines whether shipping module wiring should be active.
	Enabled bool `mapstructure:"SHIPPING_ENABLED" default:"true"`
	// TrackingCacheTTLSeconds defines tracking cache TTL values in seconds.
	TrackingCacheTTLSeconds int `mapstructure:"SHIPPING_TRACKING_CACHE_TTL_SECONDS" default:"300"`
	// Quotation defines quotation behavior configuration values.
	Quotation QuotationConfig `mapstructure:",squash"`
	// TCC defines TCC carrier configuration values.
	TCC TCCConfig `mapstructure:",squash"`
	// DefaultSender defines fallback sender information used for mark generation.
	DefaultSender DefaultSenderConfig `mapstructure:",squash"`
}

// QuotationConfig defines quotation behavior configuration values.
type QuotationConfig struct {
	// DiscountPercent defines the freight discount percentage applied to carrier quotations.
	DiscountPercent float64 `mapstructure:"SHIPPING_QUOTATION_DISCOUNT_PERCENT" default:"0"`
}

// TCCConfig defines TCC carrier configuration values.
type TCCConfig struct {
	// Enabled defines whether TCC provider wiring should be active.
	Enabled bool `mapstructure:"SHIPPING_TCC_ENABLED" default:"false"`
	// Sandbox defines whether TCC sandbox endpoints should be used.
	Sandbox bool `mapstructure:"SHIPPING_TCC_SANDBOX" default:"true"`
	// SandboxAccessToken defines TCC sandbox access-token values.
	SandboxAccessToken string `mapstructure:"SHIPPING_TCC_SANDBOX_ACCESS_TOKEN" default:""`
	// ProductionAccessToken defines TCC production access-token values.
	ProductionAccessToken string `mapstructure:"SHIPPING_TCC_PRODUCTION_ACCESS_TOKEN" default:""`
	// AccountNumber defines TCC account number values.
	AccountNumber string `mapstructure:"SHIPPING_TCC_ACCOUNT_NUMBER" default:""`
	// PaymentForm defines TCC payment-form values.
	PaymentForm int `mapstructure:"SHIPPING_TCC_PAYMENT_FORM" default:"1"`
	// CODFeePercent defines COD fee percentage applied to collected values.
	CODFeePercent float64 `mapstructure:"SHIPPING_TCC_COD_FEE_PERCENT" default:"0"`
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
