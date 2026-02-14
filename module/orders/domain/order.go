package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	// ErrIdentifierRequired is returned when order identifiers are empty.
	ErrIdentifierRequired = errors.New("order identifier is required")
	// ErrRealmRequired is returned when order realms are empty.
	ErrRealmRequired = errors.New("order realm is required")
	// ErrContactIDRequired is returned when order contact identifiers are empty.
	ErrContactIDRequired = errors.New("order contact id is required")
	// ErrItemsRequired is returned when order items are empty.
	ErrItemsRequired = errors.New("order items are required")
	// ErrItemSKURequired is returned when order item SKUs are empty.
	ErrItemSKURequired = errors.New("order item sku is required")
	// ErrItemQuantityInvalid is returned when order item quantities are invalid.
	ErrItemQuantityInvalid = errors.New("order item quantity must be greater than zero")
	// ErrStatusInvalid is returned when order statuses are not supported.
	ErrStatusInvalid = errors.New("order status is invalid")
	// ErrStatusAuthorRequired is returned when status authors are empty.
	ErrStatusAuthorRequired = errors.New("order status author is required")
	// ErrInvalidMetadata is returned when metadata keys or values are invalid.
	ErrInvalidMetadata = errors.New("order metadata is invalid")
)

// Status defines supported order-status values.
type Status string

const (
	// StatusCancelled defines cancelled-order status values.
	StatusCancelled Status = "CANCELLED"
	// StatusCreated defines created-order status values.
	StatusCreated Status = "CREATED"
	// StatusPending defines pending-order status values.
	StatusPending Status = "PENDING"
	// StatusHold defines on-hold-order status values.
	StatusHold Status = "HOLD"
	// StatusCompleted defines completed-order status values.
	StatusCompleted Status = "COMPLETED"
)

// ItemResolutionSource defines product-resolution origin values.
type ItemResolutionSource string

const (
	// ItemResolutionSourceSKU defines SKU-based resolution values.
	ItemResolutionSourceSKU ItemResolutionSource = "sku"
	// ItemResolutionSourceAlternateName defines alternate-name-based resolution values.
	ItemResolutionSourceAlternateName ItemResolutionSource = "alternate_name"
	// ItemResolutionSourceUnresolved defines unresolved item values.
	ItemResolutionSourceUnresolved ItemResolutionSource = "unresolved"
)

