package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseOptionsValidatesForceRequirements verifies force options validation behavior.
func TestParseOptionsValidatesForceRequirements(t *testing.T) {
	if _, err := parseOptions([]string{"--operation=force"}); err == nil {
		t.Fatalf("parseOptions(force missing version) expected error")
	}
	if _, err := parseOptions([]string{"--operation=force", "--force-version=3"}); err != nil {
		t.Fatalf("parseOptions(force with version) error = %v", err)
	}
}

// TestParseOptionsRejectsInvalidOperation verifies unsupported operation rejection behavior.
func TestParseOptionsRejectsInvalidOperation(t *testing.T) {
	if _, err := parseOptions([]string{"--operation=invalid"}); err == nil {
		t.Fatalf("parseOptions(invalid operation) expected error")
	}
}

// TestRunVersionCommand verifies dedicated command version operation behavior.
func TestRunVersionCommand(t *testing.T) {
	envFile := writeMigrationEnvFile(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	err := run(context.Background(), []string{"--env-file", envFile, "--operation", "version"}, stdout, stderr)
	if err != nil {
		t.Fatalf("run(version) error = %v", err)
	}
	if !strings.Contains(stdout.String(), "operation=version") {
		t.Fatalf("stdout = %q, want operation=version", stdout.String())
	}
}

// TestRunUpCommand verifies dedicated command up operation behavior.
func TestRunUpCommand(t *testing.T) {
	envFile := writeMigrationEnvFile(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	err := run(context.Background(), []string{"--env-file", envFile, "--operation", "up"}, stdout, stderr)
	if err != nil {
		t.Fatalf("run(up) error = %v", err)
	}
	if !strings.Contains(stdout.String(), "operation=up") {
		t.Fatalf("stdout = %q, want operation=up", stdout.String())
	}
}

// writeMigrationEnvFile creates a temp env file for migration command tests.
func writeMigrationEnvFile(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	dsn := filepath.Join(dir, "migrate.sqlite")
	envFile := filepath.Join(dir, ".env")
	content := strings.Join([]string{
		"CORE_ENVIRONMENT=test",
		"LOGGING_LEVEL=error",
		"DB_DRIVER=sqlite",
		"DB_DSN=" + dsn,
		"DB_MIGRATIONS_ENABLED=true",
		"DB_MIGRATIONS_TABLE=schema_migrations",
		"DB_MIGRATIONS_TIMEOUT_MS=30000",
	}, "\n") + "\n"
	if err := os.WriteFile(envFile, []byte(content), 0o600); err != nil {
		t.Fatalf("os.WriteFile(env) error = %v", err)
	}

	return envFile
}
