package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// MissingFieldError describes a required configuration field that was not found.
type MissingFieldError struct {
	// Field is the dot-path of the target struct field.
	Field string
	// Key is the configuration key expected by viper.
	Key string
}

// ValidationError aggregates required configuration failures.
type ValidationError struct {
	// Missing contains all missing required fields.
	Missing []MissingFieldError
}

// Error returns a startup-friendly validation summary.
func (e ValidationError) Error() string {
	var builder strings.Builder
	builder.WriteString("missing required configuration values:")
	for _, missing := range e.Missing {
		builder.WriteString(" ")
		builder.WriteString(missing.Field)
		builder.WriteString(" (")
		builder.WriteString(missing.Key)
		builder.WriteString(")")
	}

	return builder.String()
}

// Loader loads and validates configuration structs using .env and environment variables.
type Loader struct {
	// envFile is the .env file path used as the initial source.
	envFile string
	// logger receives startup validation errors.
	logger *zap.Logger
}

// mapstructureTag represents parsed options from the mapstructure struct tag.
type mapstructureTag struct {
	// name is the explicit key name from the tag.
	name string
	// squash indicates embedded struct flattening.
	squash bool
	// skip indicates the field must be ignored.
	skip bool
}

// NewLoader creates a Loader with optional .env path and startup logger.
func NewLoader(envFile string, startupLogger *zap.Logger) *Loader {
	resolvedEnvFile := strings.TrimSpace(envFile)
	if resolvedEnvFile == "" {
		resolvedEnvFile = ".env"
	}

	resolvedLogger := startupLogger
	if resolvedLogger == nil {
		resolvedLogger = zap.NewNop()
	}

	return &Loader{
		envFile: resolvedEnvFile,
		logger:  resolvedLogger,
	}
}

// Load is a convenience wrapper that creates a loader and fills all targets.
func Load(envFile string, startupLogger *zap.Logger, targets ...any) error {
	return NewLoader(envFile, startupLogger).Load(targets...)
}

// Load fills target config structs from .env and environment variables, then validates required fields.
func (l *Loader) Load(targets ...any) error {
	if len(targets) == 0 {
		return errors.New("at least one target config struct is required")
	}

	v := viper.New()
	v.SetConfigFile(l.envFile)
	v.SetConfigType("env")
	if err := v.ReadInConfig(); err != nil && !isMissingConfigFile(err) {
		return fmt.Errorf("read .env file %q: %w", l.envFile, err)
	}

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	missing := []MissingFieldError{}

	for _, target := range targets {
		targetType, err := resolveTargetStructType(target)
		if err != nil {
			return err
		}

		if err := registerBindingsAndDefaults(v, targetType, ""); err != nil {
			return err
		}
		if err := v.Unmarshal(target); err != nil {
			return fmt.Errorf("unmarshal config into %s: %w", targetType.String(), err)
		}

		missing = append(missing, collectMissing(v, targetType, "", targetType.Name())...)
	}

	if len(missing) > 0 {
		for _, item := range missing {
			l.logger.Error(
				"required configuration value missing",
				zap.String("field", item.Field),
				zap.String("key", item.Key),
			)
		}

		return ValidationError{Missing: missing}
	}

	return nil
}

// isMissingConfigFile returns true when the .env source does not exist.
func isMissingConfigFile(err error) bool {
	var cfgMissing viper.ConfigFileNotFoundError
	if errors.As(err, &cfgMissing) {
		return true
	}

	return errors.Is(err, os.ErrNotExist)
}

// resolveTargetStructType validates a target and returns its underlying struct type.
func resolveTargetStructType(target any) (reflect.Type, error) {
	targetType := reflect.TypeOf(target)
	if targetType == nil {
		return nil, errors.New("nil target config provided")
	}
	if targetType.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("target %s must be a pointer to struct", targetType.String())
	}

	structType := dereferenceType(targetType)
	if structType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("target %s must point to struct", targetType.String())
	}

	return structType, nil
}

