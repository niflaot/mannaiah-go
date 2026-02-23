package publisher

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	watermill "github.com/ThreeDotsLabs/watermill"
	wmmsg "github.com/ThreeDotsLabs/watermill/message"
	wmmiddleware "github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"mannaiah/module/core/messaging/bus"
	correlationctx "mannaiah/module/core/messaging/watermill/internal/correlation"
	coretelemetry "mannaiah/module/core/telemetry"
)

var (
	// ErrNilPublisher is returned when a nil Watermill publisher is provided.
	ErrNilPublisher = errors.New("watermill publisher must not be nil")
	// ErrMessageIDRequired is returned when a message id is not provided.
	ErrMessageIDRequired = errors.New("message id is required")
	// ErrMessageTopicRequired is returned when a message topic is not provided.
	ErrMessageTopicRequired = errors.New("message topic is required")
)

// Adapter adapts Watermill publishers to bus.Publisher contracts.
type Adapter struct {
	// publisher is the wrapped Watermill publisher implementation.
	publisher wmmsg.Publisher
}

var (
	// _ ensures Adapter implements bus.Publisher.
	_ bus.Publisher = (*Adapter)(nil)
)

// NewAdapter creates a bus.Publisher adapter over a Watermill publisher.
func NewAdapter(publisher wmmsg.Publisher) (*Adapter, error) {
	if publisher == nil {
		return nil, ErrNilPublisher
	}

	return &Adapter{
		publisher: publisher,
	}, nil
}

// Publish publishes a bus envelope through the wrapped Watermill publisher.
func (a *Adapter) Publish(ctx context.Context, msg bus.Message) (err error) {
	if strings.TrimSpace(msg.ID) == "" {
		return ErrMessageIDRequired
	}
	if strings.TrimSpace(msg.Topic) == "" {
		return ErrMessageTopicRequired
	}

	outgoing := wmmsg.NewMessage(msg.ID, append([]byte(nil), msg.Payload...))
	for key, value := range msg.Metadata {
		outgoing.Metadata.Set(key, value)
	}

	applyCorrelationMetadata(ctx, outgoing)
	applyEventMetadata(msg.ID, outgoing)
	applyTraceMetadata(ctx, outgoing)

	startedAt := time.Now()
	spanCtx, span := coretelemetry.StartSpan(
		ctx,
		"mannaiah/messaging",
		"messaging.publish",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "watermill"),
			attribute.String("messaging.operation", "publish"),
			attribute.String("messaging.destination.name", msg.Topic),
		),
	)
	defer func() {
		coretelemetry.EndSpan(span, err)
		coretelemetry.RecordMessaging(msg.Topic, "publish", startedAt, err)
	}()

	traceparent := coretelemetry.TraceparentFromContext(spanCtx)
	if strings.TrimSpace(traceparent) != "" && outgoing.Metadata.Get(bus.MetadataTraceparent) == "" {
		outgoing.Metadata.Set(bus.MetadataTraceparent, traceparent)
	}

	if err = a.publisher.Publish(msg.Topic, outgoing); err != nil {
		return fmt.Errorf("publish topic %q: %w", msg.Topic, err)
	}

	span.SetAttributes(attribute.String("messaging.message.id", msg.ID))

	return nil
}

// applyCorrelationMetadata propagates correlation values from context when absent.
func applyCorrelationMetadata(ctx context.Context, message *wmmsg.Message) {
	if message.Metadata.Get(bus.MetadataCorrelationID) != "" {
		return
	}

	if ctx == nil {
		wmmiddleware.SetCorrelationID(watermill.NewUUID(), message)
		return
	}

	correlationID, ok := correlationctx.FromContext(ctx)
	if ok && strings.TrimSpace(correlationID) != "" {
		wmmiddleware.SetCorrelationID(correlationID, message)
		return
	}

	wmmiddleware.SetCorrelationID(watermill.NewUUID(), message)
}

// applyEventMetadata ensures required event metadata keys are present.
func applyEventMetadata(messageID string, message *wmmsg.Message) {
	if message.Metadata.Get(bus.MetadataEventID) == "" {
		message.Metadata.Set(bus.MetadataEventID, messageID)
	}
}

// applyTraceMetadata propagates traceparent metadata from context when absent.
func applyTraceMetadata(ctx context.Context, message *wmmsg.Message) {
	if message.Metadata.Get(bus.MetadataTraceparent) != "" {
		return
	}

	traceparent := coretelemetry.TraceparentFromContext(ctx)
	if strings.TrimSpace(traceparent) == "" {
		return
	}

	message.Metadata.Set(bus.MetadataTraceparent, traceparent)
}
