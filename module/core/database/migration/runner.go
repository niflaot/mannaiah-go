package migration

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strings"
	"time"

	coredatabase "mannaiah/module/core/database"

	"github.com/golang-migrate/migrate/v4"
	migratedatabase "github.com/golang-migrate/migrate/v4/database"
	mysqlmigrate "github.com/golang-migrate/migrate/v4/database/mysql"
	postgresmigrate "github.com/golang-migrate/migrate/v4/database/postgres"
	sqlitemigrate "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	// ErrNilDB is returned when migration execution receives nil DB dependencies.
	ErrNilDB = errors.New("database migration db must not be nil")
	// ErrUnsupportedDriver is returned when migration execution receives unsupported driver values.
	ErrUnsupportedDriver = errors.New("unsupported migration database driver")
)

const (
	defaultMigrationTimeout = 30 * time.Second
)

//go:embed migrations/mysql/*.sql migrations/postgres/*.sql migrations/sqlite/*.sql
var migrationFiles embed.FS

// Config defines startup migration execution settings.
type Config struct {
	// Enabled defines whether startup should execute migrations.
	Enabled bool
	// Driver defines database driver values used to resolve migration database drivers.
	Driver string
	// Table defines migration state table names.
	Table string
	// Timeout defines best-effort migration execution timeout values.
	Timeout time.Duration
}

// FromDatabaseConfig maps database runtime config values into migration config values.
func FromDatabaseConfig(cfg coredatabase.Config) Config {
	table := strings.TrimSpace(cfg.MigrationsTable)
	if table == "" {
		table = "schema_migrations"
	}

	timeout := defaultMigrationTimeout
	if cfg.MigrationsTimeoutMS > 0 {
		timeout = time.Duration(cfg.MigrationsTimeoutMS) * time.Millisecond
	}

	return Config{
		Enabled: cfg.MigrationsEnabled,
		Driver:  strings.TrimSpace(cfg.Driver),
		Table:   table,
		Timeout: timeout,
	}
}

// Apply executes embedded SQL migrations against the provided database handle.
func Apply(ctx context.Context, db *gorm.DB, cfg Config, providedLogger *zap.Logger) error {
	if !cfg.Enabled {
		return nil
	}
	if db == nil {
		return ErrNilDB
	}

	logger := providedLogger
	if logger == nil {
		logger = zap.NewNop()
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("access sql db handle for migrations: %w", err)
	}

	databaseDriver, driverName, sourcePath, err := resolveDatabaseDriver(sqlDB, cfg)
	if err != nil {
		return err
	}
	sourceDriver, err := iofs.New(migrationFiles, sourcePath)
	if err != nil {
		return fmt.Errorf("create migration source driver: %w", err)
	}

	runner, err := migrate.NewWithInstance("iofs", sourceDriver, driverName, databaseDriver)
	if err != nil {
		return fmt.Errorf("create migration runner: %w", err)
	}

	migrationCtx := ctx
	if migrationCtx == nil {
		migrationCtx = context.Background()
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultMigrationTimeout
	}
	migrationCtx, cancel := context.WithTimeout(migrationCtx, timeout)
	defer cancel()

	go func() {
		<-migrationCtx.Done()
		if errors.Is(migrationCtx.Err(), context.DeadlineExceeded) || errors.Is(migrationCtx.Err(), context.Canceled) {
			runner.GracefulStop <- true
		}
	}()

	if upErr := runner.Up(); upErr != nil {
		if errors.Is(upErr, migrate.ErrNoChange) {
			logger.Debug("database migrations have no changes")
			return nil
		}
		if errors.Is(migrationCtx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("database migrations timeout exceeded: %w", migrationCtx.Err())
		}
		if errors.Is(migrationCtx.Err(), context.Canceled) {
			return fmt.Errorf("database migrations cancelled: %w", migrationCtx.Err())
		}
		return fmt.Errorf("apply database migrations: %w", upErr)
	}

	logger.Info("database migrations applied successfully")
	return nil
}

// resolveDatabaseDriver resolves migrate database drivers from configured database driver values.
func resolveDatabaseDriver(sqlDB *sql.DB, cfg Config) (migratedatabase.Driver, string, string, error) {
	driver := strings.ToLower(strings.TrimSpace(cfg.Driver))
	table := strings.TrimSpace(cfg.Table)
	if table == "" {
		table = "schema_migrations"
	}

	switch driver {
	case "mysql":
		databaseDriver, err := mysqlmigrate.WithInstance(sqlDB, &mysqlmigrate.Config{MigrationsTable: table})
		if err != nil {
			return nil, "", "", fmt.Errorf("create mysql migration driver: %w", err)
		}
		return databaseDriver, "mysql", "migrations/mysql", nil
	case "postgres", "postgresql":
		databaseDriver, err := postgresmigrate.WithInstance(sqlDB, &postgresmigrate.Config{MigrationsTable: table})
		if err != nil {
			return nil, "", "", fmt.Errorf("create postgres migration driver: %w", err)
		}
		return databaseDriver, "postgres", "migrations/postgres", nil
	case "sqlite":
		databaseDriver, err := sqlitemigrate.WithInstance(sqlDB, &sqlitemigrate.Config{MigrationsTable: table})
		if err != nil {
			return nil, "", "", fmt.Errorf("create sqlite migration driver: %w", err)
		}
		return databaseDriver, "sqlite3", "migrations/sqlite", nil
	default:
		return nil, "", "", fmt.Errorf("%w: %q", ErrUnsupportedDriver, cfg.Driver)
	}
}
