package http

import (
	"strings"

	corehttp "mannaiah/module/core/http"
)

// listCarriers handles carrier listing requests.
func (h *Handler) listCarriers(ctx corehttp.Context) error {
	rows, err := h.carriers.List(ctx.Context())
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(map[string]any{"data": rows, "total": len(rows)})
}

// getCarrier handles carrier by-id requests.
func (h *Handler) getCarrier(ctx corehttp.Context) error {
	carrier, err := h.carriers.Get(ctx.Context(), strings.TrimSpace(ctx.Params("id")))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(carrier)
}
