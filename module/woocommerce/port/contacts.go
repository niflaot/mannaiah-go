package port

import (
	"context"
	"time"
)

// UpsertOutcome defines contact upsert outcomes.
type UpsertOutcome string

const (
	// UpsertOutcomeCreated defines newly-created upsert outcomes.
	UpsertOutcomeCreated UpsertOutcome = "created"
	// UpsertOutcomeUpdated defines modified-row upsert outcomes.
	UpsertOutcomeUpdated UpsertOutcome = "updated"
	// UpsertOutcomeUnchanged defines no-op upsert outcomes.
	UpsertOutcomeUnchanged UpsertOutcome = "unchanged"
)

// ContactSyncCommand defines contact upsert payload values produced by WooCommerce syncs.
type ContactSyncCommand struct {
	// Email defines contact email values.
	Email string
	// FirstName defines contact first names.
	FirstName string
	// LastName defines contact last names.
	LastName string
	// LegalName defines contact legal name values.
	LegalName string
	// Phone defines contact phone values.
	Phone string
	// Address defines contact address values.
	Address string
	// AddressExtra defines additional address values.
	AddressExtra string
	// CityCode defines contact city values.
	CityCode string
	// DocumentType defines contact document type values.
	DocumentType string
	// DocumentNumber defines contact document number values.
	DocumentNumber string
	// CreatedAt defines source-based contact creation timestamps.
	CreatedAt *time.Time
	// Metadata defines contact metadata values produced by sync behavior.
	Metadata map[string]string
}

// ContactSyncTarget defines upsert behavior required by WooCommerce sync services.
type ContactSyncTarget interface {
	// UpsertByEmail creates or updates contacts keyed by email and reports upsert outcomes.
	UpsertByEmail(ctx context.Context, command ContactSyncCommand) (outcome UpsertOutcome, err error)
}
