package config

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	corelogger "mannaiah/module/core/logger"
)

// requiredProbe is a sample config with a single required key.
type requiredProbe struct {
	// Value is a required field used to verify missing-key validation.
	Value string `mapstructure:"UT_REQUIRED_VALUE"`
}

// defaultsProbe is a sample config that verifies default tag behavior.
type defaultsProbe struct {
	// Name is loaded from default tags when no source value is set.
	Name string `mapstructure:"UT_DEFAULTS_NAME" default:"mannaiah"`
	// Enabled is loaded from default tags when no source value is set.
	Enabled bool `mapstructure:"UT_DEFAULTS_ENABLED" default:"true"`
	// Count is loaded from default tags when no source value is set.
	Count int `mapstructure:"UT_DEFAULTS_COUNT" default:"7"`
}

// moduleConfig is a generic module-level configuration example.
type moduleConfig struct {
	// APIKey is a required module-level value.
	APIKey string `mapstructure:"UT_MODULE_API_KEY"`
	// Timeout is an optional module-level value.
	Timeout int `mapstructure:"UT_MODULE_TIMEOUT" default:"30"`
}

// moduleFeatureConfig is a generic nested module feature configuration example.
type moduleFeatureConfig struct {
	// Enabled indicates whether the feature is enabled.
	Enabled bool `mapstructure:"UT_MODULE_FEATURE_ENABLED" default:"true"`
	// Name identifies the feature instance.
	Name string `mapstructure:"UT_MODULE_FEATURE_NAME"`
}

// moduleExtensionConfig is a generic extension config that flattens feature keys.
type moduleExtensionConfig struct {
	// Feature contains nested feature fields flattened by mapstructure squash.
	Feature moduleFeatureConfig `mapstructure:",squash"`
}

// TestLoadFromDotEnv verifies base config loading from the .env file.
func TestLoadFromDotEnv(t *testing.T) {
	envFile := writeEnvFile(
		t,
		"CORE_HOST=127.0.0.1",
		"CORE_PORT=9090",
		"CORE_ENVIRONMENT=staging",
		"LOG_FORMAT=json",
		"LOG_LEVEL=debug",
	)

	var cfg Core
	if err := NewLoader(envFile, nil).Load(&cfg); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Host != "127.0.0.1" {
		t.Fatalf("Host = %q, want %q", cfg.Host, "127.0.0.1")
	}
	if cfg.Port != 9090 {
		t.Fatalf("Port = %d, want %d", cfg.Port, 9090)
	}
	if cfg.Environment != "staging" {
		t.Fatalf("Environment = %q, want %q", cfg.Environment, "staging")
	}
	if cfg.Logging.Format != "json" {
		t.Fatalf("Logging.Format = %q, want %q", cfg.Logging.Format, "json")
	}
	if cfg.Logging.Level != "debug" {
		t.Fatalf("Logging.Level = %q, want %q", cfg.Logging.Level, "debug")
	}
}

// TestLoadEnvOverridesDotEnv verifies environment variables override .env values.
func TestLoadEnvOverridesDotEnv(t *testing.T) {
	envFile := writeEnvFile(
		t,
		"CORE_HOST=file-host",
		"CORE_PORT=9090",
	)
	t.Setenv("CORE_HOST", "env-host")

	var cfg Core
	if err := NewLoader(envFile, nil).Load(&cfg); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Host != "env-host" {
		t.Fatalf("Host = %q, want %q", cfg.Host, "env-host")
	}
}

// TestLoadAppliesDefaults verifies default tags are applied when values are absent.
func TestLoadAppliesDefaults(t *testing.T) {
	envFile := filepath.Join(t.TempDir(), ".env.missing")
	var probe defaultsProbe

	if err := NewLoader(envFile, nil).Load(&probe); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if probe.Name != "mannaiah" {
		t.Fatalf("Name = %q, want %q", probe.Name, "mannaiah")
	}
	if probe.Enabled != true {
		t.Fatalf("Enabled = %v, want %v", probe.Enabled, true)
	}
	if probe.Count != 7 {
		t.Fatalf("Count = %d, want %d", probe.Count, 7)
	}
}

