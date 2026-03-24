package http

import (
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
	units := make([]domain.PackageUnit, 0, len(request.Units))
	for _, unit := range request.Units {
		units = append(units, domain.PackageUnit{
			Description: strings.TrimSpace(unit.Description),
			PackageType: strings.TrimSpace(unit.PackageType),
			Dimensions:  unit.Dimensions,
		})
	}
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
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(201).JSON(mark)
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
