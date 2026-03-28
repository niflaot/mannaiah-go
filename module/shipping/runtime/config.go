package runtime

// Config defines shipping runtime configuration values.
type Config struct {
	// Enabled defines whether shipping module wiring should be active.
	Enabled bool `mapstructure:"SHIPPING_ENABLED" default:"true"`
	// TrackingCacheTTLSeconds defines tracking cache TTL values in seconds.
	TrackingCacheTTLSeconds int `mapstructure:"SHIPPING_TRACKING_CACHE_TTL_SECONDS" default:"300"`
	// BatchManifestCacheTTLSeconds defines merged manifest-document cache TTL values in seconds.
	BatchManifestCacheTTLSeconds int `mapstructure:"SHIPPING_BATCH_MANIFEST_CACHE_TTL_SECONDS" default:"300"`
	// BatchManifestTemplatePath defines optional JSON template file path for manifest cover rendering.
	BatchManifestTemplatePath string `mapstructure:"SHIPPING_BATCH_MANIFEST_TEMPLATE_PATH" default:""`
	// TransactionalTrackingBaseURL defines the base URL for shipping tracking links in transactional emails.
	TransactionalTrackingBaseURL string `mapstructure:"SHIPPING_TRANSACTIONAL_TRACKING_BASE_URL" default:"https://rastreo.flockstore.co"`
	// TransactionalHelpPhoneURL defines the WhatsApp help URL for transactional shipping emails.
	TransactionalHelpPhoneURL string `mapstructure:"SHIPPING_TRANSACTIONAL_HELP_PHONE_URL" default:"https://wa.me/573104314990"`
	// Quotation defines quotation behavior configuration values.
	Quotation QuotationConfig `mapstructure:",squash"`
	// TCC defines TCC carrier configuration values.
	TCC TCCConfig `mapstructure:",squash"`
	// DefaultSender defines fallback sender information used for mark generation.
	DefaultSender DefaultSenderConfig `mapstructure:",squash"`
}

// QuotationConfig defines quotation behavior configuration values.
type QuotationConfig struct {
	// ExpirationTTLMinutes defines how many minutes stored quotations remain valid before expiring.
	// Zero or negative values default to 10 minutes.
	ExpirationTTLMinutes int `mapstructure:"SHIPPING_QUOTATION_TTL_MINUTES" default:"10"`
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
	// ParcelAccountNumber defines TCC account number used for parcel (standard) shipments.
	ParcelAccountNumber string `mapstructure:"SHIPPING_TCC_PARCEL_ACCOUNT_NUMBER" default:""`
	// ExpressAccountNumber defines TCC account number used for express shipments.
	ExpressAccountNumber string `mapstructure:"SHIPPING_TCC_EXPRESS_ACCOUNT_NUMBER" default:""`
	// Declaration defines the default contents description sent as dicecontener for each unit.
	Declaration string `mapstructure:"SHIPPING_TCC_DECLARATION" default:""`
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
