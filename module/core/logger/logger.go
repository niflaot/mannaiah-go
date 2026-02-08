package logger

import (
	"fmt"
	"io"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type encoderFactory func(zapcore.EncoderConfig) zapcore.Encoder

// Settings defines logger configuration values loaded from configuration sources.
type Settings struct {
	// Format selects the encoder format. Supported values are "pretty" and "json".
	Format string `mapstructure:"LOG_FORMAT" default:"pretty"`
	// Level defines the minimum enabled log level.
	Level string `mapstructure:"LOG_LEVEL" default:"info"`
}

// New builds a logger from settings and writes logs to stdout/stderr.
func New(settings Settings) (*zap.Logger, error) {
	return NewWithWriters(settings, os.Stdout, os.Stderr)
}

// Resolve returns the provided logger when present, otherwise creates a new configured logger.
func Resolve(provided *zap.Logger, settings Settings) (*zap.Logger, error) {
	if provided != nil {
		return provided, nil
	}

	return New(settings)
}

// NewWithWriters builds a logger using custom output writers.
func NewWithWriters(settings Settings, output io.Writer, errorOutput io.Writer) (*zap.Logger, error) {
	level, err := zapcore.ParseLevel(strings.ToLower(strings.TrimSpace(settings.Level)))
	if err != nil {
		return nil, fmt.Errorf("invalid LOG_LEVEL %q: %w", settings.Level, err)
	}

	factory, err := selectEncoderFactory(settings.Format)
	if err != nil {
		return nil, err
	}

	core := zapcore.NewCore(
		factory(defaultEncoderConfig()),
		zapcore.Lock(zapcore.AddSync(output)),
		level,
	)

	opts := []zap.Option{}
	if errorOutput != nil {
		opts = append(opts, zap.ErrorOutput(zapcore.AddSync(errorOutput)))
	}

	return zap.New(core, opts...), nil
}

// defaultEncoderConfig defines the shared encoder keys and value encoders.
func defaultEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

// selectEncoderFactory maps configured format strings to Zap encoder factories.
func selectEncoderFactory(format string) (encoderFactory, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "pretty", "console":
		return zapcore.NewConsoleEncoder, nil
	case "json":
		return zapcore.NewJSONEncoder, nil
	default:
		return nil, fmt.Errorf("invalid LOG_FORMAT %q: supported values are pretty or json", format)
	}
}
