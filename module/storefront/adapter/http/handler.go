package http

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	corehttp "mannaiah/module/core/http"
	coresearch "mannaiah/module/core/search"
	pageservice "mannaiah/module/storefront/application/page/service"
	renderableservice "mannaiah/module/storefront/application/renderable/service"
	"mannaiah/module/storefront/domain"
	"mannaiah/module/storefront/port"
)

var (
	// ErrNilRenderableService is returned when renderable service dependencies are nil.
	ErrNilRenderableService = errors.New("renderable service must not be nil")
	// ErrNilPageService is returned when static-page service dependencies are nil.
	ErrNilPageService = errors.New("static page service must not be nil")
)

// Authorizer defines authentication and authorization behavior required by storefront endpoints.
type Authorizer interface {
	// Require authenticates and authorizes requests using required permissions.
	Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
	// IsUnauthorized reports authentication errors.
	IsUnauthorized(err error) bool
	// IsForbidden reports authorization errors.
	IsForbidden(err error) bool
}

// RenderableService defines renderable use-case behavior required by HTTP handlers.
type RenderableService interface {
	// Create persists a new draft renderable.
	Create(ctx context.Context, cmd renderableservice.CreateCommand) (*domain.Renderable, error)
	// GetByID loads one renderable by identifier.
	GetByID(ctx context.Context, id string) (*domain.Renderable, error)
	// Update applies draft changes.
	Update(ctx context.Context, cmd renderableservice.UpdateCommand) (*domain.Renderable, error)
	// Delete removes one renderable.
	Delete(ctx context.Context, id string) error
	// List returns paginated renderables.
	List(ctx context.Context, query port.RenderableListQuery) ([]domain.Renderable, int64, error)
	// Publish creates a new published renderable snapshot.
	Publish(ctx context.Context, id string) (*domain.RenderableVersion, error)
	// ListVersions returns paginated published versions.
	ListVersions(ctx context.Context, id string, page int, pageSize int) ([]domain.RenderableVersion, int64, error)
	// GetVersionByID loads one published version.
	GetVersionByID(ctx context.Context, id string, versionID string) (*domain.RenderableVersion, error)
	// Rollback creates a fresh published snapshot from one historical version.
	Rollback(ctx context.Context, id string, versionID string) (*domain.RenderableVersion, error)
}

// PageService defines static-page use-case behavior required by HTTP handlers.
type PageService interface {
	// Create persists a new static page.
	Create(ctx context.Context, cmd pageservice.CreateCommand) (*domain.StaticPage, error)
	// GetByID loads one static page.
	GetByID(ctx context.Context, id string) (*domain.StaticPage, error)
	// Update applies page mutations.
	Update(ctx context.Context, cmd pageservice.UpdateCommand) (*domain.StaticPage, error)
	// Delete removes one static page.
	Delete(ctx context.Context, id string) error
	// List returns paginated static pages.
	List(ctx context.Context, query port.StaticPageListQuery) ([]domain.StaticPage, int64, error)
}

// Handler defines HTTP route handlers for storefront endpoints.
type Handler struct {
	// renderables defines renderable use-case dependencies.
	renderables RenderableService
	// pages defines static-page use-case dependencies.
	pages PageService
	// authorizer defines optional endpoint auth dependencies.
	authorizer Authorizer
}

// NewHandler creates storefront HTTP handlers.
func NewHandler(renderables RenderableService, pages PageService, authorizers ...Authorizer) (*Handler, error) {
	if renderables == nil {
		return nil, ErrNilRenderableService
	}
	if pages == nil {
		return nil, ErrNilPageService
	}

	var authorizer Authorizer
	if len(authorizers) > 0 {
		authorizer = authorizers[0]
	}

	return &Handler{renderables: renderables, pages: pages, authorizer: authorizer}, nil
}

// SetAuthorizer configures endpoint authentication dependencies.
func (h *Handler) SetAuthorizer(authorizer Authorizer) {
	if h == nil {
		return
	}

	h.authorizer = authorizer
}

