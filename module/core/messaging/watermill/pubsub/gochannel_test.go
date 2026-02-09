package pubsub

import (
	"context"
	"testing"
	"time"

	wmmsg "github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/zap"
	"mannaiah/module/core/messaging/platform"
)

// TestNewGoChannelPublishSubscribe verifies in-memory publish/subscribe behavior.
func TestNewGoChannelPublishSubscribe(t *testing.T) {
	instance := NewGoChannel(platform.Config{GoChannelBuffer: 1}, zap.NewNop())
	defer func() {
		if err := instance.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	subscriber, err := instance.Subscribe(context.Background(), "topic.v1")
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	publishDone := make(chan error, 1)
	go func() {
		publishDone <- instance.Publish("topic.v1", wmmsg.NewMessage("evt-1", []byte(`{"ok":true}`)))
	}()

	select {
	case message := <-subscriber:
		if message.UUID != "evt-1" {
			t.Fatalf("message UUID = %q, want %q", message.UUID, "evt-1")
		}
		message.Ack()
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for subscribed message")
	}

	select {
	case err := <-publishDone:
		if err != nil {
			t.Fatalf("Publish() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for publish completion")
	}
}

// TestNewGoChannelNilLogger verifies logger fallback behavior.
func TestNewGoChannelNilLogger(t *testing.T) {
	instance := NewGoChannel(platform.Config{}, nil)
	if instance == nil {
		t.Fatalf("expected non-nil gochannel instance")
	}

	if err := instance.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}
