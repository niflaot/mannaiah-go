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

// updateCommentRequest defines request payload for order-comment updates.
type updateCommentRequest struct {
	// Author defines optional comment author values.
	Author *string `json:"author,omitempty"`
	// Comment defines optional comment text values.
	Comment *string `json:"comment,omitempty"`
	// Internal defines optional comment visibility values.
	Internal *bool `json:"internal,omitempty"`
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

// updateComment handles order-comment update endpoints.
func (h *Handler) updateComment(ctx corehttp.Context) error {
	var request updateCommentRequest
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	entity, err := h.service.UpdateComment(ctx.Context(), ctx.Params("id"), ctx.Params("commentId"), ordersapplication.UpdateCommentCommand{
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

// deleteComment handles order-comment delete endpoints.
func (h *Handler) deleteComment(ctx corehttp.Context) error {
	entity, err := h.service.DeleteComment(ctx.Context(), ctx.Params("id"), ctx.Params("commentId"), ordersapplication.DeleteCommentCommand{
		Source: resolveCommandSource(ctx, ""),
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(entity)
}
