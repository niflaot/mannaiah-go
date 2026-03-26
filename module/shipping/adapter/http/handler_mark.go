package http

import (
	"strings"

	corehttp "mannaiah/module/core/http"
	markservice "mannaiah/module/shipping/application/mark/service"
	"mannaiah/module/shipping/domain"
)

// getOrderDispatch handles order dispatch provisioning status requests.
func (h *Handler) getOrderDispatch(ctx corehttp.Context) error {
	result, err := h.marks.QueryDispatch(ctx.Context(), markservice.DispatchQuery{
		OrderID: strings.TrimSpace(ctx.Params("orderID")),
	})
	if err != nil {
		return h.mapError(err)
	}
	batchID := ""
	if result.BatchID != nil {
		batchID = *result.BatchID
	}

	return ctx.Status(200).JSON(orderDispatchResponse{
		OrderID:     result.OrderID,
		Provisioned: result.Provisioned,
		MarkID:      result.MarkID,
		BatchID:     batchID,
		Status:      result.Status,
	})
}

// markRequest defines shipping-mark request payload values.
type markRequest struct {
	// OrderID defines order identifier values.
	OrderID string `json:"orderId"`
	// CarrierID defines carrier identifier values.
	CarrierID string `json:"carrierId"`
	// Sender defines sender address values.
	Sender domain.Address `json:"sender"`
	// Recipient defines recipient address values.
	Recipient domain.Address `json:"recipient"`
	// Units defines package-unit values.
	Units []markUnitRequest `json:"units"`
	// DeclaredValue defines declared shipment value amounts.
	DeclaredValue float64 `json:"declaredValue"`
	// PaymentForm defines payment arrangement values.
	PaymentForm string `json:"paymentForm"`
	// CollectOnDeliveryAmount defines requested cash-on-delivery collection amounts.
	CollectOnDeliveryAmount float64 `json:"collectOnDeliveryAmount"`
	// Observations defines observation values.
	Observations string `json:"observations"`
	// TrackingNumber defines optional manual tracking-number values.
	TrackingNumber string `json:"trackingNumber"`
	// DocumentType defines optional manual document-type values.
	DocumentType domain.MarkDocumentType `json:"documentType"`
	// DocumentRef defines optional manual document reference values.
	DocumentRef string `json:"documentRef"`
	// ManifestType defines optional manual manifest document-type values.
	ManifestType domain.MarkDocumentType `json:"manifestType"`
	// ManifestRef defines optional manual manifest document reference values.
	ManifestRef string `json:"manifestRef"`
	// ShipmentMode defines the delivery mode for this mark (parcel or express).
	ShipmentMode domain.ShipmentMode `json:"shipmentMode"`
}

// markUnitRequest defines shipping-mark package-unit payload values.
type markUnitRequest struct {
	// Description defines package description values.
	Description string `json:"description"`
	// PackageType defines package-type values.
	PackageType string `json:"packageType"`
	// Dimensions defines package dimension values.
	Dimensions domain.Dimensions `json:"dimensions"`
}

// markListResponse defines shipping mark list response values.
type markListResponse struct {
	// Data defines mark rows.
	Data []domain.ShippingMark `json:"data"`
	// Total defines total row count values.
	Total int64 `json:"total"`
	// Page defines current page values.
	Page int `json:"page"`
	// Limit defines page-size values.
	Limit int `json:"limit"`
}

// voidMarkRequest defines mark void request payload values.
type voidMarkRequest struct {
	// Reason defines void reason values.
	Reason string `json:"reason"`
}

// orderDispatchResponse defines order dispatch provisioning status response payload values.
type orderDispatchResponse struct {
	// OrderID defines the queried order identifier.
	OrderID string `json:"orderId"`
	// Provisioned reports whether the order has an active mark in the dispatch workflow.
	Provisioned bool `json:"provisioned"`
	// MarkID defines the active mark identifier when provisioned.
	MarkID string `json:"markId,omitempty"`
	// BatchID defines the associated dispatch batch identifier when provisioned.
	BatchID string `json:"batchId,omitempty"`
	// Status defines the active mark status when provisioned.
	Status domain.MarkStatus `json:"status,omitempty"`
}

// createMark handles mark creation requests.
func (h *Handler) createMark(ctx corehttp.Context) error {
	request := markRequest{}
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}
	units := make([]domain.PackageUnit, 0, len(request.Units))
	for _, unit := range request.Units {
		units = append(units, domain.PackageUnit{
			Description: strings.TrimSpace(unit.Description),
			PackageType: strings.TrimSpace(unit.PackageType),
			Dimensions:  unit.Dimensions,
		})
	}
	mark, err := h.marks.Generate(ctx.Context(), markservice.GenerateCommand{
		OrderID:                 strings.TrimSpace(request.OrderID),
		CarrierID:               strings.TrimSpace(request.CarrierID),
		Sender:                  request.Sender,
		Recipient:               request.Recipient,
		Units:                   units,
		DeclaredValue:           request.DeclaredValue,
		PaymentForm:             strings.TrimSpace(request.PaymentForm),
		CollectOnDeliveryAmount: request.CollectOnDeliveryAmount,
		ShipmentMode:            request.ShipmentMode,
		Observations:            strings.TrimSpace(request.Observations),
		TrackingNumber:          strings.TrimSpace(request.TrackingNumber),
		DocumentType:            request.DocumentType,
		DocumentRef:             strings.TrimSpace(request.DocumentRef),
		ManifestType:            request.ManifestType,
		ManifestRef:             strings.TrimSpace(request.ManifestRef),
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(201).JSON(mark)
}

// getMark handles mark by-id requests.
func (h *Handler) getMark(ctx corehttp.Context) error {
	mark, err := h.marks.Get(ctx.Context(), strings.TrimSpace(ctx.Params("id")))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(mark)
}

// listMarks handles shipping mark listing requests.
func (h *Handler) listMarks(ctx corehttp.Context) error {
	page, limit := parsePageLimit(ctx, 20)
	rows, total, err := h.marks.List(ctx.Context(), markservice.ListQuery{
		OrderID: strings.TrimSpace(ctx.Query("orderID")),
		BatchID: strings.TrimSpace(ctx.Query("batchID")),
		Page:    page,
		Limit:   limit,
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(markListResponse{Data: rows, Total: total, Page: page, Limit: limit})
}

// listRelatedMarks handles shipping mark related-list requests.
func (h *Handler) listRelatedMarks(ctx corehttp.Context) error {
	rows, err := h.marks.Related(ctx.Context(), strings.TrimSpace(ctx.Params("id")))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(markListResponse{
		Data:  rows,
		Total: int64(len(rows)),
		Page:  1,
		Limit: len(rows),
	})
}

// voidMark handles mark void requests.
func (h *Handler) voidMark(ctx corehttp.Context) error {
	request := voidMarkRequest{}
	_ = ctx.BodyParser(&request)

	mark, err := h.marks.Void(ctx.Context(), strings.TrimSpace(ctx.Params("id")), strings.TrimSpace(request.Reason))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(mark)
}
