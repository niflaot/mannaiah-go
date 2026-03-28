package category

import (
	"context"
	"errors"
	"strconv"

	corehttp "mannaiah/module/core/http"
	categoryapplication "mannaiah/module/products/application/category"
	categorydomain "mannaiah/module/products/domain/category"
	categoryport "mannaiah/module/products/port/category"
)

var (
	// ErrNilService is returned when service dependencies are nil.
	ErrNilService = errors.New("category service must not be nil")
)

// Authorizer defines authentication and authorization behavior required by category endpoints.
type Authorizer interface {
	// Require authenticates and authorizes requests using required permissions.
	Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
	// IsUnauthorized reports authentication errors.
	IsUnauthorized(err error) bool
	// IsForbidden reports authorization errors.
	IsForbidden(err error) bool
}

// Handler defines HTTP route handlers for categories.
type Handler struct {
	// service defines category use-case dependencies.
	service categoryapplication.Service
	// authorizer defines optional endpoint auth dependencies.
	authorizer Authorizer
}

// createCategoryRequest defines request payload for category creation.
type createCategoryRequest struct {
	// Slug defines URL-friendly slug values.
	Slug string `json:"slug"`
	// Name defines human-readable name values.
	Name string `json:"name"`
	// Description defines optional description values.
	Description string `json:"description"`
	// ParentID defines optional parent category identifiers.
	ParentID *string `json:"parentId"`
	// IncludeChildren reports whether descendant categories are included.
	IncludeChildren bool `json:"includeChildren"`
	// FilterTags defines tag filter values.
	FilterTags []string `json:"filterTags"`
	// FilterMinPrice defines optional minimum price filter.
	FilterMinPrice *float64 `json:"filterMinPrice"`
	// FilterMaxPrice defines optional maximum price filter.
	FilterMaxPrice *float64 `json:"filterMaxPrice"`
	// FilterCategoryRefs defines category reference filter IDs.
	FilterCategoryRefs []string `json:"filterCategoryRefs"`
	// ProductIDs defines manually pinned product IDs.
	ProductIDs []string `json:"productIds"`
}

// updateCategoryRequest defines request payload for category updates.
type updateCategoryRequest struct {
	// Slug defines optional slug updates.
	Slug *string `json:"slug"`
	// Name defines optional name updates.
	Name *string `json:"name"`
	// Description defines optional description updates.
	Description *string `json:"description"`
	// ParentID defines optional parent category update.
	ParentID *string `json:"parentId"`
	// IncludeChildren defines optional include-children updates.
	IncludeChildren *bool `json:"includeChildren"`
	// FilterTags defines optional tag filter replacement.
	FilterTags *[]string `json:"filterTags"`
	// FilterMinPrice defines optional minimum price filter update.
	FilterMinPrice *float64 `json:"filterMinPrice"`
	// FilterMaxPrice defines optional maximum price filter update.
	FilterMaxPrice *float64 `json:"filterMaxPrice"`
	// FilterCategoryRefs defines optional category ref replacement.
	FilterCategoryRefs *[]string `json:"filterCategoryRefs"`
	// ProductIDs defines optional pinned product ID replacement.
	ProductIDs *[]string `json:"productIds"`
}

// deleteResponse defines delete response payload.
type deleteResponse struct {
	// Status defines delete status values.
	Status string `json:"status"`
}

// listCategoryResponse defines list-category response payload.
type listCategoryResponse struct {
	// Data defines listed categories.
	Data []*categorydomain.Category `json:"data"`
}

// listProductsResponse defines paginated product list response payload.
type listProductsResponse struct {
	// Data defines paginated product results.
	Data   interface{} `json:"data"`
	// Total defines total product count.
	Total  int64       `json:"total"`
	// Page defines current page number.
	Page   int         `json:"page"`
	// PageSize defines current page size.
	PageSize int        `json:"pageSize"`
}

// NewHandler creates category HTTP handlers.
func NewHandler(service categoryapplication.Service, authorizers ...Authorizer) (*Handler, error) {
	if service == nil {
		return nil, ErrNilService
	}

	var authorizer Authorizer
	if len(authorizers) > 0 {
		authorizer = authorizers[0]
	}

	return &Handler{service: service, authorizer: authorizer}, nil
}

// SetAuthorizer configures auth dependencies for protected endpoints.
func (h *Handler) SetAuthorizer(authorizer Authorizer) {
	if h == nil {
		return
	}

	h.authorizer = authorizer
}

// RegisterRoutes registers category CRUD and listing endpoints.
func (h *Handler) RegisterRoutes(router corehttp.Router) {
	router.Post("/categories", h.protect("product:manage", h.create))
	router.Get("/categories", h.protect("product:view", h.tree))
	router.Get("/categories/:id", h.protect("product:view", h.findOne))
	router.Get("/categories/:id/children", h.protect("product:view", h.findChildren))
	router.Get("/categories/:id/products", h.protect("product:view", h.listProducts))
	router.Patch("/categories/:id", h.protect("product:manage", h.update))
	router.Delete("/categories/:id", h.protect("product:manage", h.remove))
}

