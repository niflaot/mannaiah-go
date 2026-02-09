package bus

import "context"

// Publisher defines technology-neutral integration event publication behavior.
type Publisher interface {
	// Publish sends a message through the integration event bus.
	Publish(ctx context.Context, msg Message) error
}
