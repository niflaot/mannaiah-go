package middleware

import (
	"time"

	"github.com/ThreeDotsLabs/watermill"
	wmmsg "github.com/ThreeDotsLabs/watermill/message"
	wmmiddleware "github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"mannaiah/module/core/messaging/bus"
	"mannaiah/module/core/messaging/platform"
	coretelemetry "mannaiah/module/core/telemetry"
)

// AddRouterMiddlewares adds router-level middleware in the required execution order.
func AddRouterMiddlewares(router *wmmsg.Router) {
	router.AddMiddleware(
		Correlation,
	)
}

// Correlation ensures incoming messages always carry a correlation id.
func Correlation(next wmmsg.HandlerFunc) wmmsg.HandlerFunc {
	return func(message *wmmsg.Message) ([]*wmmsg.Message, error) {
		correlationID := message.Metadata.Get(bus.MetadataCorrelationID)
		if correlationID == "" {
			wmmiddleware.SetCorrelationID(watermill.NewUUID(), message)
		}

		producedMessages, err := next(message)
		correlationID = message.Metadata.Get(bus.MetadataCorrelationID)
		for _, produced := range producedMessages {
			if produced.Metadata.Get(bus.MetadataCorrelationID) == "" {
				wmmiddleware.SetCorrelationID(correlationID, produced)
			}
		}

		return producedMessages, err
	}
}

// ShouldRetry classifies retry decisions for middleware retries.
func ShouldRetry(params wmmiddleware.RetryParams) bool {
	retriable := !platform.IsNonRetriable(params.Err)
	if retriable {
		coretelemetry.RecordMessaging("unknown", "retry", time.Now(), nil)
	}

	return retriable
}

// NewRetry creates retry middleware from normalized platform configuration.
func NewRetry(cfg platform.Config, logger watermill.LoggerAdapter) wmmsg.HandlerMiddleware {
	normalizedCfg := cfg.Normalized()
	retry := wmmiddleware.Retry{
		MaxRetries:          normalizedCfg.RetryMaxRetries,
		InitialInterval:     time.Duration(normalizedCfg.RetryInitialIntervalMS) * time.Millisecond,
		MaxInterval:         time.Duration(normalizedCfg.RetryMaxIntervalMS) * time.Millisecond,
		Multiplier:          normalizedCfg.RetryMultiplier,
		Logger:              logger,
		ShouldRetry:         ShouldRetry,
		ResetContextOnRetry: true,
	}

	return retry.Middleware
}
