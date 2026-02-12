package e2e_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	coremsgbus "mannaiah/module/core/messaging/bus"
	corewatermill "mannaiah/module/core/messaging/watermill"
)

// AwaitCreatedEvent waits for a created-contact integration event.
func (h *contactsE2EHarness) AwaitCreatedEvent(t *testing.T) contactEventRecord {
	t.Helper()

	return awaitEventRecord(t, h.createdEvents, "contacts.v1.created")
}

// AwaitUpdatedEvent waits for an updated-contact integration event.
func (h *contactsE2EHarness) AwaitUpdatedEvent(t *testing.T) contactEventRecord {
	t.Helper()

	return awaitEventRecord(t, h.updatedEvents, "contacts.v1.updated")
}

// AwaitAssetCreatedEvent waits for an asset-created integration event.
func (h *contactsE2EHarness) AwaitAssetCreatedEvent(t *testing.T) contactEventRecord {
	t.Helper()

	return awaitEventRecord(t, h.assetCreatedEvents, "assets.v1.created")
}

// AwaitAssetUpdatedEvent waits for an asset-updated integration event.
func (h *contactsE2EHarness) AwaitAssetUpdatedEvent(t *testing.T) contactEventRecord {
	t.Helper()

	return awaitEventRecord(t, h.assetUpdatedEvents, "assets.v1.updated")
}

// AwaitAssetDeletedEvent waits for an asset-deleted integration event.
func (h *contactsE2EHarness) AwaitAssetDeletedEvent(t *testing.T) contactEventRecord {
	t.Helper()

	return awaitEventRecord(t, h.assetDeletedEvents, "assets.v1.deleted")
}

// registerContactTopicHandler registers event listeners for a topic and pushes decoded events to a channel.
func registerContactTopicHandler(t *testing.T, messaging *corewatermill.InMemoryPlatform, topic string, sink chan<- contactEventRecord) {
	t.Helper()

	err := messaging.Registrar().AddHandler(topic, func(ctx context.Context, msg coremsgbus.Message) error {
		payload := map[string]any{}
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return err
		}

		sink <- contactEventRecord{
			Topic:    msg.Topic,
			Payload:  payload,
			Metadata: msg.Metadata,
		}

		return nil
	})
	if err != nil {
		t.Fatalf("Registrar().AddHandler(%q) error = %v", topic, err)
	}
}

// awaitEventRecord waits for an event record on the provided channel.
func awaitEventRecord(t *testing.T, source <-chan contactEventRecord, expectedTopic string) contactEventRecord {
	t.Helper()

	select {
	case event := <-source:
		if event.Topic != expectedTopic {
			t.Fatalf("event.Topic = %q, want %q", event.Topic, expectedTopic)
		}
		return event
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting for topic %q", expectedTopic)
		return contactEventRecord{}
	}
}
