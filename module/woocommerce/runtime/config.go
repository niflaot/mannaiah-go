package runtime

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
	// SyncOrders enables order sync behavior.
	SyncOrders bool `mapstructure:"WOOCOMMERCE_SYNC_ORDERS" default:"false"`
	// SyncOrdersCron defines cron specs for scheduled order sync behavior.
	SyncOrdersCron string `mapstructure:"WOOCOMMERCE_SYNC_ORDERS_CRON" default:"0 0 * * *"`
	// SyncPageSize defines order page sizes for sync behavior.
	SyncPageSize int `mapstructure:"WOOCOMMERCE_SYNC_PAGE_SIZE" default:"100"`
	// SyncWorkers defines concurrent upsert worker counts for sync behavior.
	SyncWorkers int `mapstructure:"WOOCOMMERCE_SYNC_WORKERS" default:"8"`
	// SyncTimeoutMS defines cron sync execution timeout values in milliseconds.
	SyncTimeoutMS int `mapstructure:"WOOCOMMERCE_SYNC_TIMEOUT_MS" default:"600000"`
	// RequestTimeoutMS defines WooCommerce API timeout values in milliseconds.
	RequestTimeoutMS int `mapstructure:"WOOCOMMERCE_REQUEST_TIMEOUT_MS" default:"5000"`
	// VerifySSL defines whether WooCommerce TLS certificates must be verified.
	VerifySSL bool `mapstructure:"WOOCOMMERCE_VERIFY_SSL" default:"true"`
	// ValidationTimeoutMS defines integration validation timeout values in milliseconds.
	ValidationTimeoutMS int `mapstructure:"WOOCOMMERCE_VALIDATION_TIMEOUT_MS" default:"3000"`
	// CircuitBreakerEnabled defines whether WooCommerce source requests are circuit-breaker protected.
	CircuitBreakerEnabled bool `mapstructure:"WOOCOMMERCE_CIRCUIT_BREAKER_ENABLED" default:"true"`
	// CircuitBreakerMaxRequests defines half-open max concurrent requests.
	CircuitBreakerMaxRequests uint32 `mapstructure:"WOOCOMMERCE_CIRCUIT_BREAKER_MAX_REQUESTS" default:"1"`
	// CircuitBreakerIntervalMS defines closed-state counter reset intervals in milliseconds.
	CircuitBreakerIntervalMS int `mapstructure:"WOOCOMMERCE_CIRCUIT_BREAKER_INTERVAL_MS" default:"60000"`
	// CircuitBreakerTimeoutMS defines open-state timeout windows in milliseconds.
	CircuitBreakerTimeoutMS int `mapstructure:"WOOCOMMERCE_CIRCUIT_BREAKER_TIMEOUT_MS" default:"30000"`
	// CircuitBreakerFailureThreshold defines consecutive failure count that opens the source breaker.
	CircuitBreakerFailureThreshold uint32 `mapstructure:"WOOCOMMERCE_CIRCUIT_BREAKER_FAILURE_THRESHOLD" default:"3"`
}
