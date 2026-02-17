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

// resolveValidationTimeout resolves startup validation timeout values.
func resolveValidationTimeout(timeoutMS int) time.Duration {
	if timeoutMS <= 0 {
		return 3 * time.Second
	}

	return time.Duration(timeoutMS) * time.Millisecond
}

// resolveRequestTimeout resolves Falabella request timeout values.
func resolveRequestTimeout(timeoutMS int) time.Duration {
	if timeoutMS <= 0 {
		return 5 * time.Second
	}

	return time.Duration(timeoutMS) * time.Millisecond
}
