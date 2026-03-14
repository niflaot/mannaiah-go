package port

import "context"

// Store defines analytics storage behavior.
type Store interface {
	// Ping verifies analytics backend connectivity.
	Ping(ctx context.Context) error
}
