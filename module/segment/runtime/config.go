package runtime

// Config defines segment runtime configuration values.
type Config struct {
	// Enabled defines whether segment module wiring should be active.
	Enabled bool `mapstructure:"SEGMENT_ENABLED" default:"false"`
}
