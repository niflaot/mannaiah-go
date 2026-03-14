package store

import "time"

// stampModel defines membership stamp persistence row values.
type stampModel struct {
	// ID defines stamp identifier values.
	ID string `gorm:"column:id;primaryKey"`
	// ContactID defines contact identifier values.
	ContactID string `gorm:"column:contact_id"`
	// Channel defines channel values.
	Channel string `gorm:"column:channel"`
	// Action defines action values.
	Action string `gorm:"column:action"`
	// Source defines source values.
	Source string `gorm:"column:source"`
	// OccurredAt defines action timestamp values.
	OccurredAt time.Time `gorm:"column:occurred_at"`
	// CreatedAt defines row creation timestamp values.
	CreatedAt time.Time `gorm:"column:created_at"`
}

// TableName resolves stamp table names.
func (stampModel) TableName() string {
	return "membership_stamps"
}
