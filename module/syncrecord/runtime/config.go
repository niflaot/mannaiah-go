package runtime

// Config defines sync record runtime configuration values.
type Config struct {
	// Enabled defines whether sync record module wiring should be active.
	Enabled bool `mapstructure:"SYNC_RECORD_ENABLED" default:"true"`
	// RetentionDays defines data retention in days.
	RetentionDays int `mapstructure:"SYNC_RECORD_RETENTION_DAYS" default:"90"`
	// CleanupEnabled defines whether cleanup cron is enabled.
	CleanupEnabled bool `mapstructure:"SYNC_RECORD_CLEANUP_ENABLED" default:"true"`
	// CleanupCron defines cleanup cron specs.
	CleanupCron string `mapstructure:"SYNC_RECORD_CLEANUP_CRON" default:"0 3 * * *"`
	// CleanupTimeoutMS defines cleanup execution timeout values in milliseconds.
	CleanupTimeoutMS int `mapstructure:"SYNC_RECORD_CLEANUP_TIMEOUT_MS" default:"60000"`
}
