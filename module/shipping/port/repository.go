package port

import (
	"context"
	"time"

	"mannaiah/module/shipping/domain"
)

// MarkListQuery defines shipping-mark listing query values.
type MarkListQuery struct {
	// OrderID filters rows by order identifier.
	OrderID string
	// BatchID filters rows by batch identifier.
	BatchID string
	// Page defines 1-based page values.
	Page int
	// Limit defines page-size values.
	Limit int
}

// BatchListQuery defines dispatch-batch listing query values.
type BatchListQuery struct {
	// CarrierID filters rows by carrier identifier.
	CarrierID string
	// Status filters rows by batch status values.
	Status domain.BatchStatus
	// Page defines 1-based page values.
	Page int
	// Limit defines page-size values.
	Limit int
}

// QuotationRecord defines persisted quotation records.
type QuotationRecord struct {
	// ID defines quotation identifier values.
	ID string `json:"id"`
	// OrderID defines optional order identifier values.
	OrderID string `json:"orderId"`
	// OrderIdentifier defines optional external order identifier values (e.g. WooCommerce number).
	OrderIdentifier string `json:"orderIdentifier"`
	// CarrierID defines carrier identifier values.
	CarrierID string `json:"carrierId"`
	// OriginCityCode defines origin city-code values.
	OriginCityCode string `json:"originCityCode"`
	// DestCityCode defines destination city-code values.
	DestCityCode string `json:"destCityCode"`
	// FreightCost defines carrier-reported freight-cost values.
	FreightCost float64 `json:"freightCost"`
	// CollectOnDeliveryFeeAmount defines the COD fee amount applied to this quotation.
	CollectOnDeliveryFeeAmount float64 `json:"collectOnDeliveryFeeAmount,omitempty"`
	// EstimatedDays defines estimated delivery-day values.
	EstimatedDays int `json:"estimatedDays"`
	// CurrencyCode defines currency-code values.
	CurrencyCode string `json:"currencyCode"`
	// ExpiresAt defines quotation expiration timestamps.
	ExpiresAt time.Time `json:"expiresAt"`
	// Units defines the package units used in the quotation request.
	Units []domain.PackageUnit `json:"units,omitempty"`
	// RequestSnapshot defines serialized quotation request payload values.
	RequestSnapshot string `json:"requestSnapshot"`
	// RawResponse defines provider raw response payload values.
	RawResponse string `json:"rawResponse"`
	// CreatedAt defines row creation timestamps.
	CreatedAt time.Time `json:"createdAt"`
}

// ShippingMarkRepository defines shipping-mark persistence behavior.
type ShippingMarkRepository interface {
	// Create creates one shipping mark.
	Create(ctx context.Context, mark *domain.ShippingMark) error
	// GetByID loads one shipping mark by identifier.
	GetByID(ctx context.Context, id string) (*domain.ShippingMark, error)
	// GetByTrackingNumber loads one shipping mark by tracking number.
	GetByTrackingNumber(ctx context.Context, trackingNumber string) (*domain.ShippingMark, error)
	// ListByOrderID loads shipping marks by order identifier.
	ListByOrderID(ctx context.Context, orderID string) ([]domain.ShippingMark, error)
	// ListByBatchID loads shipping marks by batch identifier.
	ListByBatchID(ctx context.Context, batchID string) ([]domain.ShippingMark, error)
	// Update updates one shipping mark.
	Update(ctx context.Context, mark *domain.ShippingMark) error
	// Delete permanently deletes one shipping mark and its units by identifier.
	Delete(ctx context.Context, id string) error
	// List lists marks using pagination and filters.
	List(ctx context.Context, query MarkListQuery) ([]domain.ShippingMark, int64, error)
}

// DispatchBatchRepository defines dispatch-batch persistence behavior.
type DispatchBatchRepository interface {
	// Create creates one dispatch batch.
	Create(ctx context.Context, batch *domain.DispatchBatch) error
	// GetByID loads one dispatch batch by identifier.
	GetByID(ctx context.Context, id string) (*domain.DispatchBatch, error)
	// Close closes one dispatch batch.
	Close(ctx context.Context, id string) error
	// AddMark assigns one mark to the batch.
	AddMark(ctx context.Context, batchID string, markID string) error
	// RemoveMark removes one mark from the batch.
	RemoveMark(ctx context.Context, batchID string, markID string) error
	// List lists dispatch batches using pagination and filters.
	List(ctx context.Context, query BatchListQuery) ([]domain.DispatchBatch, int64, error)
}

// QuotationRepository defines quotation persistence behavior.
type QuotationRepository interface {
	// Create creates one quotation audit record and returns the persisted record ID.
	// If an equivalent non-expired quotation already exists the existing ID is returned.
	Create(ctx context.Context, record QuotationRecord) (string, error)
	// GetByID loads one quotation record by identifier.
	GetByID(ctx context.Context, id string) (*QuotationRecord, error)
	// ListByOrderID lists quotation records by order identifier.
	ListByOrderID(ctx context.Context, orderID string) ([]QuotationRecord, error)
	// GetLatestByOrderAndCarrier returns the most recent non-expired quotation for an order and carrier.
	GetLatestByOrderAndCarrier(ctx context.Context, orderID string, carrierID string) (*QuotationRecord, error)
	// DeleteExpired deletes all quotation records whose expiration timestamp is in the past.
	DeleteExpired(ctx context.Context) (int64, error)
}
