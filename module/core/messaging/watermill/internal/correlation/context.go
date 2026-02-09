package correlation

import "context"

// contextCorrelationIDKey is a private context key for correlation propagation.
type contextCorrelationIDKey struct{}

// WithContext stores correlation id values inside a context.
func WithContext(ctx context.Context, correlationID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithValue(ctx, contextCorrelationIDKey{}, correlationID)
}

// FromContext reads correlation id values from a context.
func FromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}

	correlationID, ok := ctx.Value(contextCorrelationIDKey{}).(string)
	return correlationID, ok
}
