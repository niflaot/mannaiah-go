package migration

import (
	"context"
	"errors"
	"strings"
	"testing"

	coredatabase "mannaiah/module/core/database"

	"go.uber.org/zap"
)

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
	_, _, _, err := resolveDatabaseDriver(nil, Config{Driver: "sqlite"})
	if !errors.Is(err, ErrUnsupportedDriver) {
		t.Fatalf("resolveDatabaseDriver(sqlite) error = %v, want %v", err, ErrUnsupportedDriver)
	}
	if !strings.Contains(err.Error(), "sqlite") {
		t.Fatalf("resolveDatabaseDriver(sqlite) error = %v, want quoted driver", err)
	}
}

// TestRunForceRequiresVersion verifies force operation validation behavior.
func TestRunForceRequiresVersion(t *testing.T) {
	_, runErr := runOperation(nil, RunOptions{Operation: OperationForce, ForceVersion: -1})
	if runErr == nil {
		t.Fatalf("runOperation(force without version) expected error")
	}
}