// RegisterRoutes registers storefront management endpoints.
func (h *Handler) RegisterRoutes(router corehttp.Router) {
	router.Post("/storefront/renderable", h.protect(h.createRenderable))
	router.Get("/storefront/renderable", h.protect(h.listRenderables))
	router.Get("/storefront/renderable/:id", h.protect(h.getRenderable))
	router.Patch("/storefront/renderable/:id", h.protect(h.updateRenderable))
	router.Delete("/storefront/renderable/:id", h.protect(h.deleteRenderable))
	router.Post("/storefront/renderable/:id/publish", h.protect(h.publishRenderable))
	router.Get("/storefront/renderable/:id/versions", h.protect(h.listRenderableVersions))
	router.Get("/storefront/renderable/:id/versions/:versionId", h.protect(h.getRenderableVersion))
	router.Post("/storefront/renderable/:id/versions/:versionId/rollback", h.protect(h.rollbackRenderableVersion))

	router.Post("/storefront/page", h.protect(h.createPage))
	router.Get("/storefront/page", h.protect(h.listPages))
	router.Get("/storefront/page/:id", h.protect(h.getPage))
	router.Patch("/storefront/page/:id", h.protect(h.updatePage))
	router.Delete("/storefront/page/:id", h.protect(h.deletePage))
}

// renderableRequest defines renderable create/update payload values.
type renderableRequest struct {
	// Kind defines the renderable kind.
	Kind string `json:"kind"`
	// Metadata defines renderable metadata JSON.
	Metadata json.RawMessage `json:"metadata"`
	// Content defines renderable editor JSON.
	Content json.RawMessage `json:"content"`
}

// staticPageRequest defines static-page create/update payload values.
type staticPageRequest struct {
	// RenderableID defines the bound renderable identifier.
	RenderableID string `json:"renderableId"`
	// Title defines page title values.
	Title string `json:"title"`
	// URL defines storefront URL values.
	URL string `json:"url"`
	// SEOTags defines frontend-provided SEO JSON.
	SEOTags json.RawMessage `json:"seoTags"`
}

