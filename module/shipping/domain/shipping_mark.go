package domain

import (
	"strings"
	"time"
)

// MarkStatus defines shipping-mark lifecycle status values.
type MarkStatus string

const (
	// MarkStatusPending defines pending mark statuses (legacy standalone flow).
	MarkStatusPending MarkStatus = "PENDING"
	// MarkStatusGenerated defines generated mark statuses (legacy standalone flow).
	MarkStatusGenerated MarkStatus = "GENERATED"
	// MarkStatusFailed defines failed mark statuses.
	MarkStatusFailed MarkStatus = "FAILED"
	// MarkStatusVoided defines voided mark statuses.
	MarkStatusVoided MarkStatus = "VOIDED"
	// MarkStatusQuoted defines draft marks staged in a batch with a quotation reference.
	MarkStatusQuoted MarkStatus = "QUOTED"
	// MarkStatusCreated defines marks successfully submitted to the carrier at batch close.
	MarkStatusCreated MarkStatus = "CREATED"
	// MarkStatusRemoved defines draft marks removed from a batch before carrier submission.
	MarkStatusRemoved MarkStatus = "REMOVED"
)

// MarkDocumentType defines mark artifact-storage mode values.
type MarkDocumentType string

const (
	// MarkDocumentLink defines URL-based document references.
	MarkDocumentLink MarkDocumentType = "LINK"
	// MarkDocumentFile defines binary/object-storage document references.
	MarkDocumentFile MarkDocumentType = "FILE"
)

// ShippingMark defines generated shipping-mark values.
type ShippingMark struct {
	// ID defines mark identifier values.
	ID string `json:"id"`
	// OrderID defines order identifier values.
	OrderID string `json:"orderId"`
	// CarrierID defines carrier identifier values.
	CarrierID string `json:"carrierId"`
	// TrackingNumber defines carrier tracking-number values.
	TrackingNumber string `json:"trackingNumber,omitempty"`
	// Status defines mark status values.
	Status MarkStatus `json:"status"`
	// DocumentType defines mark document-type values.
	DocumentType MarkDocumentType `json:"documentType,omitempty"`
	// DocumentRef defines mark document-reference values.
	DocumentRef string `json:"documentRef,omitempty"`
	// Sender defines sender address details.
	Sender Address `json:"sender"`
	// Recipient defines recipient address details.
	Recipient Address `json:"recipient"`
	// Units defines package units.
	Units []PackageUnit `json:"units"`
	// TotalWeight defines aggregate real-weight values.
	TotalWeight float64 `json:"totalWeight"`
	// TotalVolumetricWeight defines aggregate volumetric-weight values.
	TotalVolumetricWeight float64 `json:"totalVolumetricWeight"`
	// DeclaredValue defines declared value amounts.
	DeclaredValue float64 `json:"declaredValue"`
	// PaymentForm defines freight payment-form values.
	PaymentForm string `json:"paymentForm,omitempty"`
	// CollectOnDeliveryAmount defines requested cash-on-delivery collection amounts.
	CollectOnDeliveryAmount float64 `json:"collectOnDeliveryAmount,omitempty"`
	// CollectOnDeliveryFeePercent defines applied COD fee percentage values.
	CollectOnDeliveryFeePercent float64 `json:"collectOnDeliveryFeePercent,omitempty"`
	// CollectOnDeliveryChargedAmount defines final COD amount requested from the carrier.
	CollectOnDeliveryChargedAmount float64 `json:"collectOnDeliveryChargedAmount,omitempty"`
	// Observations defines observation values.
	Observations string `json:"observations,omitempty"`
	// DispatchBatchID defines assigned dispatch batch identifiers.
	DispatchBatchID *string `json:"dispatchBatchId,omitempty"`
	// QuotationID defines the optional quotation used when drafting this mark.
	QuotationID *string `json:"quotationId,omitempty"`
	// QuotedFreightCost defines the freight cost snapshot from the quotation at draft time.
	QuotedFreightCost float64 `json:"quotedFreightCost,omitempty"`
	// DraftSnapshot defines a JSON snapshot of all mark fields captured before carrier submission.
	DraftSnapshot string `json:"draftSnapshot,omitempty"`
	// CreatedAt defines row creation timestamps.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt defines row update timestamps.
	UpdatedAt time.Time `json:"updatedAt"`
}

