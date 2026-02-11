package woocommerce

// Config defines WooCommerce integration configuration values.
type Config struct {
	// URL defines WooCommerce store base URLs.
	URL string `mapstructure:"WOOCOMMERCE_URL" default:""`
	// ConsumerKey defines WooCommerce API consumer key values.
	ConsumerKey string `mapstructure:"WOOCOMMERCE_CONSUMER_KEY" default:""`
	// ConsumerSecret defines WooCommerce API consumer secret values.
	ConsumerSecret string `mapstructure:"WOOCOMMERCE_CONSUMER_SECRET" default:""`
	// SyncContacts enables contact sync behavior.
	SyncContacts bool `mapstructure:"WOOCOMMERCE_SYNC_CONTACTS" default:"false"`
	// SyncContactsCron defines cron specs for scheduled contact sync behavior.
	SyncContactsCron string `mapstructure:"WOOCOMMERCE_SYNC_CONTACTS_CRON" default:"0 0 * * *"`
	// SyncPageSize defines order page sizes for sync behavior.
	SyncPageSize int `mapstructure:"WOOCOMMERCE_SYNC_PAGE_SIZE" default:"100"`
	// SyncWorkers defines concurrent upsert worker counts for sync behavior.
	SyncWorkers int `mapstructure:"WOOCOMMERCE_SYNC_WORKERS" default:"8"`
	// RequestTimeoutMS defines WooCommerce API timeout values in milliseconds.
	RequestTimeoutMS int `mapstructure:"WOOCOMMERCE_REQUEST_TIMEOUT_MS" default:"5000"`
	// VerifySSL defines whether WooCommerce TLS certificates must be verified.
	VerifySSL bool `mapstructure:"WOOCOMMERCE_VERIFY_SSL" default:"true"`
	// ValidationTimeoutMS defines integration validation timeout values in milliseconds.
	ValidationTimeoutMS int `mapstructure:"WOOCOMMERCE_VALIDATION_TIMEOUT_MS" default:"3000"`
}
