package cron

// Config defines cron scheduler runtime settings.
type Config struct {
	// Location defines the IANA timezone used by cron expression evaluation.
	Location string `mapstructure:"CRON_LOCATION" default:"UTC"`
	// WithSeconds enables six-field cron expressions with seconds precision.
	WithSeconds bool `mapstructure:"CRON_WITH_SECONDS" default:"false"`
}
