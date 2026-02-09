package bus

// Message defines a technology-neutral integration message envelope.
type Message struct {
	// ID is the unique message identifier.
	ID string
	// Topic is the integration event topic name.
	Topic string
	// Payload is the serialized message payload.
	Payload []byte
	// Metadata is optional key-value metadata.
	Metadata map[string]string
}

const (
	// MetadataEventID is the metadata key for event identity.
	MetadataEventID = "event_id"
	// MetadataCorrelationID is the metadata key for end-to-end flow correlation.
	MetadataCorrelationID = "correlation_id"
	// MetadataSchemaVersion is the metadata key for event schema version.
	MetadataSchemaVersion = "schema_version"
	// MetadataCausationID is the optional metadata key for causation identity.
	MetadataCausationID = "causation_id"
	// MetadataProducedAt is the optional metadata key for message creation timestamp.
	MetadataProducedAt = "produced_at"
	// MetadataTraceparent is the optional metadata key for distributed tracing propagation.
	MetadataTraceparent = "traceparent"
)

const (
	// MetadataDLQOriginalTopic is the metadata key for dead-letter original topic.
	MetadataDLQOriginalTopic = "dlq_original_topic"
	// MetadataDLQError is the metadata key for dead-letter failure reason.
	MetadataDLQError = "dlq_error"
	// MetadataDLQFailedAt is the metadata key for dead-letter timestamp.
	MetadataDLQFailedAt = "dlq_failed_at"
)
