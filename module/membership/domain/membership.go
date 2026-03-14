package domain

import "time"

// Channel defines consent channel values.
type Channel string

const (
	// ChannelAll defines global consent channels.
	ChannelAll Channel = "all"
	// ChannelEmail defines email consent channels.
	ChannelEmail Channel = "email"
)

// Action defines consent transition values.
type Action string

const (
	// ActionOptIn defines opt-in actions.
	ActionOptIn Action = "opt_in"
	// ActionOptOut defines opt-out actions.
	ActionOptOut Action = "opt_out"
)

// Status defines current membership status values.
type Status struct {
	// ContactID defines contact identifier values.
	ContactID string `json:"contactId"`
	// Channel defines channel values.
	Channel Channel `json:"channel"`
	// Action defines latest action values.
	Action Action `json:"action"`
	// Source defines latest source values.
	Source string `json:"source"`
	// OccurredAt defines latest action timestamp values.
	OccurredAt time.Time `json:"occurredAt"`
	// UpdatedAt defines status update timestamp values.
	UpdatedAt time.Time `json:"updatedAt"`
}

// Stamp defines immutable consent stamp values.
type Stamp struct {
	// ID defines stamp row identifier values.
	ID string `json:"id"`
	// ContactID defines contact identifier values.
	ContactID string `json:"contactId"`
	// Channel defines channel values.
	Channel Channel `json:"channel"`
	// Action defines action values.
	Action Action `json:"action"`
	// Source defines stamp source values.
	Source string `json:"source"`
	// OccurredAt defines action timestamp values.
	OccurredAt time.Time `json:"occurredAt"`
	// CreatedAt defines row creation timestamp values.
	CreatedAt time.Time `json:"createdAt"`
}
