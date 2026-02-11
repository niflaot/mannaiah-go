package store

// Config defines Redis connectivity and runtime behavior.
type Config struct {
	// URL defines the Redis connection URL.
	URL string `mapstructure:"REDIS_URL" default:"redis://localhost:6379/0"`
	// Username defines the Redis ACL username override.
	Username string `mapstructure:"REDIS_USERNAME" default:""`
	// Password defines the Redis password override.
	Password string `mapstructure:"REDIS_PASSWORD" default:""`
	// PoolSize defines the maximum number of socket connections.
	PoolSize int `mapstructure:"REDIS_POOL_SIZE" default:"20"`
	// MinIdleConns defines the minimum number of idle pooled connections.
	MinIdleConns int `mapstructure:"REDIS_MIN_IDLE_CONNS" default:"5"`
	// DialTimeoutMS defines the Redis dial timeout in milliseconds.
	DialTimeoutMS int `mapstructure:"REDIS_DIAL_TIMEOUT_MS" default:"5000"`
	// ReadTimeoutMS defines the Redis read timeout in milliseconds.
	ReadTimeoutMS int `mapstructure:"REDIS_READ_TIMEOUT_MS" default:"3000"`
	// WriteTimeoutMS defines the Redis write timeout in milliseconds.
	WriteTimeoutMS int `mapstructure:"REDIS_WRITE_TIMEOUT_MS" default:"3000"`
	// ScanCount defines the SCAN hint used when iterating keys by pattern.
	ScanCount int64 `mapstructure:"REDIS_SCAN_COUNT" default:"200"`
	// BatchSize defines the batch size used by MGET during pattern retrieval.
	BatchSize int `mapstructure:"REDIS_BATCH_SIZE" default:"200"`
	// CircuitBreakerEnabled defines whether Redis operations are guarded by a circuit breaker.
	CircuitBreakerEnabled bool `mapstructure:"REDIS_CIRCUIT_BREAKER_ENABLED" default:"true"`
	// CircuitBreakerMaxRequests defines half-open max concurrent requests.
	CircuitBreakerMaxRequests uint32 `mapstructure:"REDIS_CIRCUIT_BREAKER_MAX_REQUESTS" default:"1"`
	// CircuitBreakerIntervalMS defines closed-state counter reset intervals in milliseconds.
	CircuitBreakerIntervalMS int `mapstructure:"REDIS_CIRCUIT_BREAKER_INTERVAL_MS" default:"60000"`
	// CircuitBreakerTimeoutMS defines open-state timeout windows in milliseconds.
	CircuitBreakerTimeoutMS int `mapstructure:"REDIS_CIRCUIT_BREAKER_TIMEOUT_MS" default:"30000"`
	// CircuitBreakerFailureThreshold defines consecutive failure count that opens the breaker.
	CircuitBreakerFailureThreshold uint32 `mapstructure:"REDIS_CIRCUIT_BREAKER_FAILURE_THRESHOLD" default:"5"`
}