// Item defines normalized order-item values.
type Item struct {
	// SKU defines product SKU values from order payloads.
	SKU string `json:"sku"`
	// AlternateName defines alternate product-name values used for fallback lookup.
	AlternateName string `json:"alternateName"`
	// Quantity defines ordered-quantity values.
	Quantity int `json:"quantity"`
	// ProductID defines resolved product identifiers.
	ProductID string `json:"productId,omitempty"`
	// ResolutionSource defines item resolution origin values.
	ResolutionSource ItemResolutionSource `json:"resolutionSource"`
	// Metadata defines item metadata values.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// StatusEntry defines order-status history values.
type StatusEntry struct {
	// Status defines status values.
	Status Status `json:"status"`
	// Author defines author values (user or system).
	Author string `json:"author"`
	// Description defines optional status-description values.
	Description string `json:"description,omitempty"`
	// OccurredAt defines status transition timestamps.
	OccurredAt time.Time `json:"occurredAt"`
}

// ShippingAddress defines order shipping-address values.
type ShippingAddress struct {
	// Address defines shipping address line 1 values.
	Address string `json:"address"`
	// Address2 defines shipping address line 2 values.
	Address2 string `json:"address2,omitempty"`
	// Phone defines shipping phone values.
	Phone string `json:"phone,omitempty"`
	// CityCode defines shipping city-code values.
	CityCode string `json:"cityCode"`
}

// Order defines normalized order aggregate values.
type Order struct {
	// ID defines unique order identifiers.
	ID string `json:"id"`
	// Identifier defines external order identifiers.
	Identifier string `json:"identifier"`
	// Realm defines order realm values.
	Realm string `json:"realm"`
	// ContactID defines customer contact identifiers.
	ContactID string `json:"contactId"`
	// Items defines order item values.
	Items []Item `json:"items"`
	// CurrentStatus defines current order status values.
	CurrentStatus Status `json:"currentStatus"`
	// StatusHistory defines order status-history values.
	StatusHistory []StatusEntry `json:"statusHistory"`
	// ShippingAddress defines resolved shipping-address values.
	ShippingAddress ShippingAddress `json:"shippingAddress"`
	// HasCustomShippingAddress reports whether shipping was explicitly provided for this order.
	HasCustomShippingAddress bool `json:"hasCustomShippingAddress"`
	// Metadata defines order metadata values.
	Metadata map[string]string `json:"metadata,omitempty"`
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time `json:"updatedAt"`
}

// Normalize canonicalizes order values before validation and persistence.
func (o *Order) Normalize() {
	if o == nil {
		return
	}

	o.ID = strings.TrimSpace(o.ID)
	o.Identifier = strings.TrimSpace(o.Identifier)
	o.Realm = strings.TrimSpace(o.Realm)
	o.ContactID = strings.TrimSpace(o.ContactID)
	o.CurrentStatus = Status(strings.TrimSpace(string(o.CurrentStatus)))
	o.ShippingAddress = normalizeShippingAddress(o.ShippingAddress)

	for index := range o.Items {
		o.Items[index].SKU = strings.TrimSpace(o.Items[index].SKU)
		o.Items[index].AlternateName = strings.TrimSpace(o.Items[index].AlternateName)
		o.Items[index].ProductID = strings.TrimSpace(o.Items[index].ProductID)
		o.Items[index].ResolutionSource = ItemResolutionSource(strings.TrimSpace(string(o.Items[index].ResolutionSource)))
		o.Items[index].Metadata = normalizeMetadata(o.Items[index].Metadata)
	}
	for index := range o.StatusHistory {
		o.StatusHistory[index].Status = Status(strings.TrimSpace(string(o.StatusHistory[index].Status)))
		o.StatusHistory[index].Author = strings.TrimSpace(o.StatusHistory[index].Author)
		o.StatusHistory[index].Description = strings.TrimSpace(o.StatusHistory[index].Description)
	}
	o.Metadata = normalizeMetadata(o.Metadata)
}

// Validate validates order aggregate invariants.
func (o Order) Validate() error {
	if strings.TrimSpace(o.Identifier) == "" {
		return ErrIdentifierRequired
	}
	if strings.TrimSpace(o.Realm) == "" {
		return ErrRealmRequired
	}
	if strings.TrimSpace(o.ContactID) == "" {
		return ErrContactIDRequired
	}
	if len(o.Items) == 0 {
		return ErrItemsRequired
	}
	for _, item := range o.Items {
		if strings.TrimSpace(item.SKU) == "" {
			return ErrItemSKURequired
		}
		if item.Quantity <= 0 {
			return ErrItemQuantityInvalid
		}
		if !isValidMetadata(item.Metadata) {
			return ErrInvalidMetadata
		}
	}
	if err := validateStatus(o.CurrentStatus); err != nil {
		return err
	}
	for _, entry := range o.StatusHistory {
		if err := validateStatus(entry.Status); err != nil {
			return err
		}
		if strings.TrimSpace(entry.Author) == "" {
			return ErrStatusAuthorRequired
		}
	}
	if !isValidMetadata(o.Metadata) {
		return ErrInvalidMetadata
	}

	return nil
}

// validateStatus validates status values.
func validateStatus(value Status) error {
	switch value {
	case StatusCancelled, StatusCreated, StatusPending, StatusHold, StatusCompleted:
		return nil
	default:
		return ErrStatusInvalid
	}
}

// normalizeShippingAddress canonicalizes shipping-address values.
func normalizeShippingAddress(value ShippingAddress) ShippingAddress {
	value.Address = strings.TrimSpace(value.Address)
	value.Address2 = strings.TrimSpace(value.Address2)
	value.Phone = strings.TrimSpace(value.Phone)
	value.CityCode = strings.TrimSpace(value.CityCode)

	return value
}

// normalizeMetadata canonicalizes metadata keys and values and drops empty keys.
func normalizeMetadata(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}

	normalized := make(map[string]string, len(values))
	for key, value := range values {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		normalized[trimmedKey] = strings.TrimSpace(value)
	}
	if len(normalized) == 0 {
		return nil
	}

	return normalized
}

// isValidMetadata reports whether metadata keys and values satisfy size constraints.
func isValidMetadata(values map[string]string) bool {
	for key, value := range values {
		if len(strings.TrimSpace(key)) > 128 || len(strings.TrimSpace(value)) > 2048 {
			return false
		}
	}

	return true
}
