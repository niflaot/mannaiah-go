package router

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	wmmsg "github.com/ThreeDotsLabs/watermill/message"
	wmmiddleware "github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"mannaiah/module/core/messaging/bus"
	"mannaiah/module/core/messaging/platform"
	correlationctx "mannaiah/module/core/messaging/watermill/internal/correlation"
	watermilllogger "mannaiah/module/core/messaging/watermill/logger"
	watermillmiddleware "mannaiah/module/core/messaging/watermill/middleware"
	watermillpublisher "mannaiah/module/core/messaging/watermill/publisher"
	watermillpubsub "mannaiah/module/core/messaging/watermill/pubsub"
	coretelemetry "mannaiah/module/core/telemetry"
)

var (
	// ErrNilHandler is returned when a nil handler is provided.
	ErrNilHandler = errors.New("messaging handler must not be nil")
	// ErrEmptyTopic is returned when a topic argument is empty.
	ErrEmptyTopic = errors.New("messaging topic must not be empty")
)

// InMemoryPlatform defines an in-memory Watermill messaging platform wrapper.
type InMemoryPlatform struct {
	// router is the Watermill router instance.
	router *wmmsg.Router
	// pubSub is the in-memory gochannel pubsub instance.
	pubSub pubSub
	// publisher is the abstract publisher adapter.
	publisher bus.Publisher
	// registrar is the abstract handler registrar.
	registrar bus.Registrar
}

// pubSub defines the minimal combined pubsub contract used by this platform.
type pubSub interface {
	wmmsg.Publisher
	wmmsg.Subscriber
	Close() error
}

// NewInMemoryPlatform creates an in-memory messaging platform with Watermill adapters.
func NewInMemoryPlatform(cfg platform.Config, providedLogger *zap.Logger) (*InMemoryPlatform, error) {
	loggerAdapter := watermilllogger.NewZapAdapter(providedLogger)
	router, err := wmmsg.NewRouter(wmmsg.RouterConfig{}, loggerAdapter)
	if err != nil {
		return nil, fmt.Errorf("create messaging router: %w", err)
	}

	pubSub := watermillpubsub.NewGoChannel(cfg, providedLogger)
	publisherAdapter, err := watermillpublisher.NewAdapter(pubSub)
	if err != nil {
		return nil, err
	}

	watermillmiddleware.AddRouterMiddlewares(router)

	registrar := &registrar{
		router:          router,
		subscriber:      pubSub,
		dlqPublisher:    pubSub,
		dlqSuffix:       cfg.Normalized().DLQSuffix,
		retryMiddleware: watermillmiddleware.NewRetry(cfg, loggerAdapter),
	}

	return &InMemoryPlatform{
		router:    router,
		pubSub:    pubSub,
		publisher: publisherAdapter,
		registrar: registrar,
	}, nil
}

// Publisher returns the abstract bus publisher.
func (p *InMemoryPlatform) Publisher() bus.Publisher {
	return p.publisher
}

// Registrar returns the abstract subscription registrar.
func (p *InMemoryPlatform) Registrar() bus.Registrar {
	return p.registrar
}

// Run starts the underlying router lifecycle.
func (p *InMemoryPlatform) Run(ctx context.Context) error {
	return p.router.Run(ctx)
}

// Running returns a channel closed when the router is running.
func (p *InMemoryPlatform) Running() <-chan struct{} {
	return p.router.Running()
}

// Close closes router and pubsub resources.
func (p *InMemoryPlatform) Close() error {
	var closeErr error

	if err := p.router.Close(); err != nil {
		closeErr = errors.Join(closeErr, err)
	}
	if err := p.pubSub.Close(); err != nil {
		closeErr = errors.Join(closeErr, err)
	}

	return closeErr
}

// registrar adapts Watermill routing registration to bus registrar ports.
type registrar struct {
	// router is the Watermill router receiving handlers.
	router *wmmsg.Router
	// subscriber is the message subscriber used for all registrations.
	subscriber wmmsg.Subscriber
	// dlqPublisher is the publisher used for dead-letter forwarding.
	dlqPublisher wmmsg.Publisher
	// dlqSuffix defines dead-letter topic suffix.
	dlqSuffix string
	// retryMiddleware defines handler retry behavior.
	retryMiddleware wmmsg.HandlerMiddleware
	// counter generates stable unique handler names.
	counter uint64
}

var (
	// _ ensures registrar satisfies bus.Registrar.
	_ bus.Registrar = (*registrar)(nil)
)

// AddHandler registers a topic handler through Watermill consumer handlers.
func (r *registrar) AddHandler(topic string, handler bus.Handler) error {
	trimmedTopic := strings.TrimSpace(topic)
	if trimmedTopic == "" {
		return ErrEmptyTopic
	}
	if handler == nil {
		return ErrNilHandler
	}

	handlerName := fmt.Sprintf("bus-handler-%s-%d", sanitizeTopic(trimmedTopic), atomic.AddUint64(&r.counter, 1))
	registeredHandler := r.router.AddConsumerHandler(
		handlerName,
		trimmedTopic,
		r.subscriber,
		func(message *wmmsg.Message) error {
			metadata := make(map[string]string, len(message.Metadata))
			for key, value := range message.Metadata {
				metadata[key] = value
			}

			ctx := correlationctx.WithContext(message.Context(), message.Metadata.Get(bus.MetadataCorrelationID))
			ctx = coretelemetry.ContextWithTraceparent(ctx, message.Metadata.Get(bus.MetadataTraceparent))
			startedAt := time.Now()
			spanCtx, span := coretelemetry.StartSpan(
				ctx,
				"mannaiah/messaging",
				"messaging.consume",
				trace.WithSpanKind(trace.SpanKindConsumer),
				trace.WithAttributes(
					attribute.String("messaging.system", "watermill"),
					attribute.String("messaging.operation", "consume"),
					attribute.String("messaging.destination.name", trimmedTopic),
					attribute.String("messaging.message.id", message.UUID),
				),
			)

			err := handler(spanCtx, bus.Message{
				ID:       message.UUID,
				Topic:    trimmedTopic,
				Payload:  append([]byte(nil), message.Payload...),
				Metadata: metadata,
			})
			coretelemetry.RecordMessaging(trimmedTopic, "consume", startedAt, err)
			coretelemetry.EndSpan(span, err)

			return err
		},
	)

	if !strings.HasSuffix(trimmedTopic, r.dlqSuffix) {
		registeredHandler.AddMiddleware(
			watermillmiddleware.NewDLQ(trimmedTopic, r.dlqSuffix, r.dlqPublisher),
			r.retryMiddleware,
			wmmiddleware.Recoverer,
		)
	} else {
		registeredHandler.AddMiddleware(
			r.retryMiddleware,
			wmmiddleware.Recoverer,
		)
	}

	select {
	case <-r.router.Running():
		if err := r.router.RunHandlers(context.Background()); err != nil {
			return fmt.Errorf("run newly registered handler %q: %w", handlerName, err)
		}
	default:
	}

	return nil
}

// sanitizeTopic normalizes topic strings for deterministic handler naming.
func sanitizeTopic(topic string) string {
	cleaned := strings.ReplaceAll(topic, ".", "_")
	cleaned = strings.ReplaceAll(cleaned, "/", "_")
	cleaned = strings.ReplaceAll(cleaned, ":", "_")

	return cleaned
}
