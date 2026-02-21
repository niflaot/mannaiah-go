package database

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gosqlmysql "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

// logProbeModel is a minimal model used by logging tests.
type logProbeModel struct {
	// Model provides ID/timestamps/soft-delete fields.
	Model
	// Name is the persisted sample field.
	Name string
}

// TestOpenSQLiteSuccess verifies SQLite initialization and basic connectivity.
func TestOpenSQLiteSuccess(t *testing.T) {
	db, err := Open(
		Config{
			Driver: "sqlite",
			DSN:    "file::memory:?cache=shared",
		},
		nil,
	)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("DB() error = %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	if pingErr := sqlDB.Ping(); pingErr != nil {
		t.Fatalf("Ping() error = %v", pingErr)
	}
}

// TestOpenAppliesPoolConfig verifies Open applies pool and logger configuration branches.
func TestOpenAppliesPoolConfig(t *testing.T) {
	db, err := Open(
		Config{
			Driver:               "sqlite",
			DSN:                  "file::memory:?cache=shared",
			MaxOpenConns:         7,
			MaxIdleConns:         3,
			ConnMaxLifetimeMS:    1000,
			ConnMaxIdleTimeMS:    1000,
			GormLogLevel:         "info",
			SlowQueryThresholdMS: 1,
		},
		zap.NewNop(),
	)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("DB() error = %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
}

// TestOpenReturnsDialectorError verifies dialector-open failures are wrapped and returned.
func TestOpenReturnsDialectorError(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "missing-dir", "db.sqlite")
	_, err := Open(
		Config{
			Driver: "sqlite",
			DSN:    fmt.Sprintf("file:%s?mode=rw", missingPath),
		},
		nil,
	)
	if err == nil {
		t.Fatalf("expected Open() to fail for sqlite mode=rw missing file")
	}
	if !strings.Contains(err.Error(), "open database") {
		t.Fatalf("expected wrapped open error, got %q", err.Error())
	}
}

// TestOpenUnsupportedDriver verifies invalid DB_DRIVER values are rejected.
func TestOpenUnsupportedDriver(t *testing.T) {
	_, err := Open(
		Config{
			Driver: "oracle",
			DSN:    "dsn",
		},
		nil,
	)
	if !errors.Is(err, ErrUnsupportedDriver) {
		t.Fatalf("Open() error = %v, want ErrUnsupportedDriver", err)
	}
}

// TestResolveDialectorSupportsAliases verifies driver alias support for postgres and mysql.
func TestResolveDialectorSupportsAliases(t *testing.T) {
	if _, err := resolveDialector("postgresql", "host=localhost user=x"); err != nil {
		t.Fatalf("resolveDialector(postgresql) error = %v", err)
	}
	if _, err := resolveDialector("mysql", "user:pass@tcp(localhost:3306)/db"); err != nil {
		t.Fatalf("resolveDialector(mysql) error = %v", err)
	}
}

// TestNormalizeMySQLDSNEnablesMultiStatements verifies MySQL DSN normalization forces multiStatements=true.
func TestNormalizeMySQLDSNEnablesMultiStatements(t *testing.T) {
	normalized, err := normalizeMySQLDSN("user:pass@tcp(localhost:3306)/db?parseTime=true")
	if err != nil {
		t.Fatalf("normalizeMySQLDSN() error = %v", err)
	}

	cfg, err := gosqlmysql.ParseDSN(normalized)
	if err != nil {
		t.Fatalf("ParseDSN(normalized) error = %v", err)
	}
	if !cfg.MultiStatements {
		t.Fatalf("MultiStatements = %t, want true", cfg.MultiStatements)
	}
	if !cfg.ParseTime {
		t.Fatalf("ParseTime = %t, want true", cfg.ParseTime)
	}
}

// TestNormalizeMySQLDSNRejectsInvalid verifies invalid mysql DSN values are rejected.
func TestNormalizeMySQLDSNRejectsInvalid(t *testing.T) {
	_, err := normalizeMySQLDSN("not-a-valid-dsn")
	if err == nil {
		t.Fatalf("expected normalizeMySQLDSN() to fail for invalid DSN")
	}
	if !strings.Contains(err.Error(), "parse mysql dsn") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestResolveGormLogLevel verifies GORM level mapping behavior.
func TestResolveGormLogLevel(t *testing.T) {
	cases := map[string]glogger.LogLevel{
		"silent":  glogger.Silent,
		"error":   glogger.Error,
		"warn":    glogger.Warn,
		"warning": glogger.Warn,
		"info":    glogger.Info,
		"invalid": glogger.Warn,
	}

	for input, want := range cases {
		got := resolveGormLogLevel(input)
		if got != want {
			t.Fatalf("resolveGormLogLevel(%q) = %v, want %v", input, got, want)
		}
	}
}

// TestGormZapLoggerLogMode verifies logger cloning with updated level.
func TestGormZapLoggerLogMode(t *testing.T) {
	logger := newGormZapLogger(zap.NewNop(), "warn", time.Second)
	updated := logger.LogMode(glogger.Info)
	probe, ok := updated.(*gormZapLogger)
	if !ok {
		t.Fatalf("expected *gormZapLogger, got %T", updated)
	}
	if probe.level != glogger.Info {
		t.Fatalf("LogMode level = %v, want %v", probe.level, glogger.Info)
	}
}

// TestGormZapLoggerTraceLogsSlowQuery verifies slow query warning logging.
func TestGormZapLoggerTraceLogsSlowQuery(t *testing.T) {
	var out bytes.Buffer
	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(&out), zapcore.DebugLevel)
	logger := zap.New(core)
	gormLogger := newGormZapLogger(logger, "warn", time.Millisecond)

	gormLogger.Trace(context.Background(), time.Now().Add(-20*time.Millisecond), func() (string, int64) {
		return "SELECT 1", 1
	}, nil)

	if !strings.Contains(out.String(), "gorm slow query") {
		t.Fatalf("expected slow query log, got %q", out.String())
	}
}

