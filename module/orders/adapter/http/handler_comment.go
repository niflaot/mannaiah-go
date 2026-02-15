package http

import (
	corehttp "mannaiah/module/core/http"
	ordersapplication "mannaiah/module/orders/application"
)

// addCommentRequest defines request payload for order comments.
type addCommentRequest struct {
	// Author defines comment author values.
	Author string `json:"author"`
	// Comment defines comment text values.
	Comment string `json:"comment"`
	// Internal reports whether comments are internal-only.
	Internal bool `json:"internal"`
	// Source defines optional mutation source values.
	Source string `json:"source,omitempty"`
}

// addComment handles order-comment append endpoints.
func (h *Handler) addComment(ctx corehttp.Context) error {
	var request addCommentRequest
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	entity, err := h.service.AddComment(ctx.Context(), ctx.Params("id"), ordersapplication.AddCommentCommand{
		Author:   request.Author,
		Comment:  request.Comment,
		Internal: request.Internal,
		Source:   resolveCommandSource(ctx, request.Source),
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(entity)
}
