package database

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

// newGormZapLogger builds a GORM logger backed by Zap.
func newGormZapLogger(providedLogger *zap.Logger, level string, slowThreshold time.Duration) glogger.Interface {
	resolvedLogger := providedLogger
	if resolvedLogger == nil {
		resolvedLogger = zap.NewNop()
	}

	return &gormZapLogger{
		logger:        resolvedLogger,
		level:         resolveGormLogLevel(level),
		slowThreshold: slowThreshold,
	}
}

// resolveGormLogLevel maps configuration values to GORM logger levels.
func resolveGormLogLevel(level string) glogger.LogLevel {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "silent":
		return glogger.Silent
	case "error":
		return glogger.Error
	case "warn", "warning":
		return glogger.Warn
	case "info":
		return glogger.Info
	default:
		return glogger.Warn
	}
}

// gormZapLogger adapts Zap to GORM logger.Interface.
type gormZapLogger struct {
	// logger is the underlying Zap logger used for query logs.
	logger *zap.Logger
	// level is the active GORM log level.
	level glogger.LogLevel
	// slowThreshold controls slow query warning logging.
	slowThreshold time.Duration
}

// LogMode clones the logger with the provided log level.
func (l *gormZapLogger) LogMode(level glogger.LogLevel) glogger.Interface {
	return &gormZapLogger{
		logger:        l.logger,
		level:         level,
		slowThreshold: l.slowThreshold,
	}
}

// Info logs informational GORM messages.
func (l *gormZapLogger) Info(_ context.Context, message string, args ...interface{}) {
	if l.level < glogger.Info {
		return
	}

	l.logger.Sugar().Infof(message, args...)
}

// Warn logs warning GORM messages.
func (l *gormZapLogger) Warn(_ context.Context, message string, args ...interface{}) {
	if l.level < glogger.Warn {
		return
	}

	l.logger.Sugar().Warnf(message, args...)
}

// Error logs error GORM messages.
func (l *gormZapLogger) Error(_ context.Context, message string, args ...interface{}) {
	if l.level < glogger.Error {
		return
	}

	l.logger.Sugar().Errorf(message, args...)
}

// Trace logs SQL traces and slow/error query events.
func (l *gormZapLogger) Trace(_ context.Context, begin time.Time, source func() (string, int64), err error) {
	if l.level == glogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := source()
	fields := []zap.Field{
		zap.Duration("elapsed", elapsed),
		zap.String("sql", sql),
		zap.Int64("rows", rows),
	}

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) && l.level >= glogger.Error {
		l.logger.Error("gorm query failed", append(fields, zap.Error(err))...)
		return
	}
	if l.slowThreshold > 0 && elapsed > l.slowThreshold && l.level >= glogger.Warn {
		l.logger.Warn("gorm slow query", fields...)
		return
	}
	if l.level >= glogger.Info {
		l.logger.Info("gorm query", fields...)
	}
}