// TestGormZapLoggerTraceLogsError verifies query error logging.
func TestGormZapLoggerTraceLogsError(t *testing.T) {
	var out bytes.Buffer
	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(&out), zapcore.DebugLevel)
	logger := zap.New(core)
	gormLogger := newGormZapLogger(logger, "error", time.Second)

	gormLogger.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "SELECT broken", -1
	}, errors.New("query failed"))

	if !strings.Contains(out.String(), "gorm query failed") {
		t.Fatalf("expected query failed log, got %q", out.String())
	}
}

// TestGormZapLoggerTraceIgnoresNotFound verifies record-not-found errors are not logged as failures.
func TestGormZapLoggerTraceIgnoresNotFound(t *testing.T) {
	var out bytes.Buffer
	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(&out), zapcore.DebugLevel)
	logger := zap.New(core)
	gormLogger := newGormZapLogger(logger, "error", time.Second)

	gormLogger.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "SELECT", 0
	}, gorm.ErrRecordNotFound)

	if out.Len() != 0 {
		t.Fatalf("expected no log output for record-not-found trace, got %q", out.String())
	}
}

// TestGormZapLoggerInfoWarnError verifies direct Info/Warn/Error methods emit logs by level.
func TestGormZapLoggerInfoWarnError(t *testing.T) {
	var out bytes.Buffer
	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(&out), zapcore.DebugLevel)
	logger := zap.New(core)
	gormLogger := newGormZapLogger(logger, "info", time.Second)

	gormLogger.Info(context.Background(), "info %s", "message")
	gormLogger.Warn(context.Background(), "warn %s", "message")
	gormLogger.Error(context.Background(), "error %s", "message")

	payload := out.String()
	if !strings.Contains(payload, "info message") {
		t.Fatalf("expected info log payload, got %q", payload)
	}
	if !strings.Contains(payload, "warn message") {
		t.Fatalf("expected warn log payload, got %q", payload)
	}
	if !strings.Contains(payload, "error message") {
		t.Fatalf("expected error log payload, got %q", payload)
	}
}

// TestGormZapLoggerTraceInfoPath verifies successful trace logging when level is info.
func TestGormZapLoggerTraceInfoPath(t *testing.T) {
	var out bytes.Buffer
	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(&out), zapcore.DebugLevel)
	logger := zap.New(core)
	gormLogger := newGormZapLogger(logger, "info", time.Second)

	gormLogger.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "SELECT 42", 1
	}, nil)

	if !strings.Contains(out.String(), "gorm query") {
		t.Fatalf("expected info trace log, got %q", out.String())
	}
}

// TestGormZapLoggerLevelFiltering verifies logger methods honor configured level thresholds.
func TestGormZapLoggerLevelFiltering(t *testing.T) {
	var out bytes.Buffer
	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(&out), zapcore.DebugLevel)
	logger := zap.New(core)

	errorOnly := newGormZapLogger(logger, "error", time.Second)
	errorOnly.Info(context.Background(), "info hidden")
	errorOnly.Warn(context.Background(), "warn hidden")
	if out.Len() != 0 {
		t.Fatalf("expected no info/warn output at error level, got %q", out.String())
	}

	silent := newGormZapLogger(logger, "silent", time.Second)
	silent.Error(context.Background(), "error hidden")
	if strings.Contains(out.String(), "error hidden") {
		t.Fatalf("expected error helper output to be suppressed at silent level, got %q", out.String())
	}
}

// TestGormZapLoggerTraceSilent verifies silent mode suppresses trace logs.
func TestGormZapLoggerTraceSilent(t *testing.T) {
	var out bytes.Buffer
	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(&out), zapcore.DebugLevel)
	logger := zap.New(core)
	gormLogger := newGormZapLogger(logger, "silent", time.Nanosecond)

	gormLogger.Trace(context.Background(), time.Now().Add(-time.Second), func() (string, int64) {
		return "SELECT silent", 1
	}, nil)

	if out.Len() != 0 {
		t.Fatalf("expected no output at silent trace level, got %q", out.String())
	}
}
