package port

import (
	"context"
	"time"

	"mannaiah/module/membership/domain"
)

// StampInput defines stamp persistence payload values.
type StampInput struct {
	// ContactID defines contact identifier values.
	ContactID string
	// Channel defines channel values.
	Channel domain.Channel
	// Action defines action values.
	Action domain.Action
	// Source defines source values.
	Source string
	// OccurredAt defines action timestamp values.
	OccurredAt time.Time
}

// StampResult defines stamp persistence output values.
type StampResult struct {
	// Status defines current status values.
	Status domain.Status
	// Created defines whether a new stamp row was created.
	Created bool
}

// Repository defines membership persistence behavior.
type Repository interface {
	// SaveStamp persists immutable stamps and resolves latest effective status values.
	SaveStamp(ctx context.Context, input StampInput) (*StampResult, error)
	// GetStatus retrieves latest effective status by contact and channel.
	GetStatus(ctx context.Context, contactID string, channel domain.Channel) (*domain.Status, error)
	// GetStatuses retrieves effective statuses for every contact channel.
	GetStatuses(ctx context.Context, contactID string) ([]domain.Status, error)
	// ListStamps retrieves stamps by contact and channel filters.
	ListStamps(ctx context.Context, contactID string, channel domain.Channel, limit int) ([]domain.Stamp, error)
}
