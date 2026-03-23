package http

import (
	"strings"

	corehttp "mannaiah/module/core/http"
	dispatchservice "mannaiah/module/shipping/application/dispatch/service"
	"mannaiah/module/shipping/domain"
)

// createBatchRequest defines batch creation request payload values.
type createBatchRequest struct {
	// Name defines batch name values.
	Name string `json:"name"`
	// CarrierID defines carrier identifier values.
	CarrierID string `json:"carrierId"`
}

// addBatchMarksRequest defines batch mark assignment request payload values.
type addBatchMarksRequest struct {
	// MarkIDs defines assigned mark identifier values.
	MarkIDs []string `json:"markIds"`
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
	batch, err := h.batches.Create(ctx.Context(), dispatchservice.CreateBatchCommand{
		Name:      strings.TrimSpace(request.Name),
		CarrierID: strings.TrimSpace(request.CarrierID),
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

// addBatchMarks handles mark assignment to one batch.
func (h *Handler) addBatchMarks(ctx corehttp.Context) error {
	request := addBatchMarksRequest{}
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}
	batch, err := h.batches.AddMarks(ctx.Context(), dispatchservice.AddMarksCommand{
		BatchID: strings.TrimSpace(ctx.Params("id")),
		MarkIDs: request.MarkIDs,
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(batch)
}

// removeBatchMark handles mark removal from one batch.
func (h *Handler) removeBatchMark(ctx corehttp.Context) error {
	batch, err := h.batches.RemoveMark(ctx.Context(), strings.TrimSpace(ctx.Params("id")), strings.TrimSpace(ctx.Params("markID")))
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
