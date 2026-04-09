package http

import (
	"fmt"
	"strings"

	corehttp "mannaiah/module/core/http"
	dispatchservice "mannaiah/module/shipping/application/dispatch/service"
	"mannaiah/module/shipping/domain"
)

// createBatchRequest defines batch creation request payload values.
type createBatchRequest struct {
	// CarrierID defines carrier identifier values.
	CarrierID string `json:"carrierId"`
}

// draftMarkRequest defines draft mark creation request payload values.
type draftMarkRequest struct {
	// QuotationID defines the optional quotation reference attached to this draft.
	QuotationID string `json:"quotationId"`
	// QuotedFreightCost defines the freight cost snapshot from the quotation.
	QuotedFreightCost float64 `json:"quotedFreightCost"`
	// OrderID defines order identifier values.
	OrderID string `json:"orderId"`
	// Sender defines sender address values.
	Sender domain.Address `json:"sender"`
	// Recipient defines recipient address values.
	Recipient domain.Address `json:"recipient"`
	// Units defines package-unit values.
	Units []draftMarkUnitRequest `json:"units"`
	// DeclaredValue defines declared shipment value amounts.
	DeclaredValue float64 `json:"declaredValue"`
	// PaymentForm defines payment arrangement values.
	PaymentForm string `json:"paymentForm"`
	// CollectOnDeliveryAmount defines requested cash-on-delivery collection amounts.
	CollectOnDeliveryAmount float64 `json:"collectOnDeliveryAmount"`
	// ShipmentMode defines the delivery mode for this draft mark (parcel or express).
	ShipmentMode domain.ShipmentMode `json:"shipmentMode"`
	// Observations defines observation values.
	Observations string `json:"observations"`
	// TrackingNumber defines optional manual tracking-number values.
	TrackingNumber string `json:"trackingNumber"`
	// DocumentType defines optional manual document-type values.
	DocumentType domain.MarkDocumentType `json:"documentType"`
	// DocumentRef defines optional manual document-reference values.
	DocumentRef string `json:"documentRef"`
	// ManifestType defines optional manual manifest document-type values.
	ManifestType domain.MarkDocumentType `json:"manifestType"`
	// ManifestRef defines optional manual manifest document-reference values.
	ManifestRef string `json:"manifestRef"`
	// CustomTrackingURL defines an optional operator-provided tracking URL override for this mark.
	CustomTrackingURL string `json:"customTrackingUrl"`
}

// updateDraftMarkRequest defines manual draft-completion payload values.
type updateDraftMarkRequest struct {
	// QuotedFreightCost defines the manual freight cost entered by the operator.
	QuotedFreightCost float64 `json:"quotedFreightCost"`
	// Observations defines the manual carrier label stored as a normalized slug.
	Observations string `json:"observations"`
	// TrackingNumber defines the manual tracking-number value.
	TrackingNumber string `json:"trackingNumber"`
	// CustomTrackingURL defines the operator-provided tracking URL override.
	CustomTrackingURL string `json:"customTrackingUrl"`
}

// createBatchMarkRequest defines one payload for quoted/direct batch mark creation from one quotation id.
type createBatchMarkRequest struct {
	// Batch defines the target batch identifier.
	Batch string `json:"batch"`
	// Direct defines whether the mark should be materialized immediately.
	Direct bool `json:"direct"`
	// QuotationID defines quotation reference values used to seed this mark.
	QuotationID string `json:"quotationId"`
}

// draftMarkUnitRequest defines package-unit values within a draft mark request.
type draftMarkUnitRequest struct {
	// Description defines package description values.
	Description string `json:"description"`
	// PackageType defines package-type values.
	PackageType string `json:"packageType"`
	// Dimensions defines package dimension values.
	Dimensions domain.Dimensions `json:"dimensions"`
}

// batchListResponse defines batch list response values.
type batchListResponse struct {
	// Data defines batch rows.
	Data []domain.DispatchBatch `json:"data"`
	// Total defines total row count values.
	Total int64 `json:"total"`
	// Page defines current page values.
	Page int `json:"page"`
	// Limit defines page-size values.
	Limit int `json:"limit"`
}

