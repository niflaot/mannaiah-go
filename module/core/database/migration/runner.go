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

// Operation defines supported migration command operations.
type Operation string

const (
	// OperationUp applies pending forward migrations.
	OperationUp Operation = "up"
	// OperationDown rolls back applied migrations.
	OperationDown Operation = "down"
	// OperationVersion reports current migration version and dirty state.
	OperationVersion Operation = "version"
	// OperationForce force-sets the migration version.
	OperationForce Operation = "force"
)

// RunOptions defines execution options for migration operations.
type RunOptions struct {
	// Operation defines the migration operation to execute.
	Operation Operation
	// Steps defines bounded step count for up/down operations when non-zero.
	Steps int
	// All defines whether down operations should rollback all migrations.
	All bool
	// ForceVersion defines the version applied by force operations.
	ForceVersion int
}

// Result defines migration execution results.
type Result struct {
	// Version defines the current migration version when available.
	Version uint
	// Dirty defines whether the migration state is dirty.
	Dirty bool
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
	_, err := Run(ctx, db, cfg, RunOptions{Operation: OperationUp}, providedLogger)
	return err
}

// Run executes migration operations against the provided database handle.
func Run(ctx context.Context, db *gorm.DB, cfg Config, options RunOptions, providedLogger *zap.Logger) (*Result, error) {
	if !cfg.Enabled {
		return &Result{}, nil
	}
	if db == nil {
		return nil, ErrNilDB
	}

	logger := providedLogger
	if logger == nil {
		logger = zap.NewNop()
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("access sql db handle for migrations: %w", err)
	}

	runner, migrationCtx, cancel, err := buildRunner(ctx, sqlDB, cfg)
	if err != nil {
		return nil, err
	}
	defer cancel()

	go func() {
		<-migrationCtx.Done()
		if errors.Is(migrationCtx.Err(), context.DeadlineExceeded) || errors.Is(migrationCtx.Err(), context.Canceled) {
			runner.GracefulStop <- true
		}
	}()

	result, runErr := runOperation(runner, options)
	if runErr != nil {
		if errors.Is(runErr, migrate.ErrNoChange) {
			logger.Debug("database migrations have no changes")
			return result, nil
		}
		if errors.Is(migrationCtx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("database migrations timeout exceeded: %w", migrationCtx.Err())
		}
		if errors.Is(migrationCtx.Err(), context.Canceled) {
			return nil, fmt.Errorf("database migrations cancelled: %w", migrationCtx.Err())
		}
		return nil, runErr
	}

	logger.Info("database migration operation completed", zap.String("operation", string(normalizeOperation(options.Operation))))
	return result, nil
}

// buildRunner resolves database and source drivers and builds migration runners with timeout context.
func buildRunner(ctx context.Context, sqlDB *sql.DB, cfg Config) (*migrate.Migrate, context.Context, context.CancelFunc, error) {
	databaseDriver, driverName, sourcePath, err := resolveDatabaseDriver(sqlDB, cfg)
	if err != nil {
		return nil, nil, nil, err
	}
	sourceDriver, err := iofs.New(migrationFiles, sourcePath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create migration source driver: %w", err)
	}

	runner, err := migrate.NewWithInstance("iofs", sourceDriver, driverName, databaseDriver)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create migration runner: %w", err)
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

	return runner, migrationCtx, cancel, nil
}

// normalizeOperation normalizes empty operations to up.
func normalizeOperation(operation Operation) Operation {
	trimmed := strings.TrimSpace(strings.ToLower(string(operation)))
	if trimmed == "" {
		return OperationUp
	}

	return Operation(trimmed)
}

// runOperation executes the requested operation and returns resulting migration state.
func runOperation(runner *migrate.Migrate, options RunOptions) (*Result, error) {
	if runner == nil {
		return nil, errors.New("migration runner must not be nil")
	}

	operation := normalizeOperation(options.Operation)
	result := &Result{}

	switch operation {
	case OperationUp:
		if options.Steps > 0 {
			if err := runner.Steps(options.Steps); err != nil {
				return nil, fmt.Errorf("apply database migrations steps: %w", err)
			}
		} else {
			if err := runner.Up(); err != nil {
				return nil, fmt.Errorf("apply database migrations: %w", err)
			}
		}
	case OperationDown:
		if options.All {
			if err := runner.Down(); err != nil {
				return nil, fmt.Errorf("rollback database migrations: %w", err)
			}
		} else {
			steps := options.Steps
			if steps <= 0 {
				steps = 1
			}
			if err := runner.Steps(-steps); err != nil {
				return nil, fmt.Errorf("rollback database migrations steps: %w", err)
			}
		}
	case OperationVersion:
		version, dirty, err := currentVersion(runner)
		if err != nil {
			return nil, err
		}
		result.Version = version
		result.Dirty = dirty
		return result, nil
	case OperationForce:
		if options.ForceVersion < 0 {
			return nil, errors.New("force operation requires non-negative version")
		}
		if err := runner.Force(options.ForceVersion); err != nil {
			return nil, fmt.Errorf("force database migration version: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported migration operation: %q", string(operation))
	}

	version, dirty, err := currentVersion(runner)
	if err != nil {
		return nil, err
	}
	result.Version = version
	result.Dirty = dirty

	return result, nil
}

// currentVersion resolves current migration version while treating nil-version state as version zero.
func currentVersion(runner *migrate.Migrate) (uint, bool, error) {
	version, dirty, err := runner.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("read database migration version: %w", err)
	}

	return version, dirty, nil
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
