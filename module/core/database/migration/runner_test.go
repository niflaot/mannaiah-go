package migration

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"testing"

	coredatabase "mannaiah/module/core/database"

	"go.uber.org/zap"
)

// TestEmbeddedMigrationPairs verifies every embedded migration has both up/down files per driver directory.
func TestEmbeddedMigrationPairs(t *testing.T) {
	driverDirectories := []string{
		"migrations/mysql",
		"migrations/sqlite",
	}

	for _, driverDirectory := range driverDirectories {
		entries, err := fs.ReadDir(migrationFiles, driverDirectory)
		if err != nil {
			t.Fatalf("ReadDir(%q) error = %v", driverDirectory, err)
		}

		pairs := map[string]map[string]bool{}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			name := entry.Name()
			switch {
			case strings.HasSuffix(name, ".up.sql"):
				base := strings.TrimSuffix(name, ".up.sql")
				if pairs[base] == nil {
					pairs[base] = map[string]bool{}
				}
				pairs[base]["up"] = true
			case strings.HasSuffix(name, ".down.sql"):
				base := strings.TrimSuffix(name, ".down.sql")
				if pairs[base] == nil {
					pairs[base] = map[string]bool{}
				}
				pairs[base]["down"] = true
			default:
				t.Fatalf("unexpected migration file %q in %q", name, driverDirectory)
			}
		}

		for base, directions := range pairs {
			if !directions["up"] {
				t.Fatalf("missing up migration for %q in %q", base, driverDirectory)
			}
			if !directions["down"] {
				t.Fatalf("missing down migration for %q in %q", base, driverDirectory)
			}

			if _, err := migrationFiles.Open(path.Join(driverDirectory, base+".up.sql")); err != nil {
				t.Fatalf("Open(%q) error = %v", path.Join(driverDirectory, base+".up.sql"), err)
			}
			if _, err := migrationFiles.Open(path.Join(driverDirectory, base+".down.sql")); err != nil {
				t.Fatalf("Open(%q) error = %v", path.Join(driverDirectory, base+".down.sql"), err)
			}
		}
	}
}

// TestFromDatabaseConfig verifies migration config mapping behavior from core database config.
func TestFromDatabaseConfig(t *testing.T) {
	cfg := FromDatabaseConfig(coredatabase.Config{
		Driver:              "mysql",
		MigrationsEnabled:   true,
		MigrationsTable:     "custom_schema_migrations",
		MigrationsTimeoutMS: 15000,
	})

	if cfg.Driver != "mysql" {
		t.Fatalf("cfg.Driver = %q, want %q", cfg.Driver, "mysql")
	}
	if !cfg.Enabled {
		t.Fatalf("cfg.Enabled = false, want true")
	}
	if cfg.Table != "custom_schema_migrations" {
		t.Fatalf("cfg.Table = %q, want %q", cfg.Table, "custom_schema_migrations")
	}
}

// TestApplyDisabled verifies disabled migration execution no-ops safely.
func TestApplyDisabled(t *testing.T) {
	if err := Apply(context.Background(), nil, Config{Enabled: false}, zap.NewNop()); err != nil {
		t.Fatalf("Apply(disabled) error = %v", err)
	}
}

// TestApplyNilDB verifies enabled migration execution requires db dependencies.
func TestApplyNilDB(t *testing.T) {
	err := Apply(context.Background(), nil, Config{Enabled: true, Driver: "mysql"}, zap.NewNop())
	if !errors.Is(err, ErrNilDB) {
		t.Fatalf("Apply(nil db) error = %v, want %v", err, ErrNilDB)
	}
}

// TestResolveDatabaseDriverRejectsUnsupported verifies unsupported migration driver handling behavior.
func TestResolveDatabaseDriverRejectsUnsupported(t *testing.T) {
	_, _, _, err := resolveDatabaseDriver(nil, Config{Driver: "postgres"})
	if !errors.Is(err, ErrUnsupportedDriver) {
		t.Fatalf("resolveDatabaseDriver(postgres) error = %v, want %v", err, ErrUnsupportedDriver)
	}
	if !strings.Contains(err.Error(), "postgres") {
		t.Fatalf("resolveDatabaseDriver(postgres) error = %v, want quoted driver", err)
	}
}

// TestRunForceRequiresVersion verifies force operation validation behavior.
func TestRunForceRequiresVersion(t *testing.T) {
	_, runErr := runOperation(nil, RunOptions{Operation: OperationForce, ForceVersion: -1})
	if runErr == nil {
		t.Fatalf("runOperation(force without version) expected error")
	}
}

// TestLatestEmbeddedMigrationVersion verifies embedded latest-version discovery per driver directory.
func TestLatestEmbeddedMigrationVersion(t *testing.T) {
	version, err := latestEmbeddedMigrationVersion("migrations/mysql")
	if err != nil {
		t.Fatalf("latestEmbeddedMigrationVersion(mysql) error = %v", err)
	}
	if version != 45 {
		t.Fatalf("latestEmbeddedMigrationVersion(mysql) = %d, want 45", version)
	}

	version, err = latestEmbeddedMigrationVersion("migrations/sqlite")
	if err != nil {
		t.Fatalf("latestEmbeddedMigrationVersion(sqlite) error = %v", err)
	}
	if version != 45 {
		t.Fatalf("latestEmbeddedMigrationVersion(sqlite) = %d, want 45", version)
	}
}

// TestIsMissingCurrentDownMigrationError verifies matching of the tolerated startup migration error.
func TestIsMissingCurrentDownMigrationError(t *testing.T) {
	err := fmt.Errorf("apply database migrations: no migration found for version 43: read down for version 43 migrations/mysql: file does not exist")
	if !isMissingCurrentDownMigrationError(err, 43) {
		t.Fatalf("isMissingCurrentDownMigrationError() = false, want true")
	}
	if isMissingCurrentDownMigrationError(err, 42) {
		t.Fatalf("isMissingCurrentDownMigrationError() = true for wrong version, want false")
	}
}