// create handles category creation endpoints.
func (h *Handler) create(ctx corehttp.Context) error {
	var request createCategoryRequest
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	cat, err := h.service.Create(ctx.Context(), categoryapplication.CreateCommand{
		Slug:               request.Slug,
		Name:               request.Name,
		Description:        request.Description,
		ParentID:           request.ParentID,
		IncludeChildren:    request.IncludeChildren,
		FilterTags:         request.FilterTags,
		FilterMinPrice:     request.FilterMinPrice,
		FilterMaxPrice:     request.FilterMaxPrice,
		FilterCategoryRefs: request.FilterCategoryRefs,
		ProductIDs:         request.ProductIDs,
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(201).JSON(cat)
}

// tree handles category tree listing endpoints.
func (h *Handler) tree(ctx corehttp.Context) error {
	cats, err := h.service.Tree(ctx.Context())
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(listCategoryResponse{Data: cats})
}

// findOne handles category-by-id retrieval endpoints.
func (h *Handler) findOne(ctx corehttp.Context) error {
	cat, err := h.service.Get(ctx.Context(), ctx.Params("id"))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(cat)
}

// findChildren handles category-children listing endpoints.
func (h *Handler) findChildren(ctx corehttp.Context) error {
	cats, err := h.service.Children(ctx.Context(), ctx.Params("id"))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(listCategoryResponse{Data: cats})
}

// listProducts handles category-products listing endpoints.
func (h *Handler) listProducts(ctx corehttp.Context) error {
	page := parseIntQuery(ctx.Query("page"), 1)
	pageSize := parseIntQuery(ctx.Query("pageSize"), 20)

	result, err := h.service.ListProducts(ctx.Context(), categoryapplication.ListProductsQuery{
		CategoryID: ctx.Params("id"),
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(listProductsResponse{
		Data:     result.Items,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	})
}

// update handles category update endpoints.
func (h *Handler) update(ctx corehttp.Context) error {
	var raw map[string]any
	if err := ctx.BodyParser(&raw); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	var request updateCategoryRequest
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	command := categoryapplication.UpdateCommand{
		Slug:            request.Slug,
		Name:            request.Name,
		Description:     request.Description,
		IncludeChildren: request.IncludeChildren,
	}
	if _, ok := raw["parentId"]; ok {
		command.HasParentID = true
		command.ParentID = request.ParentID
	}
	if request.FilterTags != nil {
		command.FilterTags = *request.FilterTags
		command.HasFilterTags = true
	}
	if request.FilterMinPrice != nil || request.FilterMaxPrice != nil {
		command.FilterMinPrice = request.FilterMinPrice
		command.FilterMaxPrice = request.FilterMaxPrice
		command.HasFilterPriceRange = true
	}
	if request.FilterCategoryRefs != nil {
		command.FilterCategoryRefs = *request.FilterCategoryRefs
		command.HasFilterCategoryRefs = true
	}
	if request.ProductIDs != nil {
		command.ProductIDs = *request.ProductIDs
		command.HasProductIDs = true
	}

	cat, err := h.service.Update(ctx.Context(), ctx.Params("id"), command)
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(cat)
}

// remove handles category delete endpoints.
func (h *Handler) remove(ctx corehttp.Context) error {
	if err := h.service.Delete(ctx.Context(), ctx.Params("id")); err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(deleteResponse{Status: "deleted"})
}

// protect wraps handlers with optional auth checks.
func (h *Handler) protect(permission string, next corehttp.Handler) corehttp.Handler {
	if h == nil || h.authorizer == nil {
		return next
	}

	return func(ctx corehttp.Context) error {
		err := h.authorizer.Require(ctx.Context(), ctx.GetHeader("Authorization"), permission)
		if err != nil {
			return h.mapError(err)
		}

		return next(ctx)
	}
}

// mapError maps domain/application/port errors to HTTP-layer app errors.
func (h *Handler) mapError(err error) error {
	if h != nil && h.authorizer != nil {
		if h.authorizer.IsUnauthorized(err) {
			return corehttp.NewAppError(401, "unauthorized", err)
		}
		if h.authorizer.IsForbidden(err) {
			return corehttp.NewAppError(403, "forbidden", err)
		}
	}
	if errors.Is(err, categoryapplication.ErrNotFound) || errors.Is(err, categoryport.ErrNotFound) {
		return corehttp.NewAppError(404, "category_not_found", err)
	}
	if errors.Is(err, categoryapplication.ErrInvalidID) {
		return corehttp.NewAppError(400, "invalid_category_id", err)
	}
	if errors.Is(err, categoryapplication.ErrDuplicateSlug) || errors.Is(err, categoryport.ErrDuplicateSlug) {
		return corehttp.NewAppError(409, "category_slug_conflict", err)
	}
	if errors.Is(err, categoryapplication.ErrHasChildren) || errors.Is(err, categoryport.ErrHasChildren) {
		return corehttp.NewAppError(409, "category_has_children", err)
	}
	if errors.Is(err, categorydomain.ErrSlugRequired) || errors.Is(err, categorydomain.ErrNameRequired) {
		return corehttp.NewAppError(400, "invalid_category", err)
	}
	if errors.Is(err, categorydomain.ErrCircularParent) {
		return corehttp.NewAppError(400, "circular_category_parent", err)
	}

	return corehttp.NewAppError(500, "internal_server_error", err)
}

// parseIntQuery parses integer query parameters with a fallback default value.
func parseIntQuery(raw string, defaultValue int) int {
	if raw == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return defaultValue
	}

	return value
}
