package database

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	// ErrUnsupportedDriver is returned when DB_DRIVER is not supported.
	ErrUnsupportedDriver = errors.New("unsupported database driver")
)

// Open initializes a GORM database connection and applies pool settings.
func Open(cfg Config, providedLogger *zap.Logger) (*gorm.DB, error) {
	dialector, err := resolveDialector(cfg.Driver, cfg.DSN)
	if err != nil {
		return nil, err
	}

	slowThreshold := time.Duration(cfg.SlowQueryThresholdMS) * time.Millisecond
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: newGormZapLogger(providedLogger, cfg.GormLogLevel, slowThreshold),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("access sql db handle: %w", err)
	}

	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetimeMS > 0 {
		sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetimeMS) * time.Millisecond)
	}
	if cfg.ConnMaxIdleTimeMS > 0 {
		sqlDB.SetConnMaxIdleTime(time.Duration(cfg.ConnMaxIdleTimeMS) * time.Millisecond)
	}

	return db, nil
}

// resolveDialector maps configured drivers to concrete GORM dialectors.
func resolveDialector(driver string, dsn string) (gorm.Dialector, error) {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "sqlite":
		return sqlite.Open(strings.TrimSpace(dsn)), nil
	case "postgres", "postgresql":
		return postgres.Open(strings.TrimSpace(dsn)), nil
	case "mysql":
		return mysql.Open(strings.TrimSpace(dsn)), nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedDriver, driver)
	}
}
