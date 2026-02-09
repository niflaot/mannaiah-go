package platform

// Config defines messaging platform runtime settings.
type Config struct {
	// GoChannelBuffer defines in-memory pubsub output channel buffer size.
	GoChannelBuffer int64 `mapstructure:"MESSAGING_GOCHANNEL_BUFFER" default:"100"`
	// RetryMaxRetries defines maximum retry attempts before dead-lettering.
	RetryMaxRetries int `mapstructure:"MESSAGING_RETRY_MAX_RETRIES" default:"3"`
	// RetryInitialIntervalMS defines retry initial delay in milliseconds.
	RetryInitialIntervalMS int `mapstructure:"MESSAGING_RETRY_INITIAL_INTERVAL_MS" default:"100"`
	// RetryMaxIntervalMS defines retry maximum delay in milliseconds.
	RetryMaxIntervalMS int `mapstructure:"MESSAGING_RETRY_MAX_INTERVAL_MS" default:"2000"`
	// RetryMultiplier defines retry backoff multiplier.
	RetryMultiplier float64 `mapstructure:"MESSAGING_RETRY_MULTIPLIER" default:"2.0"`
	// DLQSuffix defines the dead-letter topic suffix.
	DLQSuffix string `mapstructure:"MESSAGING_DLQ_SUFFIX" default:".dlq"`
}

// Normalized returns config values with safe defaults.
func (c Config) Normalized() Config {
	result := c
	if result.GoChannelBuffer <= 0 {
		result.GoChannelBuffer = 100
	}
	if result.RetryMaxRetries < 0 {
		result.RetryMaxRetries = 0
	}
	if result.RetryInitialIntervalMS <= 0 {
		result.RetryInitialIntervalMS = 100
	}
	if result.RetryMaxIntervalMS <= 0 {
		result.RetryMaxIntervalMS = 2000
	}
	if result.RetryMultiplier <= 0 {
		result.RetryMultiplier = 2.0
	}
	if result.DLQSuffix == "" {
		result.DLQSuffix = ".dlq"
	}

	return result
}