// Normalize normalizes shipping-mark fields.
func (m ShippingMark) Normalize() ShippingMark {
	units := make([]PackageUnit, 0, len(m.Units))
	totalWeight := 0.0
	totalVolWeight := 0.0
	declaredValue := m.DeclaredValue
	for _, unit := range m.Units {
		normalized := unit.Normalize()
		units = append(units, normalized)
		totalWeight += normalized.Dimensions.RealWeightKG
		totalVolWeight += normalized.Dimensions.VolumetricWeightKG
		if declaredValue <= 0 {
			declaredValue += normalized.Dimensions.DeclaredValueCOP
		}
	}
	if declaredValue < 0 {
		declaredValue = 0
	}
	collectOnDeliveryAmount := m.CollectOnDeliveryAmount
	if collectOnDeliveryAmount < 0 {
		collectOnDeliveryAmount = 0
	}
	collectOnDeliveryFeePercent := m.CollectOnDeliveryFeePercent
	if collectOnDeliveryFeePercent < 0 {
		collectOnDeliveryFeePercent = 0
	}
	if collectOnDeliveryFeePercent > 100 {
		collectOnDeliveryFeePercent = 100
	}
	collectOnDeliveryChargedAmount := m.CollectOnDeliveryChargedAmount
	if collectOnDeliveryChargedAmount < 0 {
		collectOnDeliveryChargedAmount = 0
	}
	if collectOnDeliveryAmount <= 0 {
		collectOnDeliveryFeePercent = 0
		collectOnDeliveryChargedAmount = 0
	}
	if collectOnDeliveryAmount > 0 && collectOnDeliveryChargedAmount <= 0 {
		collectOnDeliveryChargedAmount = collectOnDeliveryAmount
	}

	quotedFreightCost := m.QuotedFreightCost
	if quotedFreightCost < 0 {
		quotedFreightCost = 0
	}
	copy := ShippingMark{
		ID:                             strings.TrimSpace(m.ID),
		OrderID:                        strings.TrimSpace(m.OrderID),
		CarrierID:                      strings.TrimSpace(m.CarrierID),
		TrackingNumber:                 strings.TrimSpace(m.TrackingNumber),
		Status:                         m.Status,
		DocumentType:                   m.DocumentType,
		DocumentRef:                    strings.TrimSpace(m.DocumentRef),
		Sender:                         m.Sender.Normalize(),
		Recipient:                      m.Recipient.Normalize(),
		Units:                          units,
		TotalWeight:                    round2(totalWeight),
		TotalVolumetricWeight:          round2(totalVolWeight),
		DeclaredValue:                  round2(declaredValue),
		PaymentForm:                    strings.TrimSpace(m.PaymentForm),
		CollectOnDeliveryAmount:        round2(collectOnDeliveryAmount),
		CollectOnDeliveryFeePercent:    round2(collectOnDeliveryFeePercent),
		CollectOnDeliveryChargedAmount: round2(collectOnDeliveryChargedAmount),
		Observations:                   strings.TrimSpace(m.Observations),
		DispatchBatchID:                m.DispatchBatchID,
		QuotationID:                    m.QuotationID,
		QuotedFreightCost:              round2(quotedFreightCost),
		DraftSnapshot:                  m.DraftSnapshot,
		CreatedAt:                      m.CreatedAt,
		UpdatedAt:                      m.UpdatedAt,
	}
	if copy.Status == "" {
		copy.Status = MarkStatusPending
	}

	return copy
}

// Validate validates shipping-mark fields.
func (m ShippingMark) Validate() error {
	normalized := m.Normalize()
	if normalized.ID == "" {
		return ErrInvalidID
	}
	if normalized.OrderID == "" {
		return ErrInvalidID
	}
	if normalized.CarrierID == "" {
		return ErrInvalidCarrierID
	}
	if len(normalized.Units) == 0 {
		return ErrInvalidID
	}

	return nil
}
