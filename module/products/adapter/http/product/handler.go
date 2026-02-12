package product

import (
	"context"
	"errors"

	corehttp "mannaiah/module/core/http"
	productapplication "mannaiah/module/products/application/product"
	productdomain "mannaiah/module/products/domain/product"
	productport "mannaiah/module/products/port/product"
)

var (
	// ErrNilService is returned when service dependencies are nil.
	ErrNilService = errors.New("products service must not be nil")
)

// Authorizer defines authentication and authorization behavior required by product endpoints.
type Authorizer interface {
	// Require authenticates and authorizes requests using required permissions.
	Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
	// IsUnauthorized reports authentication errors.
	IsUnauthorized(err error) bool
	// IsForbidden reports authorization errors.
	IsForbidden(err error) bool
}

// Handler defines HTTP route handlers for products.
type Handler struct {
	// service defines product use-case dependencies.
	service productapplication.Service
	// authorizer defines optional endpoint auth dependencies.
	authorizer Authorizer
}

// createRequest defines request payload for product creation.
type createRequest struct {
	// SKU defines product stock-keeping values.
	SKU string `json:"sku"`
	// Gallery defines product gallery entries.
	Gallery []productdomain.GalleryItem `json:"gallery"`
	// Datasheets defines datasheet entries.
	Datasheets []productdomain.Datasheet `json:"datasheets"`
	// Variations defines linked variation IDs.
	Variations []string `json:"variations"`
	// Variants defines product variant entries.
	Variants []productdomain.Variant `json:"variants"`
}

// updateRequest defines request payload for product updates.
type updateRequest struct {
	// SKU defines optional SKU updates.
	SKU *string `json:"sku"`
	// Gallery defines optional gallery replacement values.
	Gallery *[]productdomain.GalleryItem `json:"gallery"`
	// Datasheets defines optional datasheet upsert values.
	Datasheets *[]productdomain.Datasheet `json:"datasheets"`
	// Variations defines optional variation replacement values.
	Variations *[]string `json:"variations"`
	// Variants defines optional variant replacement values.
	Variants *[]productdomain.Variant `json:"variants"`
}

// deleteResponse defines delete response payload.
type deleteResponse struct {
	// Status defines delete status values.
	Status string `json:"status"`
}

// listResponse defines list response payload.
type listResponse struct {
	// Data defines listed products.
	Data []productdomain.Product `json:"data"`
}

// NewHandler creates product HTTP handlers.
func NewHandler(service productapplication.Service, authorizers ...Authorizer) (*Handler, error) {
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

// RegisterRoutes registers product CRUD endpoints.
func (h *Handler) RegisterRoutes(router corehttp.Router) {
	router.Post("/products", h.protect("products:manage", h.create))
	router.Get("/products", h.protect("products:read", h.findAll))
	router.Get("/products/:id", h.protect("products:read", h.findOne))
	router.Patch("/products/:id", h.protect("products:manage", h.update))
	router.Delete("/products/:id", h.protect("products:manage", h.remove))
}

// create handles product creation endpoints.
func (h *Handler) create(ctx corehttp.Context) error {
	var request createRequest
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	product, err := h.service.Create(ctx.Context(), productapplication.CreateCommand{
		SKU:        request.SKU,
		Gallery:    request.Gallery,
		Datasheets: request.Datasheets,
		Variations: request.Variations,
		Variants:   request.Variants,
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(201).JSON(product)
}

// findAll handles product listing endpoints.
func (h *Handler) findAll(ctx corehttp.Context) error {
	products, err := h.service.List(ctx.Context())
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(listResponse{Data: products})
}

// findOne handles product-by-id retrieval endpoints.
func (h *Handler) findOne(ctx corehttp.Context) error {
	product, err := h.service.Get(ctx.Context(), ctx.Params("id"))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(product)
}

// update handles product update endpoints.
func (h *Handler) update(ctx corehttp.Context) error {
	var request updateRequest
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	command := productapplication.UpdateCommand{SKU: request.SKU}
	if request.Gallery != nil {
		command.Gallery = *request.Gallery
		command.HasGallery = true
	}
	if request.Datasheets != nil {
		command.Datasheets = *request.Datasheets
		command.HasDatasheets = true
	}
	if request.Variations != nil {
		command.Variations = *request.Variations
		command.HasVariations = true
	}
	if request.Variants != nil {
		command.Variants = *request.Variants
		command.HasVariants = true
	}

	product, err := h.service.Update(ctx.Context(), ctx.Params("id"), command)
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(product)
}

// remove handles product delete endpoints.
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

// mapError maps domain/application/repository/auth errors to HTTP-layer app errors.
func (h *Handler) mapError(err error) error {
	if h != nil && h.authorizer != nil {
		if h.authorizer.IsUnauthorized(err) {
			return corehttp.NewAppError(401, "unauthorized", err)
		}
		if h.authorizer.IsForbidden(err) {
			return corehttp.NewAppError(403, "forbidden", err)
		}
	}
	if errors.Is(err, productport.ErrNotFound) {
		return corehttp.NewAppError(404, "product_not_found", err)
	}
	if errors.Is(err, productapplication.ErrInvalidID) {
		return corehttp.NewAppError(400, "invalid_product_id", err)
	}
	if errors.Is(err, productdomain.ErrSKURequired) ||
		errors.Is(err, productdomain.ErrGalleryAssetIDRequired) ||
		errors.Is(err, productdomain.ErrDatasheetRealmRequired) {
		return corehttp.NewAppError(400, "invalid_product", err)
	}
	if errors.Is(err, productport.ErrDuplicateSKU) {
		return corehttp.NewAppError(409, "product_sku_conflict", err)
	}

	return corehttp.NewAppError(500, "internal_server_error", err)
}
