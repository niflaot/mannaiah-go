package http

import (
	"strings"

	corehttp "mannaiah/module/core/http"
)

// getTracking handles tracking history requests.
func (h *Handler) getTracking(ctx corehttp.Context) error {
	carrierID := strings.TrimSpace(ctx.Query("carrier"))
	trackingNumber := strings.TrimSpace(ctx.Params("trackingNumber"))

	history, err := h.tracking.Get(ctx.Context(), carrierID, trackingNumber)
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(history)
}
