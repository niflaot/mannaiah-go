package port

import "context"

// SNSMessage defines the minimal AWS SNS envelope values used for signature verification.
type SNSMessage struct {
	// Type defines SNS message type values.
	Type string
	// Message defines SNS embedded message values.
	Message string
	// MessageID defines SNS message identifier values.
	MessageID string
	// Subject defines optional SNS subject values.
	Subject string
	// Timestamp defines SNS RFC3339 timestamp values.
	Timestamp string
	// TopicARN defines SNS topic arn values.
	TopicARN string
	// Token defines optional SNS subscription token values.
	Token string
	// SubscribeURL defines SNS subscription confirmation URL values.
	SubscribeURL string
	// SignatureVersion defines SNS signature version values.
	SignatureVersion string
	// Signature defines SNS signature values.
	Signature string
	// SigningCertURL defines SNS signing-certificate URL values.
	SigningCertURL string
}

// SNSMessageVerifier defines signature verification behavior for SNS webhook envelopes.
type SNSMessageVerifier interface {
	// Verify validates signature values for one SNS message.
	Verify(ctx context.Context, message SNSMessage) error
}

// NoopSNSMessageVerifier defines no-op sns signature verification behavior.
type NoopSNSMessageVerifier struct{}

// Verify ignores signature verification inputs.
func (NoopSNSMessageVerifier) Verify(ctx context.Context, message SNSMessage) error {
	return nil
}
