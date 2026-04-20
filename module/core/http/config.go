package http

// Config defines HTTP server runtime settings.
type Config struct {
	// Host defines an optional code-level bind host override for standalone server construction.
	Host string `mapstructure:"-"`
	// Port defines an optional code-level bind port override for standalone server construction.
	Port int `mapstructure:"-"`
	// AppName defines the Fiber application name.
	AppName string `mapstructure:"HTTP_APP_NAME" default:"mannaiah-http"`
	// Prefork enables Fiber prefork mode.
	Prefork bool `mapstructure:"HTTP_PREFORK" default:"false"`
	// ServerHeader defines the HTTP server header value.
	ServerHeader string `mapstructure:"HTTP_SERVER_HEADER" default:"mannaiah"`
	// ReadTimeoutMS defines request read timeout in milliseconds.
	ReadTimeoutMS int `mapstructure:"HTTP_READ_TIMEOUT_MS" default:"30000"`
	// WriteTimeoutMS defines response write timeout in milliseconds.
	WriteTimeoutMS int `mapstructure:"HTTP_WRITE_TIMEOUT_MS" default:"30000"`
	// IdleTimeoutMS defines idle connection timeout in milliseconds.
	IdleTimeoutMS int `mapstructure:"HTTP_IDLE_TIMEOUT_MS" default:"120000"`
	// CORSAllowedOrigins defines a comma-separated list of allowed CORS origins. Empty disables CORS middleware.
	CORSAllowedOrigins string `mapstructure:"HTTP_CORS_ALLOWED_ORIGINS" default:""`
	// RateLimitMax defines the maximum number of requests allowed per RateLimitWindowMS. Zero disables rate limiting.
	RateLimitMax int `mapstructure:"HTTP_RATE_LIMIT_MAX" default:"0"`
	// RateLimitWindowMS defines the rate limit sliding window duration in milliseconds.
	RateLimitWindowMS int `mapstructure:"HTTP_RATE_LIMIT_WINDOW_MS" default:"60000"`
}
