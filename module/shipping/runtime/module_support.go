package runtime

import (
	"time"

	"go.uber.org/zap"
)

// resolveLogger resolves nil loggers to no-op defaults.
func resolveLogger(providedLogger *zap.Logger) *zap.Logger {
	if providedLogger != nil {
		return providedLogger
	}

	return zap.NewNop()
}

// resolveRequestTimeout resolves request timeout values from milliseconds.
func resolveRequestTimeout(timeoutMS int) time.Duration {
	if timeoutMS <= 0 {
		return 5 * time.Second
	}

	return time.Duration(timeoutMS) * time.Millisecond
}
