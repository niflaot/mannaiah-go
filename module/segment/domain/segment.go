package domain

import "time"

// Filter defines supported segment filter values.
type Filter struct {
	// Type defines filter type values.
	Type string `json:"type"`
	// Value defines scalar filter value payloads.
	Value any `json:"value,omitempty"`
}

// Segment defines audience segment definition values.
type Segment struct {
	// ID defines segment identifier values.
	ID string `json:"id"`
	// Name defines human-readable segment names.
	Name string `json:"name"`
	// Slug defines URL-safe segment slugs.
	Slug string `json:"slug"`
	// Channel defines target channel values.
	Channel string `json:"channel"`
	// Filters defines filter DSL values.
	Filters []Filter `json:"filters"`
	// CreatedAt defines row creation timestamp values.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt defines row update timestamp values.
	UpdatedAt time.Time `json:"updatedAt"`
}
