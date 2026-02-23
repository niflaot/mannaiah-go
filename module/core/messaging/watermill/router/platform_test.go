package router

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"mannaiah/module/core/messaging/bus"
	"mannaiah/module/core/messaging/platform"
	coretelemetry "mannaiah/module/core/telemetry"
)

// TestInMemoryPlatformAddHandlerValidation verifies handler registration input validation.
func TestInMemoryPlatformAddHandlerValidation(t *testing.T) {
	instance := newRunningPlatform(t, platform.Config{}, zap.NewNop())
	defer stopPlatform(t, instance)

	if err := instance.Registrar().AddHandler("", func(ctx context.Context, msg bus.Message) error {
		return nil
	}); !errors.Is(err, ErrEmptyTopic) {
		t.Fatalf("AddHandler() error = %v, want ErrEmptyTopic", err)
	}

	if err := instance.Registrar().AddHandler("topic", nil); !errors.Is(err, ErrNilHandler) {
		t.Fatalf("AddHandler() error = %v, want ErrNilHandler", err)
	}
}

// TestCorrelationPropagationProvided verifies provided correlation_id propagation across published messages.
func TestCorrelationPropagationProvided(t *testing.T) {
	instance := newRunningPlatform(t, platform.Config{}, zap.NewNop())
	defer stopPlatform(t, instance)

	firstReceived := make(chan string, 1)
	secondReceived := make(chan string, 1)

	if err := instance.Registrar().AddHandler("orders.v1.created", func(ctx context.Context, msg bus.Message) error {
		firstReceived <- msg.Metadata[bus.MetadataCorrelationID]

		return instance.Publisher().Publish(ctx, bus.Message{
			ID:      "evt-out-1",
			Topic:   "billing.v1.created",
			Payload: []byte(`{"id":"b1"}`),
			Metadata: map[string]string{
				bus.MetadataSchemaVersion: "v1",
			},
		})
	}); err != nil {
		t.Fatalf("AddHandler(orders.v1.created) error = %v", err)
	}

	if err := instance.Registrar().AddHandler("billing.v1.created", func(ctx context.Context, msg bus.Message) error {
		secondReceived <- msg.Metadata[bus.MetadataCorrelationID]
		return nil
	}); err != nil {
		t.Fatalf("AddHandler(billing.v1.created) error = %v", err)
	}

	if err := instance.Publisher().Publish(context.Background(), bus.Message{
		ID:      "evt-in-1",
		Topic:   "orders.v1.created",
		Payload: []byte(`{"id":"o1"}`),
		Metadata: map[string]string{
			bus.MetadataCorrelationID: "corr-fixed",
			bus.MetadataSchemaVersion: "v1",
		},
	}); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	firstCorrelation := readString(t, firstReceived, "first correlation")
	secondCorrelation := readString(t, secondReceived, "second correlation")

	if firstCorrelation != "corr-fixed" {
		t.Fatalf("first correlation = %q, want %q", firstCorrelation, "corr-fixed")
	}
	if secondCorrelation != "corr-fixed" {
		t.Fatalf("second correlation = %q, want %q", secondCorrelation, "corr-fixed")
	}
}

// TestCorrelationPropagationGenerated verifies generated correlation propagation when missing on ingress.
func TestCorrelationPropagationGenerated(t *testing.T) {
	instance := newRunningPlatform(t, platform.Config{}, zap.NewNop())
	defer stopPlatform(t, instance)

	firstReceived := make(chan string, 1)
	secondReceived := make(chan string, 1)

	if err := instance.Registrar().AddHandler("payments.v1.created", func(ctx context.Context, msg bus.Message) error {
		firstReceived <- msg.Metadata[bus.MetadataCorrelationID]

		return instance.Publisher().Publish(ctx, bus.Message{
			ID:      "evt-out-2",
			Topic:   "ledger.v1.created",
			Payload: []byte(`{"id":"l1"}`),
		})
	}); err != nil {
		t.Fatalf("AddHandler(payments.v1.created) error = %v", err)
	}

	if err := instance.Registrar().AddHandler("ledger.v1.created", func(ctx context.Context, msg bus.Message) error {
		secondReceived <- msg.Metadata[bus.MetadataCorrelationID]
		return nil
	}); err != nil {
		t.Fatalf("AddHandler(ledger.v1.created) error = %v", err)
	}

	if err := instance.Publisher().Publish(context.Background(), bus.Message{
		ID:      "evt-in-2",
		Topic:   "payments.v1.created",
		Payload: []byte(`{"id":"p1"}`),
	}); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	firstCorrelation := readString(t, firstReceived, "first generated correlation")
	secondCorrelation := readString(t, secondReceived, "second generated correlation")

	if firstCorrelation == "" {
		t.Fatalf("expected generated first correlation")
	}
	if secondCorrelation == "" {
		t.Fatalf("expected generated second correlation")
	}
	if firstCorrelation != secondCorrelation {
		t.Fatalf("generated correlation mismatch: first=%q second=%q", firstCorrelation, secondCorrelation)
	}
}