// registerBindingsAndDefaults binds environment keys and registers default values from struct tags.
func registerBindingsAndDefaults(v *viper.Viper, structType reflect.Type, parentKey string) error {
	for index := 0; index < structType.NumField(); index++ {
		field := structType.Field(index)
		if field.PkgPath != "" {
			continue
		}

		tag := parseMapstructureTag(field.Tag.Get("mapstructure"))
		if tag.skip {
			continue
		}

		fieldType := dereferenceType(field.Type)
		if fieldType.Kind() == reflect.Struct {
			nextParent := parentKey
			if !tag.squash {
				nextParent = joinKey(parentKey, resolveFieldKeyPart(field.Name, tag.name))
			}

			if err := registerBindingsAndDefaults(v, fieldType, nextParent); err != nil {
				return err
			}
			continue
		}

		key := joinKey(parentKey, resolveFieldKeyPart(field.Name, tag.name))
		if err := v.BindEnv(key); err != nil {
			return fmt.Errorf("bind environment key %q: %w", key, err)
		}

		defaultValue, hasDefault := field.Tag.Lookup("default")
		if !hasDefault {
			continue
		}

		v.SetDefault(key, parseDefaultValue(defaultValue, fieldType))
	}

	return nil
}

// collectMissing discovers required fields that have neither explicit values nor defaults.
func collectMissing(v *viper.Viper, structType reflect.Type, parentKey string, parentPath string) []MissingFieldError {
	missing := []MissingFieldError{}

	for index := 0; index < structType.NumField(); index++ {
		field := structType.Field(index)
		if field.PkgPath != "" {
			continue
		}

		tag := parseMapstructureTag(field.Tag.Get("mapstructure"))
		if tag.skip {
			continue
		}

		fieldType := dereferenceType(field.Type)
		fieldPath := joinPath(parentPath, field.Name)
		if fieldType.Kind() == reflect.Struct {
			nextParent := parentKey
			if !tag.squash {
				nextParent = joinKey(parentKey, resolveFieldKeyPart(field.Name, tag.name))
			}

			missing = append(missing, collectMissing(v, fieldType, nextParent, fieldPath)...)
			continue
		}

		_, hasDefault := field.Tag.Lookup("default")
		if hasDefault {
			continue
		}

		key := joinKey(parentKey, resolveFieldKeyPart(field.Name, tag.name))
		if !v.IsSet(key) {
			missing = append(missing, MissingFieldError{
				Field: fieldPath,
				Key:   key,
			})
		}
	}

	return missing
}

// parseMapstructureTag parses a mapstructure tag into a structured representation.
func parseMapstructureTag(raw string) mapstructureTag {
	if raw == "" {
		return mapstructureTag{}
	}

	parts := strings.Split(raw, ",")
	tag := mapstructureTag{}
	name := strings.TrimSpace(parts[0])
	if name == "-" {
		tag.skip = true
		return tag
	}
	if name != "" {
		tag.name = name
	}

	for _, part := range parts[1:] {
		if strings.TrimSpace(part) == "squash" {
			tag.squash = true
		}
	}

	return tag
}

// dereferenceType unwraps pointer types until the concrete type is reached.
func dereferenceType(target reflect.Type) reflect.Type {
	current := target
	for current.Kind() == reflect.Ptr {
		current = current.Elem()
	}

	return current
}

// resolveFieldKeyPart resolves the configuration key segment for a struct field.
func resolveFieldKeyPart(fieldName string, configuredName string) string {
	if strings.TrimSpace(configuredName) != "" {
		return configuredName
	}

	return camelToUpperSnake(fieldName)
}

// joinKey joins hierarchical key segments using dot notation.
func joinKey(parent string, child string) string {
	if parent == "" {
		return child
	}
	if child == "" {
		return parent
	}

	return parent + "." + child
}

// joinPath joins hierarchical field path segments.
func joinPath(parent string, child string) string {
	if parent == "" {
		return child
	}

	return parent + "." + child
}

// camelToUpperSnake converts CamelCase identifiers to UPPER_SNAKE_CASE.
func camelToUpperSnake(value string) string {
	var builder strings.Builder
	for index, char := range value {
		if index > 0 && char >= 'A' && char <= 'Z' {
			builder.WriteRune('_')
		}
		builder.WriteRune(char)
	}

	return strings.ToUpper(builder.String())
}

// parseDefaultValue converts default tag values to the expected destination type when possible.
func parseDefaultValue(raw string, targetType reflect.Type) any {
	switch targetType.Kind() {
	case reflect.String:
		return raw
	case reflect.Bool:
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return raw
		}
		return parsed
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return raw
		}
		return parsed
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return raw
		}
		return parsed
	case reflect.Float32, reflect.Float64:
		parsed, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return raw
		}
		return parsed
	case reflect.Slice:
		if targetType.Elem().Kind() == reflect.String {
			parts := strings.Split(raw, ",")
			trimmed := make([]string, 0, len(parts))
			for _, part := range parts {
				trimmed = append(trimmed, strings.TrimSpace(part))
			}
			return trimmed
		}
	}

	return raw
}