// createBatch handles batch creation requests.
func (h *Handler) createBatch(ctx corehttp.Context) error {
	request := createBatchRequest{}
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}
	createdBy := "system"
	if h.authorizer != nil {
		createdBy = h.authorizer.Subject(ctx.Context(), ctx.GetHeader("Authorization"))
	}
	batch, err := h.batches.Create(ctx.Context(), dispatchservice.CreateBatchCommand{
		CarrierID: strings.TrimSpace(request.CarrierID),
		CreatedBy: createdBy,
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(201).JSON(batch)
}

// getBatch handles batch by-id requests.
func (h *Handler) getBatch(ctx corehttp.Context) error {
	batch, err := h.batches.Get(ctx.Context(), strings.TrimSpace(ctx.Params("id")))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(batch)
}

// listBatches handles dispatch batch listing requests.
func (h *Handler) listBatches(ctx corehttp.Context) error {
	page, limit := parsePageLimit(ctx, 20)
	rows, total, err := h.batches.List(ctx.Context(), dispatchservice.ListQuery{
		CarrierID: strings.TrimSpace(ctx.Query("carrierID")),
		Status:    domain.BatchStatus(strings.TrimSpace(ctx.Query("status"))),
		Page:      page,
		Limit:     limit,
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(batchListResponse{Data: rows, Total: total, Page: page, Limit: limit})
}

// addBatchMark handles draft mark creation for one open batch.
func (h *Handler) addBatchMark(ctx corehttp.Context) error {
	request := draftMarkRequest{}
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}
	units := mapDraftMarkUnits(request.Units)
	mark, err := h.batches.DraftMark(ctx.Context(), dispatchservice.DraftMarkCommand{
		BatchID:                 strings.TrimSpace(ctx.Params("id")),
		QuotationID:             strings.TrimSpace(request.QuotationID),
		QuotedFreightCost:       request.QuotedFreightCost,
		OrderID:                 strings.TrimSpace(request.OrderID),
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
		CustomTrackingURL:       optionalString(strings.TrimSpace(request.CustomTrackingURL)),
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(201).JSON(mark)
}

// createBatchMark handles quoted/direct mark creation with explicit batch id in body.
func (h *Handler) createBatchMark(ctx corehttp.Context) error {
	request := createBatchMarkRequest{}
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}
	mark, err := h.batches.CreateBatchMarkFromQuotation(ctx.Context(), dispatchservice.CreateBatchMarkFromQuotationCommand{
		BatchID:     strings.TrimSpace(request.Batch),
		QuotationID: strings.TrimSpace(request.QuotationID),
		Direct:      request.Direct,
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(201).JSON(mark)
}

// updateBatchMark handles manual QUOTED draft completion for one open manual batch.
func (h *Handler) updateBatchMark(ctx corehttp.Context) error {
	request := updateDraftMarkRequest{}
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}
	mark, err := h.batches.UpdateDraftMark(ctx.Context(), dispatchservice.UpdateDraftMarkCommand{
		BatchID:           strings.TrimSpace(ctx.Params("id")),
		MarkID:            strings.TrimSpace(ctx.Params("markID")),
		QuotedFreightCost: request.QuotedFreightCost,
		Observations:      strings.TrimSpace(request.Observations),
		TrackingNumber:    strings.TrimSpace(request.TrackingNumber),
		CustomTrackingURL: optionalString(strings.TrimSpace(request.CustomTrackingURL)),
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(mark)
}

// removeBatchMark handles QUOTED mark removal from one batch.
func (h *Handler) removeBatchMark(ctx corehttp.Context) error {
	batch, err := h.batches.RemoveDraftMark(ctx.Context(), strings.TrimSpace(ctx.Params("id")), strings.TrimSpace(ctx.Params("markID")))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(batch)
}

// closeBatch handles batch close requests.
func (h *Handler) closeBatch(ctx corehttp.Context) error {
	batch, err := h.batches.Close(ctx.Context(), strings.TrimSpace(ctx.Params("id")))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(batch)
}

// batchManifestDocument handles merged batch manifest document downloads.
func (h *Handler) batchManifestDocument(ctx corehttp.Context) error {
	batchID := strings.TrimSpace(ctx.Params("id"))
	payload, err := h.batches.ManifestDocument(ctx.Context(), batchID)
	if err != nil {
		return h.mapError(err)
	}

	ctx.SetHeader("Content-Type", "application/pdf")
	ctx.SetHeader("Content-Disposition", fmt.Sprintf("inline; filename=\"batch-%s-manifests.pdf\"", batchID))
	ctx.SetHeader("Cache-Control", "private, max-age=300")

	return ctx.Status(200).SendBytes(payload)
}

func mapDraftMarkUnits(unitsRequest []draftMarkUnitRequest) []domain.PackageUnit {
	units := make([]domain.PackageUnit, 0, len(unitsRequest))
	for _, unit := range unitsRequest {
		units = append(units, domain.PackageUnit{
			Description: strings.TrimSpace(unit.Description),
			PackageType: strings.TrimSpace(unit.PackageType),
			Dimensions:  unit.Dimensions,
		})
	}

	return units
}
