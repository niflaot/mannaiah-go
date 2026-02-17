package runtime

// Config defines Falabella integration configuration values.
type Config struct {
	// URL defines Falabella base URL values.
	URL string `mapstructure:"FALABELLA_URL" default:"https://sellercenter-api.falabella.com"`
	// UserID defines Falabella API user values.
	UserID string `mapstructure:"FALABELLA_USER_ID" default:""`
	// APIKey defines Falabella API key values.
	APIKey string `mapstructure:"FALABELLA_API_KEY" default:""`
	// Version defines Falabella API version values.
	Version string `mapstructure:"FALABELLA_API_VERSION" default:"1.0"`
	// RequestTimeoutMS defines Falabella API timeout values in milliseconds.
	RequestTimeoutMS int `mapstructure:"FALABELLA_REQUEST_TIMEOUT_MS" default:"5000"`
	// ValidationTimeoutMS defines startup validation timeout values in milliseconds.
	ValidationTimeoutMS int `mapstructure:"FALABELLA_VALIDATION_TIMEOUT_MS" default:"3000"`
	// CircuitBreakerEnabled defines whether Falabella requests are circuit-breaker protected.
	CircuitBreakerEnabled bool `mapstructure:"FALABELLA_CIRCUIT_BREAKER_ENABLED" default:"true"`
	// CircuitBreakerMaxRequests defines half-open max concurrent requests.
	CircuitBreakerMaxRequests uint32 `mapstructure:"FALABELLA_CIRCUIT_BREAKER_MAX_REQUESTS" default:"1"`
	// CircuitBreakerIntervalMS defines closed-state counter reset intervals in milliseconds.
	CircuitBreakerIntervalMS int `mapstructure:"FALABELLA_CIRCUIT_BREAKER_INTERVAL_MS" default:"60000"`
	// CircuitBreakerTimeoutMS defines open-state timeout windows in milliseconds.
	CircuitBreakerTimeoutMS int `mapstructure:"FALABELLA_CIRCUIT_BREAKER_TIMEOUT_MS" default:"30000"`
	// CircuitBreakerFailureThreshold defines consecutive failure count that opens the breaker.
	CircuitBreakerFailureThreshold uint32 `mapstructure:"FALABELLA_CIRCUIT_BREAKER_FAILURE_THRESHOLD" default:"3"`
	// ProductRealm defines product datasheet realm values used for Falabella sync.
	ProductRealm string `mapstructure:"FALABELLA_PRODUCT_REALM" default:"falabella"`
	// ProductCategoryID defines Falabella primary category identifier values.
	ProductCategoryID string `mapstructure:"FALABELLA_PRODUCT_CATEGORY_ID" default:"1638"`
	// ProductGlobalIdentifier defines Falabella global identifier values.
	ProductGlobalIdentifier string `mapstructure:"FALABELLA_PRODUCT_GLOBAL_IDENTIFIER" default:"G08010305"`
	// ProductAttributeSetID defines Falabella attribute-set identifier values.
	ProductAttributeSetID string `mapstructure:"FALABELLA_PRODUCT_ATTRIBUTE_SET_ID" default:"5"`
}
