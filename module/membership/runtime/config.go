package runtime

// Config defines membership runtime configuration values.
type Config struct {
	// Enabled defines whether membership module wiring should be active.
	Enabled bool `mapstructure:"MEMBERSHIP_ENABLED" default:"true"`
}
