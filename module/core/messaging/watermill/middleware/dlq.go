package middleware

import (
	"fmt"
	"strings"
	"time"

	wmmsg "github.com/ThreeDotsLabs/watermill/message"
	"mannaiah/module/core/messaging/bus"
	coretelemetry "mannaiah/module/core/telemetry"
)

const dlqErrorMaxLength = 512

// NewDLQ creates middleware that publishes failed messages into dead-letter topics.
func NewDLQ(topic string, suffix string, publisher wmmsg.Publisher) wmmsg.HandlerMiddleware {
	return func(next wmmsg.HandlerFunc) wmmsg.HandlerFunc {
		return func(message *wmmsg.Message) ([]*wmmsg.Message, error) {
			startedAt := time.Now()
			produced, err := next(message)
			if err == nil {
				return produced, nil
			}

			dlqMessage := copyMessageWithDLQMetadata(topic, message, err)
			dlqTopic := topic + suffix
			if publishErr := publisher.Publish(dlqTopic, dlqMessage); publishErr != nil {
				coretelemetry.RecordMessaging(dlqTopic, "publish", startedAt, publishErr)
				return nil, fmt.Errorf("publish dlq topic %q: %w", dlqTopic, publishErr)
			}
			coretelemetry.RecordMessaging(dlqTopic, "publish", startedAt, nil)
			coretelemetry.IncMessagingDLQ(topic)

			return nil, nil
		}
	}
}

// copyMessageWithDLQMetadata clones messages and enriches them with dead-letter metadata.
func copyMessageWithDLQMetadata(topic string, message *wmmsg.Message, err error) *wmmsg.Message {
	copied := wmmsg.NewMessage(message.UUID, append([]byte(nil), message.Payload...))
	for key, value := range message.Metadata {
		copied.Metadata.Set(key, value)
	}

	copied.Metadata.Set(bus.MetadataDLQOriginalTopic, topic)
	copied.Metadata.Set(bus.MetadataDLQError, truncateError(err))
	copied.Metadata.Set(bus.MetadataDLQFailedAt, time.Now().UTC().Format(time.RFC3339))

	return copied
}

// truncateError truncates serialized errors to prevent excessive metadata growth.
func truncateError(err error) string {
	if err == nil {
		return ""
	}

	value := err.Error()
	if len(value) <= dlqErrorMaxLength {
		return value
	}

	return strings.TrimSpace(value[:dlqErrorMaxLength])
}
