package runtime

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// resolveContext resolves nil contexts to background defaults.
func resolveContext(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}

	return context.Background()
}

// resolveLogger resolves nil loggers to no-op defaults.
func resolveLogger(providedLogger *zap.Logger) *zap.Logger {
	if providedLogger != nil {
		return providedLogger
	}

	return zap.NewNop()
}

// resolveSyncTimeout resolves sync timeout values from millisecond config inputs.
func resolveSyncTimeout(timeoutMS int) time.Duration {
	if timeoutMS <= 0 {
		return 10 * time.Minute
	}

	return time.Duration(timeoutMS) * time.Millisecond
}

// resolveRequestTimeout resolves request timeout values from millisecond config inputs.
func resolveRequestTimeout(timeoutMS int) time.Duration {
	if timeoutMS <= 0 {
		return 5 * time.Second
	}

	return time.Duration(timeoutMS) * time.Millisecond
}

// resolveDurationMS resolves optional millisecond config inputs.
func resolveDurationMS(valueMS int) time.Duration {
	if valueMS <= 0 {
		return 0
	}

	return time.Duration(valueMS) * time.Millisecond
}

// resolveSyncWorkers resolves webhook worker counts.
func resolveSyncWorkers(workers int) int {
	if workers <= 0 {
		return 1
	}

	return workers
}
