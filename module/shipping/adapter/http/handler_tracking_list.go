package http

import (
	"strings"

	corehttp "mannaiah/module/core/http"
	trackingservice "mannaiah/module/shipping/application/tracking/service"
)

// trackingListResponse defines tracking list response values.
type trackingListResponse struct {
	// Data defines tracking-summary rows.
	Data []trackingservice.ListItem `json:"data"`
	// Total defines total row count values.
	Total int64 `json:"total"`
	// Page defines current page values.
	Page int `json:"page"`
	// Limit defines page-size values.
	Limit int `json:"limit"`
}

// listTracking handles paginated shipment tracking listing requests.
func (h *Handler) listTracking(ctx corehttp.Context) error {
	page, limit := parsePageLimit(ctx, 20)
	rows, total, err := h.tracking.List(ctx.Context(), trackingservice.ListQuery{
		Term:   strings.TrimSpace(ctx.Query("term")),
		Status: strings.TrimSpace(ctx.Query("status")),
		Page:   page,
		Limit:  limit,
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(trackingListResponse{
		Data:  rows,
		Total: total,
		Page:  page,
		Limit: limit,
	})
}
