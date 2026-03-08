package runtime

import (
	"context"
	"time"

	"go.uber.org/zap"
	"mannaiah/module/woocommerce/port"
)

// resolveContext resolves nil contexts to background defaults.
func resolveContext(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}

	return context.Background()
}

// resolvePublisher resolves optional integration event publisher dependencies.
func resolvePublisher(publishers []port.IntegrationEventPublisher) port.IntegrationEventPublisher {
	if len(publishers) == 0 {
		return nil
	}

	return publishers[0]
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

// resolveSyncTimeout resolves cron sync execution timeout values.
func resolveSyncTimeout(timeoutMS int) time.Duration {
	if timeoutMS <= 0 {
		return 10 * time.Minute
	}

	return time.Duration(timeoutMS) * time.Millisecond
}

// resolveRequestTimeout resolves WooCommerce request timeout values in milliseconds.
func resolveRequestTimeout(timeoutMS int) int {
	if timeoutMS <= 0 {
		return 5000
	}

	return timeoutMS
}
