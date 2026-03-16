package http

import (
	"strconv"
	"strings"

	corehttp "mannaiah/module/core/http"
)

// listDeliveries handles campaign delivery list requests.
func (h *Handler) listDeliveries(ctx corehttp.Context) error {
	id := strings.TrimSpace(ctx.Params("id"))
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	limit, _ := strconv.Atoi(ctx.Query("limit", "50"))

	result, err := h.service.ListDeliveries(ctx.Context(), id, page, limit)
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(result)
}