// TestLoadValidationErrorLogsMissingFields verifies required missing fields are logged and returned.
func TestLoadValidationErrorLogsMissingFields(t *testing.T) {
	envFile := filepath.Join(t.TempDir(), ".env.missing")
	var output bytes.Buffer

	startupLogger, err := corelogger.NewWithWriters(
		corelogger.Settings{Format: "json", Level: "debug"},
		&output,
		&output,
	)
	if err != nil {
		t.Fatalf("NewWithWriters() error = %v", err)
	}

	var probe requiredProbe
	loadErr := NewLoader(envFile, startupLogger).Load(&probe)
	if loadErr == nil {
		t.Fatalf("expected validation error for missing required field")
	}

	var validationErr ValidationError
	if !errors.As(loadErr, &validationErr) {
		t.Fatalf("expected ValidationError, got %T (%v)", loadErr, loadErr)
	}
	if len(validationErr.Missing) != 1 {
		t.Fatalf("missing count = %d, want %d", len(validationErr.Missing), 1)
	}
	if validationErr.Missing[0].Key != "UT_REQUIRED_VALUE" {
		t.Fatalf("missing key = %q, want %q", validationErr.Missing[0].Key, "UT_REQUIRED_VALUE")
	}

	logOutput := output.String()
	if !strings.Contains(logOutput, "required configuration value missing") {
		t.Fatalf("expected startup logger error output, got %q", logOutput)
	}
	if !strings.Contains(logOutput, "UT_REQUIRED_VALUE") {
		t.Fatalf("expected missing key in startup logger output, got %q", logOutput)
	}
}

// TestLoadSupportsMultipleModuleStructs verifies combined loading across multiple module config structs.
func TestLoadSupportsMultipleModuleStructs(t *testing.T) {
	envFile := writeEnvFile(
		t,
		"UT_MODULE_API_KEY=token",
		"UT_MODULE_FEATURE_NAME=payments",
		"CORE_HOST=localhost",
	)

	var coreCfg Core
	var moduleCfg moduleConfig
	var featureCfg moduleExtensionConfig

	if err := NewLoader(envFile, nil).Load(&coreCfg, &moduleCfg, &featureCfg); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if moduleCfg.APIKey != "token" {
		t.Fatalf("APIKey = %q, want %q", moduleCfg.APIKey, "token")
	}
	if moduleCfg.Timeout != 30 {
		t.Fatalf("Timeout = %d, want %d", moduleCfg.Timeout, 30)
	}
	if featureCfg.Feature.Enabled != true {
		t.Fatalf("Feature.Enabled = %v, want %v", featureCfg.Feature.Enabled, true)
	}
	if featureCfg.Feature.Name != "payments" {
		t.Fatalf("Feature.Name = %q, want %q", featureCfg.Feature.Name, "payments")
	}
	if coreCfg.Host != "localhost" {
		t.Fatalf("Core.Host = %q, want %q", coreCfg.Host, "localhost")
	}
}

// TestLoadRejectsInvalidTargets verifies loader rejects non-struct and non-pointer targets.
func TestLoadRejectsInvalidTargets(t *testing.T) {
	envFile := filepath.Join(t.TempDir(), ".env.missing")
	loader := NewLoader(envFile, nil)

	if err := loader.Load(Core{}); err == nil {
		t.Fatalf("expected error when target is not pointer")
	}

	var text string
	if err := loader.Load(&text); err == nil {
		t.Fatalf("expected error when target is not struct pointer")
	}
}

// TestLoadAllowsMissingDotEnvWhenEnvironmentHasValues verifies env-only startup works without a .env file.
func TestLoadAllowsMissingDotEnvWhenEnvironmentHasValues(t *testing.T) {
	t.Setenv("UT_REQUIRED_VALUE", "present")
	envFile := filepath.Join(t.TempDir(), ".env.missing")

	var probe requiredProbe
	if err := NewLoader(envFile, nil).Load(&probe); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if probe.Value != "present" {
		t.Fatalf("Value = %q, want %q", probe.Value, "present")
	}
}

// writeEnvFile writes a temporary .env file with the provided lines.
func writeEnvFile(t *testing.T, lines ...string) string {
	t.Helper()

	filePath := filepath.Join(t.TempDir(), ".env")
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(filePath, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	return filePath
}
