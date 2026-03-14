package port

import (
	"context"
	"time"

	"mannaiah/module/membership/domain"
)

// StampCommand defines membership stamp command values.
type StampCommand struct {
	// ContactID defines optional contact identifier values.
	ContactID string
	// Email defines optional lookup email values.
	Email string
	// Channel defines channel values.
	Channel domain.Channel
	// Action defines action values.
	Action domain.Action
	// Source defines source values.
	Source string
	// OccurredAt defines optional action timestamp values.
	OccurredAt *time.Time
}

// Stamper defines stamp behavior used by external modules.
type Stamper interface {
	// Stamp persists membership stamps and updates latest status snapshots.
	Stamp(ctx context.Context, command StampCommand) (*domain.Status, error)
}
