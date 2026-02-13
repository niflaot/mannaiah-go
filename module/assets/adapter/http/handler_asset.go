package http

import (
	"io"
	"strings"

	assetsapplication "mannaiah/module/assets/application"
	"mannaiah/module/assets/domain"
	corehttp "mannaiah/module/core/http"
)

// updateAssetRequest defines request payload for asset update operations.
type updateAssetRequest struct {
	// Name defines target asset names.
	Name *string `json:"name"`
	// FolderID defines optional folder assignment updates.
	FolderID *string `json:"folderId"`
	// Tags defines optional tag replacement updates.
	Tags *[]domain.Tag `json:"tags"`
	// Metadata defines optional metadata replacement updates.
	Metadata *map[string]string `json:"metadata"`
}

// createAsset handles asset upload endpoints.
func (h *Handler) createAsset(ctx corehttp.Context) error {
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		return corehttp.NewAppError(400, "file_required", assetsapplication.ErrFileRequired)
	}

	file, err := fileHeader.Open()
	if err != nil {
		return corehttp.NewAppError(400, "invalid_file", err)
	}
	defer func() {
		_ = file.Close()
	}()

	body, err := io.ReadAll(file)
	if err != nil {
		return corehttp.NewAppError(400, "invalid_file", err)
	}

	name := strings.TrimSpace(ctx.FormValue("name"))
	mimeType := strings.TrimSpace(fileHeader.Header.Get("Content-Type"))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	tags, tagsErr := parseJSONField[[]domain.Tag](ctx.FormValue("tags"))
	if tagsErr != nil {
		return corehttp.NewAppError(400, "invalid_tags", tagsErr)
	}
	metadata, metadataErr := parseJSONField[map[string]string](ctx.FormValue("metadata"))
	if metadataErr != nil {
		return corehttp.NewAppError(400, "invalid_metadata", metadataErr)
	}

	entity, createErr := h.service.Create(ctx.Context(), assetsapplication.CreateCommand{
		Name:         name,
		OriginalName: fileHeader.Filename,
		FolderID:     strings.TrimSpace(ctx.FormValue("folderId")),
		MimeType:     mimeType,
		Size:         fileHeader.Size,
		Body:         body,
		Tags:         dereferenceTags(tags),
		Metadata:     dereferenceMetadata(metadata),
	})
	if createErr != nil {
		return h.mapError(createErr)
	}

	return ctx.Status(201).JSON(entity)
}

// findAssets handles paginated list endpoints.
func (h *Handler) findAssets(ctx corehttp.Context) error {
	page, err := parseIntQuery(ctx, "page", 1)
	if err != nil {
		return corehttp.NewAppError(400, "invalid_page", err)
	}
	limit, err := parseIntQuery(ctx, "limit", 10)
	if err != nil {
		return corehttp.NewAppError(400, "invalid_limit", err)
	}
	filters := strings.TrimSpace(ctx.Query("filters"))

	result, listErr := h.service.List(ctx.Context(), assetsapplication.ListQuery{
		Page:    page,
		Limit:   limit,
		Filters: filters,
	})
	if listErr != nil {
		return h.mapError(listErr)
	}

	return ctx.Status(200).JSON(listResponse{
		Data: result.Data,
		Meta: listResponseMeta{Page: result.Page, Total: result.Total, Limit: result.Limit},
	})
}

// findAssetByID handles asset-by-id retrieval endpoints.
func (h *Handler) findAssetByID(ctx corehttp.Context) error {
	entity, err := h.service.Get(ctx.Context(), ctx.Params("id"))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(entity)
}

// updateAsset handles asset update endpoints.
func (h *Handler) updateAsset(ctx corehttp.Context) error {
	var request updateAssetRequest
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	entity, err := h.service.Update(ctx.Context(), ctx.Params("id"), assetsapplication.UpdateCommand{
		Name:     request.Name,
		FolderID: request.FolderID,
		Tags:     request.Tags,
		Metadata: request.Metadata,
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(entity)
}

// deleteAsset handles asset delete endpoints.
func (h *Handler) deleteAsset(ctx corehttp.Context) error {
	if err := h.service.Delete(ctx.Context(), ctx.Params("id")); err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(deleteResponse{Status: "deleted"})
}

// dereferenceTags resolves optional parsed tags.
func dereferenceTags(tags *[]domain.Tag) []domain.Tag {
	if tags == nil {
		return nil
	}

	return *tags
}

// dereferenceMetadata resolves optional parsed metadata.
func dereferenceMetadata(metadata *map[string]string) map[string]string {
	if metadata == nil {
		return nil
	}

	return *metadata
}
