package runtime

// Config defines analytics runtime configuration values.
type Config struct {
	// Enabled defines whether analytics module wiring should be active.
	Enabled bool `mapstructure:"ANALYTICS_ENABLED" default:"false"`
	// ClickHouseDSN defines ClickHouse connection DSN values.
	ClickHouseDSN string `mapstructure:"ANALYTICS_CLICKHOUSE_DSN" default:""`
	// MaxOpenConns defines max open clickhouse connection values.
	MaxOpenConns int `mapstructure:"ANALYTICS_CLICKHOUSE_MAX_OPEN_CONNS" default:"10"`
	// MaxIdleConns defines max idle clickhouse connection values.
	MaxIdleConns int `mapstructure:"ANALYTICS_CLICKHOUSE_MAX_IDLE_CONNS" default:"5"`
	// ConnMaxLifetimeMS defines clickhouse connection max lifetime values in milliseconds.
	ConnMaxLifetimeMS int64 `mapstructure:"ANALYTICS_CLICKHOUSE_CONN_MAX_LIFETIME_MS" default:"600000"`
	// BatchSize defines insert batch-size values.
	BatchSize int `mapstructure:"ANALYTICS_CLICKHOUSE_BATCH_SIZE" default:"1000"`
	// FlushIntervalMS defines batch flush interval values in milliseconds.
	FlushIntervalMS int64 `mapstructure:"ANALYTICS_CLICKHOUSE_FLUSH_INTERVAL_MS" default:"5000"`
	// MigrationEnabled defines whether analytics backend migrations should run on startup.
	MigrationEnabled bool `mapstructure:"ANALYTICS_CLICKHOUSE_MIGRATION_ENABLED" default:"true"`
}