// createRenderable handles POST /storefront/renderable.
func (h *Handler) createRenderable(ctx corehttp.Context) error {
	var request renderableRequest
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	renderable, err := h.renderables.Create(ctx.Context(), renderableservice.CreateCommand{
		Kind:     strings.TrimSpace(request.Kind),
		Metadata: request.Metadata,
		Content:  request.Content,
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(201).JSON(renderable)
}

// listRenderables handles GET /storefront/renderable.
func (h *Handler) listRenderables(ctx corehttp.Context) error {
	query := port.RenderableListQuery{
		Kind:     strings.TrimSpace(ctx.Query("kind")),
		Page:     parsePositiveInt(ctx, "page", 1),
		PageSize: parsePositiveInt(ctx, "pageSize", coresearch.DefaultPageSize),
	}
	if draftRaw := strings.TrimSpace(ctx.Query("draft")); draftRaw != "" {
		if parsed, err := strconv.ParseBool(draftRaw); err == nil {
			query.Draft = &parsed
		}
	}

	rows, total, err := h.renderables.List(ctx.Context(), query)
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(coresearch.NewResult(rows, total, query.Page, query.PageSize))
}

// getRenderable handles GET /storefront/renderable/:id.
func (h *Handler) getRenderable(ctx corehttp.Context) error {
	renderable, err := h.renderables.GetByID(ctx.Context(), ctx.Params("id"))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(renderable)
}

// updateRenderable handles PATCH /storefront/renderable/:id.
func (h *Handler) updateRenderable(ctx corehttp.Context) error {
	var request renderableRequest
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	renderable, err := h.renderables.Update(ctx.Context(), renderableservice.UpdateCommand{
		ID:       ctx.Params("id"),
		Metadata: request.Metadata,
		Content:  request.Content,
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(renderable)
}

// deleteRenderable handles DELETE /storefront/renderable/:id.
func (h *Handler) deleteRenderable(ctx corehttp.Context) error {
	if err := h.renderables.Delete(ctx.Context(), ctx.Params("id")); err != nil {
		return h.mapError(err)
	}

	return ctx.SendStatus(204)
}

// publishRenderable handles POST /storefront/renderable/:id/publish.
func (h *Handler) publishRenderable(ctx corehttp.Context) error {
	version, err := h.renderables.Publish(ctx.Context(), ctx.Params("id"))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(201).JSON(version)
}

// listRenderableVersions handles GET /storefront/renderable/:id/versions.
func (h *Handler) listRenderableVersions(ctx corehttp.Context) error {
	page := parsePositiveInt(ctx, "page", 1)
	pageSize := parsePositiveInt(ctx, "pageSize", coresearch.DefaultPageSize)
	rows, total, err := h.renderables.ListVersions(ctx.Context(), ctx.Params("id"), page, pageSize)
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(coresearch.NewResult(rows, total, page, pageSize))
}

// getRenderableVersion handles GET /storefront/renderable/:id/versions/:versionId.
func (h *Handler) getRenderableVersion(ctx corehttp.Context) error {
	version, err := h.renderables.GetVersionByID(ctx.Context(), ctx.Params("id"), ctx.Params("versionId"))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(version)
}

// rollbackRenderableVersion handles POST /storefront/renderable/:id/versions/:versionId/rollback.
func (h *Handler) rollbackRenderableVersion(ctx corehttp.Context) error {
	version, err := h.renderables.Rollback(ctx.Context(), ctx.Params("id"), ctx.Params("versionId"))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(201).JSON(version)
}

// createPage handles POST /storefront/page.
func (h *Handler) createPage(ctx corehttp.Context) error {
	var request staticPageRequest
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	page, err := h.pages.Create(ctx.Context(), pageservice.CreateCommand{
		RenderableID: strings.TrimSpace(request.RenderableID),
		Title:        strings.TrimSpace(request.Title),
		URL:          strings.TrimSpace(request.URL),
		SEOTags:      request.SEOTags,
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(201).JSON(page)
}

// listPages handles GET /storefront/page.
func (h *Handler) listPages(ctx corehttp.Context) error {
	query := port.StaticPageListQuery{
		Term:         strings.TrimSpace(ctx.Query("term")),
		RenderableID: strings.TrimSpace(ctx.Query("renderableId")),
		Page:         parsePositiveInt(ctx, "page", 1),
		PageSize:     parsePositiveInt(ctx, "pageSize", coresearch.DefaultPageSize),
	}

	rows, total, err := h.pages.List(ctx.Context(), query)
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(coresearch.NewResult(rows, total, query.Page, query.PageSize))
}

// getPage handles GET /storefront/page/:id.
func (h *Handler) getPage(ctx corehttp.Context) error {
	page, err := h.pages.GetByID(ctx.Context(), ctx.Params("id"))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(page)
}

// updatePage handles PATCH /storefront/page/:id.
func (h *Handler) updatePage(ctx corehttp.Context) error {
	var request staticPageRequest
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	page, err := h.pages.Update(ctx.Context(), pageservice.UpdateCommand{
		ID:           ctx.Params("id"),
		RenderableID: strings.TrimSpace(request.RenderableID),
		Title:        strings.TrimSpace(request.Title),
		URL:          strings.TrimSpace(request.URL),
		SEOTags:      request.SEOTags,
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(page)
}

// deletePage handles DELETE /storefront/page/:id.
func (h *Handler) deletePage(ctx corehttp.Context) error {
	if err := h.pages.Delete(ctx.Context(), ctx.Params("id")); err != nil {
		return h.mapError(err)
	}

	return ctx.SendStatus(204)
}

// protect applies storefront:manage permission checks to protected endpoints.
func (h *Handler) protect(next corehttp.Handler) corehttp.Handler {
	return func(ctx corehttp.Context) error {
		if h.authorizer == nil {
			return next(ctx)
		}

		err := h.authorizer.Require(ctx.Context(), ctx.GetHeader("Authorization"), "storefront:manage")
		if err == nil {
			return next(ctx)
		}
		if h.authorizer.IsUnauthorized(err) {
			return corehttp.NewAppError(401, "unauthorized", err)
		}
		if h.authorizer.IsForbidden(err) {
			return corehttp.NewAppError(403, "forbidden", err)
		}

		return corehttp.NewAppError(500, "authorization_failed", err)
	}
}

// mapError maps domain and service errors into API-friendly responses.
func (h *Handler) mapError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, renderableservice.ErrRenderableNotFound), errors.Is(err, pageservice.ErrStaticPageNotFound), errors.Is(err, renderableservice.ErrRenderableVersionNotFound), errors.Is(err, pageservice.ErrStaticPageRenderableNotFound):
		return corehttp.NewAppError(404, "not_found", err)
	case errors.Is(err, pageservice.ErrStaticPageURLConflict), errors.Is(err, pageservice.ErrStaticPageRenderableConflict):
		return corehttp.NewAppError(409, "conflict", err)
	case errors.Is(err, pageservice.ErrStaticPageRenderableKindMismatch), errors.Is(err, domain.ErrRenderableKindRequired), errors.Is(err, domain.ErrRenderableMetadataInvalid), errors.Is(err, domain.ErrRenderableContentInvalid), errors.Is(err, domain.ErrStaticPageRenderableIDRequired), errors.Is(err, domain.ErrStaticPageTitleRequired), errors.Is(err, domain.ErrStaticPageURLRequired), errors.Is(err, domain.ErrStaticPageSEOTagsInvalid):
		return corehttp.NewAppError(400, "validation_error", err)
	default:
		return corehttp.NewAppError(500, "internal_server_error", err)
	}
}

// parsePositiveInt resolves positive integer query values with a fallback.
func parsePositiveInt(ctx corehttp.Context, key string, fallback int) int {
	value := strings.TrimSpace(ctx.Query(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}
