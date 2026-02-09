package database

// Config defines database and GORM runtime configuration.
type Config struct {
	// Driver defines the GORM dialector driver identifier.
	Driver string `mapstructure:"DB_DRIVER" default:"sqlite"`
	// DSN defines the connection string for the selected driver.
	DSN string `mapstructure:"DB_DSN" default:"file::memory:?cache=shared"`
	// MaxOpenConns defines the SQL connection pool max open connections.
	MaxOpenConns int `mapstructure:"DB_MAX_OPEN_CONNS" default:"25"`
	// MaxIdleConns defines the SQL connection pool max idle connections.
	MaxIdleConns int `mapstructure:"DB_MAX_IDLE_CONNS" default:"5"`
	// ConnMaxLifetimeMS defines connection maximum lifetime in milliseconds.
	ConnMaxLifetimeMS int `mapstructure:"DB_CONN_MAX_LIFETIME_MS" default:"600000"`
	// ConnMaxIdleTimeMS defines connection maximum idle time in milliseconds.
	ConnMaxIdleTimeMS int `mapstructure:"DB_CONN_MAX_IDLE_TIME_MS" default:"300000"`
	// GormLogLevel defines GORM logger level: silent, error, warn, info.
	GormLogLevel string `mapstructure:"DB_GORM_LOG_LEVEL" default:"warn"`
	// SlowQueryThresholdMS defines slow query threshold in milliseconds.
	SlowQueryThresholdMS int `mapstructure:"DB_GORM_SLOW_QUERY_THRESHOLD_MS" default:"200"`
}