// TestTraceparentPropagation verifies trace context propagation through message metadata.
func TestTraceparentPropagation(t *testing.T) {
	provider, err := coretelemetry.Init(context.Background(), coretelemetry.Config{
		Enabled:       true,
		TracesEnabled: true,
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("coretelemetry.Init() error = %v", err)
	}
	defer func() {
		_ = provider.Shutdown(context.Background())
		coretelemetry.SetActive(nil)
	}()

	instance := newRunningPlatform(t, platform.Config{}, zap.NewNop())
	defer stopPlatform(t, instance)

	traceIDReceived := make(chan string, 1)

	if err := instance.Registrar().AddHandler("trace.v1", func(ctx context.Context, msg bus.Message) error {
		spanContext := trace.SpanContextFromContext(ctx)
		traceIDReceived <- spanContext.TraceID().String()
		return nil
	}); err != nil {
		t.Fatalf("AddHandler(trace.v1) error = %v", err)
	}

	parentCtx, parentSpan := coretelemetry.StartSpan(context.Background(), "test", "root")
	parentTraceparent := coretelemetry.TraceparentFromContext(parentCtx)
	parentSpanContext := trace.SpanContextFromContext(coretelemetry.ContextWithTraceparent(context.Background(), parentTraceparent))
	coretelemetry.EndSpan(parentSpan, nil)

	if err := instance.Publisher().Publish(context.Background(), bus.Message{
		ID:      "evt-trace-1",
		Topic:   "trace.v1",
		Payload: []byte(`{}`),
		Metadata: map[string]string{
			bus.MetadataTraceparent: parentTraceparent,
		},
	}); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	receivedTraceID := readString(t, traceIDReceived, "trace id")
	if receivedTraceID == "" {
		t.Fatalf("expected non-empty trace id")
	}
	if receivedTraceID != parentSpanContext.TraceID().String() {
		t.Fatalf("received trace id = %q, want %q", receivedTraceID, parentSpanContext.TraceID().String())
	}
}

// TestRetrySucceedsAfterTransientFailures verifies retry behavior for transient failures.
func TestRetrySucceedsAfterTransientFailures(t *testing.T) {
	instance := newRunningPlatform(t, platform.Config{
		RetryMaxRetries:        3,
		RetryInitialIntervalMS: 5,
		RetryMaxIntervalMS:     5,
		RetryMultiplier:        1,
	}, zap.NewNop())
	defer stopPlatform(t, instance)

	var attempts atomic.Int32
	done := make(chan struct{}, 1)

	if err := instance.Registrar().AddHandler("retry.v1", func(ctx context.Context, msg bus.Message) error {
		current := attempts.Add(1)
		if current < 3 {
			return errors.New("transient")
		}

		done <- struct{}{}
		return nil
	}); err != nil {
		t.Fatalf("AddHandler(retry.v1) error = %v", err)
	}

	if err := instance.Publisher().Publish(context.Background(), bus.Message{
		ID:      "evt-retry-1",
		Topic:   "retry.v1",
		Payload: []byte(`{}`),
	}); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	readSignal(t, done, "retry completion")
	if attempts.Load() != 3 {
		t.Fatalf("attempt count = %d, want %d", attempts.Load(), 3)
	}
}

// TestRecovererAndDLQ verifies panic recovery, dead-letter publishing, and continued router availability.
func TestRecovererAndDLQ(t *testing.T) {
	instance := newRunningPlatform(t, platform.Config{
		RetryMaxRetries:        2,
		RetryInitialIntervalMS: 5,
		RetryMaxIntervalMS:     5,
		RetryMultiplier:        1,
	}, zap.NewNop())
	defer stopPlatform(t, instance)

	var panicAttempts atomic.Int32
	dlqReceived := make(chan bus.Message, 1)
	healthyReceived := make(chan struct{}, 1)

	if err := instance.Registrar().AddHandler("panic.v1", func(ctx context.Context, msg bus.Message) error {
		panicAttempts.Add(1)
		panic("boom")
	}); err != nil {
		t.Fatalf("AddHandler(panic.v1) error = %v", err)
	}

	if err := instance.Registrar().AddHandler("panic.v1.dlq", func(ctx context.Context, msg bus.Message) error {
		dlqReceived <- msg
		return nil
	}); err != nil {
		t.Fatalf("AddHandler(panic.v1.dlq) error = %v", err)
	}

	if err := instance.Registrar().AddHandler("healthy.v1", func(ctx context.Context, msg bus.Message) error {
		healthyReceived <- struct{}{}
		return nil
	}); err != nil {
		t.Fatalf("AddHandler(healthy.v1) error = %v", err)
	}

	if err := instance.Publisher().Publish(context.Background(), bus.Message{
		ID:      "evt-panic-1",
		Topic:   "panic.v1",
		Payload: []byte(`{"id":"x"}`),
	}); err != nil {
		t.Fatalf("Publish(panic.v1) error = %v", err)
	}

	dlqMessage := readMessage(t, dlqReceived, "panic dlq")
	if dlqMessage.Topic != "panic.v1.dlq" {
		t.Fatalf("DLQ topic = %q, want %q", dlqMessage.Topic, "panic.v1.dlq")
	}
	if panicAttempts.Load() != 3 {
		t.Fatalf("panic attempts = %d, want %d", panicAttempts.Load(), 3)
	}

	if err := instance.Publisher().Publish(context.Background(), bus.Message{
		ID:      "evt-healthy-1",
		Topic:   "healthy.v1",
		Payload: []byte(`{}`),
	}); err != nil {
		t.Fatalf("Publish(healthy.v1) error = %v", err)
	}
	readSignal(t, healthyReceived, "healthy event")
}

// TestDLQMetadata verifies dead-letter payload and metadata enrichment after retries.
func TestDLQMetadata(t *testing.T) {
	instance := newRunningPlatform(t, platform.Config{
		RetryMaxRetries:        1,
		RetryInitialIntervalMS: 5,
		RetryMaxIntervalMS:     5,
		RetryMultiplier:        1,
	}, zap.NewNop())
	defer stopPlatform(t, instance)

	var attempts atomic.Int32
	dlqReceived := make(chan bus.Message, 1)

	if err := instance.Registrar().AddHandler("orders.v1.failed", func(ctx context.Context, msg bus.Message) error {
		attempts.Add(1)
		return errors.New("always failing")
	}); err != nil {
		t.Fatalf("AddHandler(orders.v1.failed) error = %v", err)
	}
	if err := instance.Registrar().AddHandler("orders.v1.failed.dlq", func(ctx context.Context, msg bus.Message) error {
		dlqReceived <- msg
		return nil
	}); err != nil {
		t.Fatalf("AddHandler(orders.v1.failed.dlq) error = %v", err)
	}

	if err := instance.Publisher().Publish(context.Background(), bus.Message{
		ID:      "evt-fail-1",
		Topic:   "orders.v1.failed",
		Payload: []byte(`{"order_id":"o1"}`),
		Metadata: map[string]string{
			bus.MetadataCorrelationID: "corr-dlq",
			bus.MetadataSchemaVersion: "v1",
		},
	}); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	dlqMessage := readMessage(t, dlqReceived, "orders dlq")
	if string(dlqMessage.Payload) != `{"order_id":"o1"}` {
		t.Fatalf("DLQ payload = %q, want %q", string(dlqMessage.Payload), `{"order_id":"o1"}`)
	}
	if dlqMessage.Metadata[bus.MetadataCorrelationID] != "corr-dlq" {
		t.Fatalf("DLQ correlation_id = %q, want %q", dlqMessage.Metadata[bus.MetadataCorrelationID], "corr-dlq")
	}
	if dlqMessage.Metadata[bus.MetadataDLQOriginalTopic] != "orders.v1.failed" {
		t.Fatalf("DLQ original topic = %q, want %q", dlqMessage.Metadata[bus.MetadataDLQOriginalTopic], "orders.v1.failed")
	}
	if dlqMessage.Metadata[bus.MetadataDLQError] == "" {
		t.Fatalf("expected non-empty DLQ error metadata")
	}
	if dlqMessage.Metadata[bus.MetadataDLQFailedAt] == "" {
		t.Fatalf("expected non-empty DLQ failed_at metadata")
	}
	if attempts.Load() != 2 {
		t.Fatalf("attempt count = %d, want %d", attempts.Load(), 2)
	}
}

// TestInMemoryPlatformCloseWithoutRun verifies close behavior when platform was not started.
func TestInMemoryPlatformCloseWithoutRun(t *testing.T) {
	instance, err := NewInMemoryPlatform(platform.Config{}, zap.NewNop())
	if err != nil {
		t.Fatalf("NewInMemoryPlatform() error = %v", err)
	}

	if err := instance.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

// TestAddHandlerNonRetriableBypassesRetries verifies non-retriable errors skip retry attempts.
func TestAddHandlerNonRetriableBypassesRetries(t *testing.T) {
	instance := newRunningPlatform(t, platform.Config{
		RetryMaxRetries:        5,
		RetryInitialIntervalMS: 5,
		RetryMaxIntervalMS:     5,
		RetryMultiplier:        1,
	}, zap.NewNop())
	defer stopPlatform(t, instance)

	var attempts atomic.Int32
	dlqReceived := make(chan bus.Message, 1)

	if err := instance.Registrar().AddHandler("payments.v1.nonretriable", func(ctx context.Context, msg bus.Message) error {
		attempts.Add(1)
		return platform.NonRetriable(fmt.Errorf("validation failed"))
	}); err != nil {
		t.Fatalf("AddHandler(payments.v1.nonretriable) error = %v", err)
	}
	if err := instance.Registrar().AddHandler("payments.v1.nonretriable.dlq", func(ctx context.Context, msg bus.Message) error {
		dlqReceived <- msg
		return nil
	}); err != nil {
		t.Fatalf("AddHandler(payments.v1.nonretriable.dlq) error = %v", err)
	}

	if err := instance.Publisher().Publish(context.Background(), bus.Message{
		ID:      "evt-nr-1",
		Topic:   "payments.v1.nonretriable",
		Payload: []byte(`{}`),
	}); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	_ = readMessage(t, dlqReceived, "non-retriable dlq")
	if attempts.Load() != 1 {
		t.Fatalf("attempt count = %d, want %d", attempts.Load(), 1)
	}
}

// newRunningPlatform creates and starts an in-memory platform for tests.
func newRunningPlatform(t *testing.T, cfg platform.Config, logger *zap.Logger) *InMemoryPlatform {
	t.Helper()

	instance, err := NewInMemoryPlatform(cfg, logger)
	if err != nil {
		t.Fatalf("NewInMemoryPlatform() error = %v", err)
	}

	go func() {
		if runErr := instance.Run(context.Background()); runErr != nil && !errors.Is(runErr, context.Canceled) {
			t.Errorf("Run() error = %v", runErr)
		}
	}()

	select {
	case <-instance.Running():
	case <-time.After(2 * time.Second):
		t.Fatalf("platform did not enter running state")
	}

	return instance
}

// stopPlatform stops an in-memory platform started for tests.
func stopPlatform(t *testing.T, instance *InMemoryPlatform) {
	t.Helper()

	if err := instance.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

// readString reads a string value from a channel with timeout.
func readString(t *testing.T, input <-chan string, label string) string {
	t.Helper()

	select {
	case value := <-input:
		return value
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for %s", label)
		return ""
	}
}

// readSignal waits for a completion signal with timeout.
func readSignal(t *testing.T, input <-chan struct{}, label string) {
	t.Helper()

	select {
	case <-input:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for %s", label)
	}
}

// readMessage reads a message value from a channel with timeout.
func readMessage(t *testing.T, input <-chan bus.Message, label string) bus.Message {
	t.Helper()

	select {
	case value := <-input:
		return value
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for %s", label)
		return bus.Message{}
	}
}
