package s3

// Config defines S3 storage runtime configuration.
type Config struct {
	// Enabled defines whether storage integrations are enabled.
	Enabled bool `mapstructure:"STORAGE_ENABLED" default:"true"`
	// Endpoint defines S3 API endpoint values.
	Endpoint string `mapstructure:"STORAGE_ENDPOINT" default:""`
	// Region defines S3 region values.
	Region string `mapstructure:"STORAGE_REGION" default:""`
	// BucketName defines S3 bucket names.
	BucketName string `mapstructure:"STORAGE_BUCKET_NAME" default:""`
	// AccessKey defines access key credentials.
	AccessKey string `mapstructure:"STORAGE_ACCESS_KEY" default:""`
	// SecretKey defines secret key credentials.
	SecretKey string `mapstructure:"STORAGE_SECRET_KEY" default:""`
	// ForcePathStyle defines path-style S3 addressing behavior.
	ForcePathStyle bool `mapstructure:"STORAGE_FORCE_PATH_STYLE" default:"false"`
	// RequestTimeoutMS defines storage request timeout in milliseconds.
	RequestTimeoutMS int `mapstructure:"STORAGE_REQUEST_TIMEOUT_MS" default:"5000"`
	// CircuitBreakerEnabled defines whether S3 operations are guarded by a circuit breaker.
	CircuitBreakerEnabled bool `mapstructure:"STORAGE_CIRCUIT_BREAKER_ENABLED" default:"true"`
	// CircuitBreakerMaxRequests defines half-open max requests.
	CircuitBreakerMaxRequests uint32 `mapstructure:"STORAGE_CIRCUIT_BREAKER_MAX_REQUESTS" default:"1"`
	// CircuitBreakerIntervalMS defines closed-state counter reset intervals in milliseconds.
	CircuitBreakerIntervalMS int `mapstructure:"STORAGE_CIRCUIT_BREAKER_INTERVAL_MS" default:"60000"`
	// CircuitBreakerTimeoutMS defines open-state timeout windows in milliseconds.
	CircuitBreakerTimeoutMS int `mapstructure:"STORAGE_CIRCUIT_BREAKER_TIMEOUT_MS" default:"30000"`
	// CircuitBreakerFailureThreshold defines consecutive failure count that opens the breaker.
	CircuitBreakerFailureThreshold uint32 `mapstructure:"STORAGE_CIRCUIT_BREAKER_FAILURE_THRESHOLD" default:"5"`
}

// UploadRequest defines object upload input values.
type UploadRequest struct {
	// Key defines object key paths.
	Key string
	// ContentType defines object mime type values.
	ContentType string
	// Body defines raw object payload bytes.
	Body []byte
}
