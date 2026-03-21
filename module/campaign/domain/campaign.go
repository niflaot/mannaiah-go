package domain

import "time"

// Status defines campaign state values.
type Status string

const (
	// StatusPlanned defines planned campaign statuses.
	StatusPlanned Status = "PLANNED"
	// StatusProcessing defines in-flight campaign statuses.
	StatusProcessing Status = "PROCESSING"
	// StatusSent defines completed campaign statuses.
	StatusSent Status = "SENT"
	// StatusFailed defines failed campaign statuses.
	StatusFailed Status = "FAILED"
)

// Campaign defines campaign entity values.
type Campaign struct {
	// ID defines campaign identifier values.
	ID string `json:"id"`
	// Name defines campaign names.
	Name string `json:"name"`
	// Slug defines campaign slugs.
	Slug string `json:"slug"`
	// Channel defines target channel values.
	Channel string `json:"channel"`
	// SegmentID defines target segment identifier values.
	SegmentID string `json:"segmentId"`
	// Subject defines email subject values.
	Subject string `json:"subject"`
	// HTMLBody defines html content values.
	HTMLBody string `json:"htmlBody"`
	// TextBody defines text content values.
	TextBody string `json:"textBody"`
	// Status defines campaign status values.
	Status Status `json:"status"`
	// TotalRecipients defines total resolved recipients values.
	TotalRecipients int `json:"totalRecipients"`
	// SentCount defines delivered send count values.
	SentCount int `json:"sentCount"`
	// FailedCount defines failed send count values.
	FailedCount int `json:"failedCount"`
	// TemplateVars defines campaign-level custom variable values available in the template context.
	TemplateVars map[string]string `json:"templateVars,omitempty"`
	// ProductBlocks defines product recommendation blocks rendered into the template context.
	ProductBlocks []ProductBlock `json:"productBlocks,omitempty"`
	// CreatedAt defines row creation timestamp values.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt defines row update timestamp values.
	UpdatedAt time.Time `json:"updatedAt"`
}
