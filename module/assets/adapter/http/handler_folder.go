package http

import (
	assetsapplication "mannaiah/module/assets/application"
	"mannaiah/module/assets/domain"
	corehttp "mannaiah/module/core/http"
)

// createFolderRequest defines request payload for folder creation.
type createFolderRequest struct {
	// Name defines folder names.
	Name string `json:"name"`
	// ParentFolderID defines optional parent-folder assignments.
	ParentFolderID string `json:"parentFolderId"`
	// Tags defines optional folder tags.
	Tags []domain.Tag `json:"tags"`
}

// updateFolderRequest defines request payload for folder updates.
type updateFolderRequest struct {
	// Name defines optional folder-name updates.
	Name *string `json:"name"`
	// ParentFolderID defines optional parent-folder assignment updates.
	ParentFolderID *string `json:"parentFolderId"`
	// Tags defines optional folder-tag updates.
	Tags *[]domain.Tag `json:"tags"`
}

// createFolder handles folder creation endpoints.
func (h *Handler) createFolder(ctx corehttp.Context) error {
	var request createFolderRequest
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	entity, err := h.service.CreateFolder(ctx.Context(), assetsapplication.CreateFolderCommand{
		Name:           request.Name,
		ParentFolderID: request.ParentFolderID,
		Tags:           request.Tags,
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(201).JSON(entity)
}

// findFolders handles paginated folder list endpoints.
func (h *Handler) findFolders(ctx corehttp.Context) error {
	page, err := parseIntQuery(ctx, "page", 1)
	if err != nil {
		return corehttp.NewAppError(400, "invalid_page", err)
	}
	limit, err := parseIntQuery(ctx, "limit", 10)
	if err != nil {
		return corehttp.NewAppError(400, "invalid_limit", err)
	}

	result, listErr := h.service.ListFolders(ctx.Context(), assetsapplication.ListQuery{
		Page:           page,
		Limit:          limit,
		Filters:        ctx.Query("filters"),
		ParentFolderID: ctx.Query("parentFolderId"),
	})
	if listErr != nil {
		return h.mapError(listErr)
	}

	return ctx.Status(200).JSON(folderListResponse{
		Data: result.Data,
		Meta: listResponseMeta{Page: result.Page, Total: result.Total, Limit: result.Limit},
	})
}

// findFolderByID handles folder-by-id retrieval endpoints.
func (h *Handler) findFolderByID(ctx corehttp.Context) error {
	entity, err := h.service.GetFolder(ctx.Context(), ctx.Params("id"))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(entity)
}

// updateFolder handles folder update endpoints.
func (h *Handler) updateFolder(ctx corehttp.Context) error {
	var request updateFolderRequest
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	entity, err := h.service.UpdateFolder(ctx.Context(), ctx.Params("id"), assetsapplication.UpdateFolderCommand{
		Name:           request.Name,
		ParentFolderID: request.ParentFolderID,
		Tags:           request.Tags,
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(entity)
}

// deleteFolder handles folder delete endpoints.
func (h *Handler) deleteFolder(ctx corehttp.Context) error {
	if err := h.service.DeleteFolder(ctx.Context(), ctx.Params("id")); err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(deleteResponse{Status: "deleted"})
}
