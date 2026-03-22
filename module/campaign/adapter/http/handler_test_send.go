package http

import (
	"strings"

	"mannaiah/module/campaign/application"
	corehttp "mannaiah/module/core/http"
)

// testSendRequest defines test send request payload values.
type testSendRequest struct {
	// ContactID defines the contact identifier used for template personalization.
	ContactID string `json:"contactId"`
	// Email defines the override recipient email address for the test delivery.
	Email string `json:"email"`
}

// testSend handles campaign test-send requests.
func (h *Handler) testSend(ctx corehttp.Context) error {
	request := testSendRequest{}
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	result, err := h.service.TestSend(ctx.Context(), strings.TrimSpace(ctx.Params("id")), application.TestSendCommand{
		ContactID: strings.TrimSpace(request.ContactID),
		Email:     strings.TrimSpace(request.Email),
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(202).JSON(result)
}
