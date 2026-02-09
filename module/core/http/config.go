package http

// Config defines HTTP server runtime settings.
type Config struct {
	// Host defines the HTTP bind host.
	Host string `mapstructure:"HTTP_HOST" default:""`
	// Port defines the HTTP bind port.
	Port int `mapstructure:"HTTP_PORT" default:"0"`
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
}
