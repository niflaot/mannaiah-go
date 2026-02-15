package store

import (
	"time"

	"gorm.io/gorm"
)

// orderRecord defines order root persistence schema.
type orderRecord struct {
	// ID defines order identifiers.
	ID string `gorm:"primaryKey;size:64"`
	// Identifier defines external order identifiers.
	Identifier string `gorm:"size:255;not null;index:idx_orders_realm_identifier,priority:2,unique"`
	// Realm defines order realm values.
	Realm string `gorm:"size:128;not null;index:idx_orders_realm_identifier,priority:1,unique"`
	// ContactID defines linked customer identifiers.
	ContactID string `gorm:"size:64;not null;index"`
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time
	// DeletedAt defines soft-delete timestamps.
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// orderItemRecord defines order-item persistence rows.
type orderItemRecord struct {
	// ID defines surrogate identifiers.
	ID uint `gorm:"primaryKey"`
	// OrderID defines owning order identifiers.
	OrderID string `gorm:"size:64;not null;index;uniqueIndex:idx_order_items_order_position,priority:1"`
	// Position defines stable item ordering.
	Position int `gorm:"not null;index;uniqueIndex:idx_order_items_order_position,priority:2"`
	// SKU defines item SKU values.
	SKU string `gorm:"size:255;not null;index"`
	// AlternateName defines item alternate-name values.
	AlternateName string `gorm:"size:255;index"`
	// Quantity defines item quantity values.
	Quantity int `gorm:"not null"`
	// Value defines item monetary value values.
	Value float64 `gorm:"not null;default:0"`
	// ProductID defines resolved product identifiers.
	ProductID *string `gorm:"size:64;index"`
	// ResolutionSource defines item resolution-source values.
	ResolutionSource string `gorm:"size:64;not null"`
}

// orderShippingChargeRecord defines shipping-charge persistence rows.
type orderShippingChargeRecord struct {
	// ID defines surrogate identifiers.
	ID uint `gorm:"primaryKey"`
	// OrderID defines owning order identifiers.
	OrderID string `gorm:"size:64;not null;index;uniqueIndex:idx_order_shipping_charges_order_position,priority:1"`
	// Position defines stable shipping-charge ordering.
	Position int `gorm:"not null;index;uniqueIndex:idx_order_shipping_charges_order_position,priority:2"`
	// MethodID defines shipping method identifier values.
	MethodID string `gorm:"size:128;index"`
	// MethodTitle defines shipping method title values.
	MethodTitle string `gorm:"size:255"`
	// Price defines shipping price values.
	Price float64 `gorm:"not null;default:0"`
}

// orderStatusRecord defines order status-history persistence rows.
type orderStatusRecord struct {
	// ID defines surrogate identifiers.
	ID uint `gorm:"primaryKey"`
	// OrderID defines owning order identifiers.
	OrderID string `gorm:"size:64;not null;index;uniqueIndex:idx_order_status_order_position,priority:1;index:idx_order_status_order_occurred,priority:1"`
	// Position defines stable history ordering.
	Position int `gorm:"not null;index;uniqueIndex:idx_order_status_order_position,priority:2"`
	// Status defines status values.
	Status string `gorm:"size:32;not null;index"`
	// Author defines status author values.
	Author string `gorm:"size:255;not null"`
	// Description defines status description values.
	Description string `gorm:"type:text"`
	// NoteOwner defines optional note owner values.
	NoteOwner string `gorm:"size:255"`
	// Note defines optional note values.
	Note string `gorm:"type:text"`
	// OccurredAt defines status timestamps.
	OccurredAt time.Time `gorm:"not null;index;index:idx_order_status_order_occurred,priority:2"`
}

// orderCommentRecord defines order comment-history persistence rows.
type orderCommentRecord struct {
	// ID defines surrogate identifiers.
	ID uint `gorm:"primaryKey"`
	// OrderID defines owning order identifiers.
	OrderID string `gorm:"size:64;not null;index;index:idx_order_comments_order_occurred,priority:1"`
	// Author defines comment author values.
	Author string `gorm:"size:255;not null"`
	// Comment defines comment text values.
	Comment string `gorm:"type:text;not null"`
	// Internal reports whether comments are internal-only.
	Internal bool `gorm:"not null;default:false"`
	// OccurredAt defines comment timestamps.
	OccurredAt time.Time `gorm:"not null;index;index:idx_order_comments_order_occurred,priority:2"`
}

// orderShippingAddressRecord defines optional shipping-address rows.
type orderShippingAddressRecord struct {
	// ID defines surrogate identifiers.
	ID uint `gorm:"primaryKey"`
	// OrderID defines owning order identifiers.
	OrderID string `gorm:"size:64;not null;uniqueIndex"`
	// Address defines shipping address line 1 values.
	Address string `gorm:"size:512;not null"`
	// Address2 defines shipping address line 2 values.
	Address2 string `gorm:"size:512"`
	// Phone defines shipping phone values.
	Phone string `gorm:"size:64"`
	// CityCode defines shipping city-code values.
	CityCode string `gorm:"size:64;not null"`
}

// orderMetadataRecord defines order metadata persistence rows.
type orderMetadataRecord struct {
	// ID defines surrogate identifiers.
	ID uint `gorm:"primaryKey"`
	// OrderID defines owning order identifiers.
	OrderID string `gorm:"size:64;not null;index;uniqueIndex:idx_order_metadata_order_key,priority:1"`
	// Key defines metadata keys.
	Key string `gorm:"size:128;not null;index;uniqueIndex:idx_order_metadata_order_key,priority:2"`
	// Value defines metadata values.
	Value string `gorm:"type:text;not null"`
}

// orderItemMetadataRecord defines order-item metadata persistence rows.
type orderItemMetadataRecord struct {
	// ID defines surrogate identifiers.
	ID uint `gorm:"primaryKey"`
	// OrderItemID defines owning order-item identifiers.
	OrderItemID uint `gorm:"not null;index;uniqueIndex:idx_order_item_metadata_item_key,priority:1"`
	// Key defines metadata keys.
	Key string `gorm:"size:128;not null;index;uniqueIndex:idx_order_item_metadata_item_key,priority:2"`
	// Value defines metadata values.
	Value string `gorm:"type:text;not null"`
}

// TableName defines storage table names.
func (orderRecord) TableName() string { return "orders" }

// TableName defines storage table names.
func (orderItemRecord) TableName() string { return "order_items" }

// TableName defines storage table names.
func (orderStatusRecord) TableName() string { return "order_status_history" }

// TableName defines storage table names.
func (orderCommentRecord) TableName() string { return "order_comments" }

// TableName defines storage table names.
func (orderShippingAddressRecord) TableName() string { return "order_shipping_addresses" }

// TableName defines storage table names.
func (orderShippingChargeRecord) TableName() string { return "order_shipping_charges" }

// TableName defines storage table names.
func (orderMetadataRecord) TableName() string { return "order_metadata" }

// TableName defines storage table names.
func (orderItemMetadataRecord) TableName() string { return "order_item_metadata" }
