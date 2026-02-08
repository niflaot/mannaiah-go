package config

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"go.uber.org/zap"
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

// autoTagProbe verifies derived keys when mapstructure tags are omitted.
type autoTagProbe struct {
	// HostName is decoded using its derived key HOST_NAME.
	HostName string
}

// invalidDecodeProbe verifies unmarshal errors for incompatible decoded values.
type invalidDecodeProbe struct {
	// Values expects a map but receives a scalar configuration value.
	Values map[string]int `mapstructure:"UT_INVALID_VALUES"`
}

// helperNestedProbe verifies nested bind/default/required behavior.
type helperNestedProbe struct {
	// Required is mandatory because it has no default.
	Required string `mapstructure:"REQUIRED"`
	// Defaulted should be initialized from a default tag.
	Defaulted int `mapstructure:"DEFAULTED" default:"10"`
}

// helperContainerProbe verifies recursion and squash behavior.
type helperContainerProbe struct {
	// Nested uses a non-squashed key prefix.
	Nested helperNestedProbe `mapstructure:"NESTED"`
	// Flat uses squash and should not use a prefix.
	Flat helperNestedProbe `mapstructure:",squash"`
	// AutoName has no mapstructure tag and should use derived key naming.
	AutoName string `default:"auto"`
	// Ignored is skipped by mapstructure binding logic.
	Ignored string `mapstructure:"-"`
	// hiddenValue is unexported and must be ignored by reflection traversal.
	hiddenValue string `mapstructure:"HIDDEN_VALUE"`
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

// TestValidationErrorError formats missing entries in a startup-friendly message.
func TestValidationErrorError(t *testing.T) {
	err := ValidationError{
		Missing: []MissingFieldError{
			{Field: "Core.Host", Key: "CORE_HOST"},
			{Field: "Core.Port", Key: "CORE_PORT"},
		},
	}

	want := "missing required configuration values: Core.Host (CORE_HOST) Core.Port (CORE_PORT)"
	got := err.Error()
	if got != want {
		t.Fatalf("ValidationError.Error() = %q, want %q", got, want)
	}
}

// TestLoadConvenienceWrapper verifies the package-level Load helper delegates correctly.
func TestLoadConvenienceWrapper(t *testing.T) {
	envFile := filepath.Join(t.TempDir(), ".env.missing")

	var cfg defaultsProbe
	if err := Load(envFile, nil, &cfg); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Name != "mannaiah" {
		t.Fatalf("Name = %q, want %q", cfg.Name, "mannaiah")
	}
}

// TestLoaderLoadRejectsEmptyTargets verifies loader returns an error when called without targets.
func TestLoaderLoadRejectsEmptyTargets(t *testing.T) {
	loader := NewLoader(filepath.Join(t.TempDir(), ".env.missing"), nil)
	if err := loader.Load(); err == nil {
		t.Fatalf("expected error when no target structs are provided")
	}
}

// TestLoaderLoadRejectsNilTarget verifies nil target entries are rejected during startup validation.
func TestLoaderLoadRejectsNilTarget(t *testing.T) {
	loader := NewLoader(filepath.Join(t.TempDir(), ".env.missing"), nil)
	if err := loader.Load(nil); err == nil {
		t.Fatalf("expected error when target entry is nil")
	}
}

// TestLoaderLoadRejectsInvalidDotEnv verifies parse failures in .env content are returned.
func TestLoaderLoadRejectsInvalidDotEnv(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(filePath, []byte("INVALID_LINE\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var cfg defaultsProbe
	err := NewLoader(filePath, nil).Load(&cfg)
	if err == nil {
		t.Fatalf("expected parse error for invalid .env syntax")
	}
	if !strings.Contains(err.Error(), "read .env file") {
		t.Fatalf("expected .env read context in error, got %q", err.Error())
	}
}

// TestLoaderLoadReturnsUnmarshalError verifies incompatible decoded values fail startup.
func TestLoaderLoadReturnsUnmarshalError(t *testing.T) {
	envFile := writeEnvFile(t, "UT_INVALID_VALUES=plain-text")

	var cfg invalidDecodeProbe
	err := NewLoader(envFile, nil).Load(&cfg)
	if err == nil {
		t.Fatalf("expected unmarshal error for incompatible target type")
	}
	if !strings.Contains(err.Error(), "unmarshal config into") {
		t.Fatalf("expected unmarshal context in error, got %q", err.Error())
	}
}

// TestIsMissingConfigFile verifies both missing-file detection branches and negative cases.
func TestIsMissingConfigFile(t *testing.T) {
	if !isMissingConfigFile(viper.ConfigFileNotFoundError{}) {
		t.Fatalf("expected ConfigFileNotFoundError to be treated as missing file")
	}
	if !isMissingConfigFile(os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist to be treated as missing file")
	}
	if isMissingConfigFile(errors.New("different error")) {
		t.Fatalf("expected unrelated errors not to be treated as missing files")
	}
}

// TestResolveTargetStructTypeNilTarget verifies nil interface targets are rejected.
func TestResolveTargetStructTypeNilTarget(t *testing.T) {
	var target any
	_, err := resolveTargetStructType(target)
	if err == nil {
		t.Fatalf("expected nil target to fail validation")
	}
}

// TestNewLoaderDefaultsAndOverrides verifies loader normalization for .env path and startup logger injection.
func TestNewLoaderDefaultsAndOverrides(t *testing.T) {
	defaultLoader := NewLoader("", nil)
	if defaultLoader.envFile != ".env" {
		t.Fatalf("default envFile = %q, want %q", defaultLoader.envFile, ".env")
	}
	if defaultLoader.logger == nil {
		t.Fatalf("expected default startup logger instance")
	}

	providedLogger := zap.NewNop()
	customLoader := NewLoader("  custom.env  ", providedLogger)
	if customLoader.envFile != "custom.env" {
		t.Fatalf("custom envFile = %q, want %q", customLoader.envFile, "custom.env")
	}
	if customLoader.logger != providedLogger {
		t.Fatalf("expected NewLoader() to preserve provided logger instance")
	}
}

// TestRegisterBindingsAndDefaultsAndCollectMissing verifies recursion, squash, defaults, and required checks.
func TestRegisterBindingsAndDefaultsAndCollectMissing(t *testing.T) {
	v := viper.New()
	structType := reflect.TypeOf(helperContainerProbe{})

	if err := registerBindingsAndDefaults(v, structType, ""); err != nil {
		t.Fatalf("registerBindingsAndDefaults() error = %v", err)
	}

	if got := v.GetInt("NESTED.DEFAULTED"); got != 10 {
		t.Fatalf("NESTED.DEFAULTED = %d, want %d", got, 10)
	}
	if got := v.GetInt("DEFAULTED"); got != 10 {
		t.Fatalf("DEFAULTED = %d, want %d", got, 10)
	}
	if got := v.GetString("AUTO_NAME"); got != "auto" {
		t.Fatalf("AUTO_NAME = %q, want %q", got, "auto")
	}

	missing := collectMissing(v, structType, "", "helperContainerProbe")
	if len(missing) != 2 {
		t.Fatalf("missing count = %d, want %d", len(missing), 2)
	}

	v.Set("NESTED.REQUIRED", "nested-ok")
	v.Set("REQUIRED", "flat-ok")
	missing = collectMissing(v, structType, "", "helperContainerProbe")
	if len(missing) != 0 {
		t.Fatalf("expected all required keys satisfied, got %d missing entries", len(missing))
	}
}

// TestParseMapstructureTag verifies tag parsing for empty, skip, and squash variants.
func TestParseMapstructureTag(t *testing.T) {
	empty := parseMapstructureTag("")
	if empty.name != "" || empty.skip || empty.squash {
		t.Fatalf("unexpected parsed empty tag: %+v", empty)
	}

	skip := parseMapstructureTag("-")
	if !skip.skip {
		t.Fatalf("expected skip option for '-' tag")
	}

	complex := parseMapstructureTag("FIELD_NAME,squash")
	if complex.name != "FIELD_NAME" || !complex.squash || complex.skip {
		t.Fatalf("unexpected parsed complex tag: %+v", complex)
	}
}

// TestResolveFieldKeyPartAndJoinHelpers verifies key derivation and join helper edge cases.
func TestResolveFieldKeyPartAndJoinHelpers(t *testing.T) {
	if got := resolveFieldKeyPart("HostName", ""); got != "HOST_NAME" {
		t.Fatalf("resolveFieldKeyPart() = %q, want %q", got, "HOST_NAME")
	}
	if got := resolveFieldKeyPart("HostName", "CUSTOM_NAME"); got != "CUSTOM_NAME" {
		t.Fatalf("resolveFieldKeyPart() = %q, want %q", got, "CUSTOM_NAME")
	}

	if got := joinKey("", "CHILD"); got != "CHILD" {
		t.Fatalf("joinKey() = %q, want %q", got, "CHILD")
	}
	if got := joinKey("PARENT", ""); got != "PARENT" {
		t.Fatalf("joinKey() = %q, want %q", got, "PARENT")
	}
	if got := joinKey("PARENT", "CHILD"); got != "PARENT.CHILD" {
		t.Fatalf("joinKey() = %q, want %q", got, "PARENT.CHILD")
	}

	if got := joinPath("", "Child"); got != "Child" {
		t.Fatalf("joinPath() = %q, want %q", got, "Child")
	}
	if got := joinPath("Parent", "Child"); got != "Parent.Child" {
		t.Fatalf("joinPath() = %q, want %q", got, "Parent.Child")
	}
}

// TestParseDefaultValue verifies type conversion behavior for supported and fallback types.
func TestParseDefaultValue(t *testing.T) {
	type parseCase struct {
		// Name identifies the case for failure output.
		Name string
		// Raw is the string input from struct tag defaults.
		Raw string
		// Type is the target field type.
		Type reflect.Type
		// Want is the expected parsed result.
		Want any
	}

	cases := []parseCase{
		{Name: "string", Raw: "value", Type: reflect.TypeOf(""), Want: "value"},
		{Name: "bool-true", Raw: "true", Type: reflect.TypeOf(false), Want: true},
		{Name: "bool-invalid", Raw: "not-bool", Type: reflect.TypeOf(false), Want: "not-bool"},
		{Name: "int", Raw: "7", Type: reflect.TypeOf(int(0)), Want: int64(7)},
		{Name: "int-invalid", Raw: "bad", Type: reflect.TypeOf(int(0)), Want: "bad"},
		{Name: "uint", Raw: "9", Type: reflect.TypeOf(uint(0)), Want: uint64(9)},
		{Name: "uint-invalid", Raw: "-1", Type: reflect.TypeOf(uint(0)), Want: "-1"},
		{Name: "float", Raw: "1.5", Type: reflect.TypeOf(float64(0)), Want: float64(1.5)},
		{Name: "float-invalid", Raw: "pi", Type: reflect.TypeOf(float64(0)), Want: "pi"},
		{Name: "slice-string", Raw: "a, b, c", Type: reflect.TypeOf([]string{}), Want: []string{"a", "b", "c"}},
		{Name: "slice-non-string", Raw: "1,2", Type: reflect.TypeOf([]int{}), Want: "1,2"},
	}

	for _, tc := range cases {
		got := parseDefaultValue(tc.Raw, tc.Type)
		if !reflect.DeepEqual(got, tc.Want) {
			t.Fatalf("%s: parseDefaultValue() = %#v, want %#v", tc.Name, got, tc.Want)
		}
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
