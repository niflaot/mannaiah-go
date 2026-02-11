package logger

import (
	"fmt"
	"strings"

	wmlog "github.com/ThreeDotsLabs/watermill"
	"go.uber.org/zap"
)

// zapAdapter adapts Zap loggers to Watermill logger contracts.
type zapAdapter struct {
	// logger is the wrapped Zap logger instance.
	logger *zap.Logger
}

// NewZapAdapter creates a Watermill logger adapter from Zap.
func NewZapAdapter(providedLogger *zap.Logger) wmlog.LoggerAdapter {
	resolvedLogger := providedLogger
	if resolvedLogger == nil {
		resolvedLogger = zap.NewNop()
	}

	return &zapAdapter{
		logger: resolvedLogger,
	}
}

// Error logs Watermill error-level events.
func (a *zapAdapter) Error(message string, err error, fields wmlog.LogFields) {
	entries := append(logFieldsToZap(fields), zap.Error(err))
	a.logger.Error(message, entries...)
}

// Info logs Watermill info-level events.
func (a *zapAdapter) Info(message string, fields wmlog.LogFields) {
	if isNoSubscribersMessage(message) {
		a.logger.Debug(message, logFieldsToZap(fields)...)
		return
	}

	a.logger.Info(message, logFieldsToZap(fields)...)
}

// Debug logs Watermill debug-level events.
func (a *zapAdapter) Debug(message string, fields wmlog.LogFields) {
	a.logger.Debug(message, logFieldsToZap(fields)...)
}

// Trace logs Watermill trace-level events using debug level.
func (a *zapAdapter) Trace(message string, fields wmlog.LogFields) {
	a.logger.Debug(message, logFieldsToZap(fields)...)
}

// With returns a logger adapter enriched with static fields.
func (a *zapAdapter) With(fields wmlog.LogFields) wmlog.LoggerAdapter {
	return &zapAdapter{
		logger: a.logger.With(logFieldsToZap(fields)...),
	}
}

// logFieldsToZap converts Watermill log fields into Zap fields.
func logFieldsToZap(fields wmlog.LogFields) []zap.Field {
	if len(fields) == 0 {
		return nil
	}

	converted := make([]zap.Field, 0, len(fields))
	for key, value := range fields {
		converted = append(converted, zap.Any(key, normalizeLogValue(value)))
	}

	return converted
}

// normalizeLogValue normalizes field values into loggable primitives where possible.
func normalizeLogValue(value any) any {
	if value == nil {
		return nil
	}

	return fmt.Sprintf("%v", value)
}

// isNoSubscribersMessage reports Watermill pubsub no-subscriber information messages.
func isNoSubscribersMessage(message string) bool {
	return strings.EqualFold(strings.TrimSpace(message), "No subscribers to send message")
}
