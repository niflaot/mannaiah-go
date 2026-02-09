package middleware

import (
	"errors"
	"testing"

	watermill "github.com/ThreeDotsLabs/watermill"
	wmmsg "github.com/ThreeDotsLabs/watermill/message"
	wmmiddleware "github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"mannaiah/module/core/messaging/bus"
	"mannaiah/module/core/messaging/platform"
)

// failingPublisherProbe is a publisher probe that always fails.
type failingPublisherProbe struct{}

// Publish always returns a configured failure.
func (f failingPublisherProbe) Publish(topic string, messages ...*wmmsg.Message) error {
	return errors.New("publish failed")
}

// Close is a no-op for failing publisher probes.
func (f failingPublisherProbe) Close() error {
	return nil
}

// TestCorrelationAddsCorrelation verifies missing correlation id generation.
func TestCorrelationAddsCorrelation(t *testing.T) {
	handler := Correlation(func(message *wmmsg.Message) ([]*wmmsg.Message, error) {
		return nil, nil
	})

	msg := wmmsg.NewMessage("id-1", []byte(`{}`))
	if _, err := handler(msg); err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if msg.Metadata.Get(bus.MetadataCorrelationID) == "" {
		t.Fatalf("expected correlation_id to be generated")
	}
}

// TestCorrelationPropagatesToProducedMessages verifies produced message correlation propagation.
func TestCorrelationPropagatesToProducedMessages(t *testing.T) {
	handler := Correlation(func(message *wmmsg.Message) ([]*wmmsg.Message, error) {
		return []*wmmsg.Message{
			wmmsg.NewMessage("id-2", []byte(`{}`)),
		}, nil
	})

	msg := wmmsg.NewMessage("id-1", []byte(`{}`))
	msg.Metadata.Set(bus.MetadataCorrelationID, "corr-1")
	produced, err := handler(msg)
	if err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if produced[0].Metadata.Get(bus.MetadataCorrelationID) != "corr-1" {
		t.Fatalf("produced correlation = %q, want %q", produced[0].Metadata.Get(bus.MetadataCorrelationID), "corr-1")
	}
}

// TestShouldRetryClassification verifies non-retriable classification behavior.
func TestShouldRetryClassification(t *testing.T) {
	if ShouldRetry(wmmiddleware.RetryParams{Err: platform.NonRetriable(errors.New("validation"))}) {
		t.Fatalf("expected non-retriable errors not to be retried")
	}
	if !ShouldRetry(wmmiddleware.RetryParams{Err: errors.New("transient")}) {
		t.Fatalf("expected transient errors to be retried")
	}
}

// TestTruncateError verifies dead-letter error truncation behavior.
func TestTruncateError(t *testing.T) {
	if got := truncateError(nil); got != "" {
		t.Fatalf("truncateError(nil) = %q, want empty", got)
	}

	short := truncateError(errors.New("short"))
	if short != "short" {
		t.Fatalf("truncateError(short) = %q, want %q", short, "short")
	}

	longErr := errors.New(makeLongString(dlqErrorMaxLength + 20))
	long := truncateError(longErr)
	if len(long) != dlqErrorMaxLength {
		t.Fatalf("truncateError(long) length = %d, want %d", len(long), dlqErrorMaxLength)
	}
}

// TestDLQPublishFailure verifies publish errors are returned when dead-letter publishing fails.
func TestDLQPublishFailure(t *testing.T) {
	handler := NewDLQ("orders.v1.created", ".dlq", failingPublisherProbe{})(func(message *wmmsg.Message) ([]*wmmsg.Message, error) {
		return nil, errors.New("handler failed")
	})

	_, err := handler(wmmsg.NewMessage("id-1", []byte(`{}`)))
	if err == nil {
		t.Fatalf("expected dlq publish failure error")
	}
}

// TestAddRouterMiddlewaresAndNewRetry verifies middleware constructors execute correctly.
func TestAddRouterMiddlewaresAndNewRetry(t *testing.T) {
	router, err := wmmsg.NewRouter(wmmsg.RouterConfig{}, watermill.NopLogger{})
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	AddRouterMiddlewares(router)
	retryMiddleware := NewRetry(platform.Config{
		RetryMaxRetries:        1,
		RetryInitialIntervalMS: 1,
		RetryMaxIntervalMS:     1,
		RetryMultiplier:        1,
	}, watermill.NopLogger{})

	attempts := 0
	handler := retryMiddleware(func(message *wmmsg.Message) ([]*wmmsg.Message, error) {
		attempts++
		if attempts == 1 {
			return nil, errors.New("transient")
		}

		return nil, nil
	})

	if _, err := handler(wmmsg.NewMessage("id-1", []byte(`{}`))); err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want %d", attempts, 2)
	}
}

// makeLongString creates a repeated value for truncation tests.
func makeLongString(length int) string {
	buffer := make([]byte, length)
	for index := range buffer {
		buffer[index] = 'a'
	}

	return string(buffer)
}
