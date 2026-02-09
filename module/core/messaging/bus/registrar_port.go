package bus

import "context"

// Handler defines a technology-neutral integration message handler.
type Handler func(ctx context.Context, msg Message) error

// Registrar defines technology-neutral subscription registration behavior.
type Registrar interface {
	// AddHandler registers a topic handler.
	AddHandler(topic string, handler Handler) error
}
