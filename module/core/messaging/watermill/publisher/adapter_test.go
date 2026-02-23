package publisher

import (
	"context"
	"errors"
	"testing"

	wmmsg "github.com/ThreeDotsLabs/watermill/message"
	"mannaiah/module/core/messaging/bus"
	correlationctx "mannaiah/module/core/messaging/watermill/internal/correlation"
	coretelemetry "mannaiah/module/core/telemetry"
)

// publisherProbe is a message publisher probe used for adapter unit tests.
type publisherProbe struct {
	// topic stores the last published topic.
	topic string
	// messages stores all published messages.
	messages []*wmmsg.Message
	// err defines the returned publish error.
	err error
}

// Publish stores publish calls and returns the configured probe error.
func (p *publisherProbe) Publish(topic string, messages ...*wmmsg.Message) error {
	p.topic = topic
	p.messages = append(p.messages, messages...)
	return p.err
}

// Close is a no-op for probe publishers.
func (p *publisherProbe) Close() error {
	return nil
}

// TestNewAdapterRejectsNil verifies constructor validation for nil publishers.
func TestNewAdapterRejectsNil(t *testing.T) {
	_, err := NewAdapter(nil)
	if !errors.Is(err, ErrNilPublisher) {
		t.Fatalf("NewAdapter() error = %v, want ErrNilPublisher", err)
	}
}

// TestPublishMapsEnvelope verifies envelope-to-watermill mapping behavior.
func TestPublishMapsEnvelope(t *testing.T) {
	probe := &publisherProbe{}
	adapter, err := NewAdapter(probe)
	if err != nil {
		t.Fatalf("NewAdapter() error = %v", err)
	}

	input := bus.Message{
		ID:      "evt-1",
		Topic:   "orders.v1.created",
		Payload: []byte(`{"id":"o1"}`),
		Metadata: map[string]string{
			bus.MetadataCorrelationID: "corr-1",
			bus.MetadataSchemaVersion: "v1",
		},
	}
	if err := adapter.Publish(context.Background(), input); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	if probe.topic != "orders.v1.created" {
		t.Fatalf("topic = %q, want %q", probe.topic, "orders.v1.created")
	}
	if len(probe.messages) != 1 {
		t.Fatalf("published message count = %d, want %d", len(probe.messages), 1)
	}

	output := probe.messages[0]
	if output.UUID != "evt-1" {
		t.Fatalf("UUID = %q, want %q", output.UUID, "evt-1")
	}
	if string(output.Payload) != `{"id":"o1"}` {
		t.Fatalf("Payload = %q, want %q", string(output.Payload), `{"id":"o1"}`)
	}
	if output.Metadata.Get(bus.MetadataCorrelationID) != "corr-1" {
		t.Fatalf("correlation_id = %q, want %q", output.Metadata.Get(bus.MetadataCorrelationID), "corr-1")
	}
	if output.Metadata.Get(bus.MetadataEventID) != "evt-1" {
		t.Fatalf("event_id = %q, want %q", output.Metadata.Get(bus.MetadataEventID), "evt-1")
	}
}

// TestPublishPropagatesCorrelationFromContext verifies correlation propagation from handler context.
func TestPublishPropagatesCorrelationFromContext(t *testing.T) {
	probe := &publisherProbe{}
	adapter, err := NewAdapter(probe)
	if err != nil {
		t.Fatalf("NewAdapter() error = %v", err)
	}

	ctx := correlationctx.WithContext(context.Background(), "ctx-correlation")
	if err := adapter.Publish(ctx, bus.Message{
		ID:      "evt-2",
		Topic:   "billing.v1.created",
		Payload: []byte(`{}`),
	}); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	correlationID := probe.messages[0].Metadata.Get(bus.MetadataCorrelationID)
	if correlationID != "ctx-correlation" {
		t.Fatalf("correlation_id = %q, want %q", correlationID, "ctx-correlation")
	}
}

// TestPublishGeneratesCorrelation verifies generated correlation id when missing.
func TestPublishGeneratesCorrelation(t *testing.T) {
	probe := &publisherProbe{}
	adapter, err := NewAdapter(probe)
	if err != nil {
		t.Fatalf("NewAdapter() error = %v", err)
	}

	if err := adapter.Publish(context.Background(), bus.Message{
		ID:      "evt-3",
		Topic:   "billing.v1.created",
		Payload: []byte(`{}`),
	}); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	correlationID := probe.messages[0].Metadata.Get(bus.MetadataCorrelationID)
	if correlationID == "" {
		t.Fatalf("expected generated correlation_id")
	}
}

// TestPublishInjectsTraceparent verifies publish calls propagate traceparent metadata.
func TestPublishInjectsTraceparent(t *testing.T) {
	provider, err := coretelemetry.Init(context.Background(), coretelemetry.Config{
		Enabled:       true,
		TracesEnabled: true,
	}, nil)
	if err != nil {
		t.Fatalf("coretelemetry.Init() error = %v", err)
	}
	defer func() {
		_ = provider.Shutdown(context.Background())
		coretelemetry.SetActive(nil)
	}()

	probe := &publisherProbe{}
	adapter, err := NewAdapter(probe)
	if err != nil {
		t.Fatalf("NewAdapter() error = %v", err)
	}

	spanCtx, span := coretelemetry.StartSpan(context.Background(), "test", "root")
	defer coretelemetry.EndSpan(span, nil)

	if err := adapter.Publish(spanCtx, bus.Message{
		ID:      "evt-trace-1",
		Topic:   "orders.v1.created",
		Payload: []byte(`{}`),
	}); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	traceparent := probe.messages[0].Metadata.Get(bus.MetadataTraceparent)
	if traceparent == "" {
		t.Fatalf("expected traceparent metadata to be injected")
	}
}

// TestPublishValidation verifies required message fields.
func TestPublishValidation(t *testing.T) {
	probe := &publisherProbe{}
	adapter, err := NewAdapter(probe)
	if err != nil {
		t.Fatalf("NewAdapter() error = %v", err)
	}

	if err := adapter.Publish(context.Background(), bus.Message{
		Topic:   "orders.v1.created",
		Payload: []byte(`{}`),
	}); !errors.Is(err, ErrMessageIDRequired) {
		t.Fatalf("Publish() error = %v, want ErrMessageIDRequired", err)
	}

	if err := adapter.Publish(context.Background(), bus.Message{
		ID:      "evt-4",
		Payload: []byte(`{}`),
	}); !errors.Is(err, ErrMessageTopicRequired) {
		t.Fatalf("Publish() error = %v, want ErrMessageTopicRequired", err)
	}
}

// TestPublishPropagatesPublisherError verifies transport publish errors are wrapped.
func TestPublishPropagatesPublisherError(t *testing.T) {
	probe := &publisherProbe{err: errors.New("transport failed")}
	adapter, err := NewAdapter(probe)
	if err != nil {
		t.Fatalf("NewAdapter() error = %v", err)
	}

	err = adapter.Publish(context.Background(), bus.Message{
		ID:      "evt-5",
		Topic:   "orders.v1.created",
		Payload: []byte(`{}`),
	})
	if err == nil {
		t.Fatalf("expected wrapped publish error")
	}
}
