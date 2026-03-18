package tag

import "time"

// Tag defines the canonical product taxonomy tag registry entry.
type Tag struct {
	// ID defines unique tag identifiers.
	ID uint `json:"id"`
	// Name defines the tag name value.
	Name string `json:"name"`
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time `json:"updatedAt"`
	// DeletedAt defines optional soft-delete timestamps.
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// TagCorrelation defines a cross-sell probability mapping between two product tags.
type TagCorrelation struct {
	// ID defines unique correlation identifiers.
	ID uint `json:"id"`
	// SourceTag defines the source tag name.
	SourceTag string `json:"sourceTag"`
	// TargetTag defines the correlated target tag name.
	TargetTag string `json:"targetTag"`
	// Probability defines cross-sell purchase probability (0.00–100.00).
	Probability float64 `json:"probability"`
	// Notes defines optional marketing notes.
	Notes string `json:"notes,omitempty"`
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time `json:"updatedAt"`
}
