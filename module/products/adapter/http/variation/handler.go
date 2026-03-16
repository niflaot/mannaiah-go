package variation

import (
	"context"
	"errors"

	corehttp "mannaiah/module/core/http"
	variationapplication "mannaiah/module/products/application/variation"
	variationdomain "mannaiah/module/products/domain/variation"
	variationport "mannaiah/module/products/port/variation"
)

var (
	// ErrNilService is returned when service dependencies are nil.
	ErrNilService = errors.New("variations service must not be nil")
)

// Authorizer defines authentication and authorization behavior required by variation endpoints.
type Authorizer interface {
	// Require authenticates and authorizes requests using required permissions.
	Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
	// IsUnauthorized reports authentication errors.
	IsUnauthorized(err error) bool
	// IsForbidden reports authorization errors.
	IsForbidden(err error) bool
}

// Handler defines HTTP route handlers for variations.
type Handler struct {
	// service defines variation use-case dependencies.
	service variationapplication.Service
	// authorizer defines optional endpoint auth dependencies.
	authorizer Authorizer
}

// createRequest defines request payload for variation creation.
type createRequest struct {
	// Name defines variation labels.
	Name string `json:"name"`
	// Definition defines variation type.
	Definition variationdomain.Definition `json:"definition"`
	// Value defines machine-readable variation values.
	Value string `json:"value"`
}

// updateRequest defines request payload for variation updates.
type updateRequest struct {
	// Name defines optional variation label updates.
	Name *string `json:"name"`
	// Definition defines optional variation definition updates.
	Definition *variationdomain.Definition `json:"definition"`
	// Value defines optional variation value updates.
	Value *string `json:"value"`
}

// deleteResponse defines delete endpoint response payload.
type deleteResponse struct {
	// Status defines delete status values.
	Status string `json:"status"`
}

// listResponse defines list endpoint response payload.
type listResponse struct {
	// Data defines listed variation rows.
	Data []variationdomain.Variation `json:"data"`
}

// NewHandler creates variation HTTP handlers.
func NewHandler(service variationapplication.Service, authorizers ...Authorizer) (*Handler, error) {
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

// RegisterRoutes registers variation CRUD endpoints.
func (h *Handler) RegisterRoutes(router corehttp.Router) {
	router.Post("/variations", h.protect("products:create", h.create))
	router.Get("/variations", h.protect("products:read", h.findAll))
	router.Get("/variations/:id", h.protect("products:read", h.findOne))
	router.Patch("/variations/:id", h.protect("products:update", h.update))
	router.Delete("/variations/:id", h.protect("products:delete", h.remove))
}

// create handles variation creation endpoints.
func (h *Handler) create(ctx corehttp.Context) error {
	var request createRequest
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	variation, err := h.service.Create(ctx.Context(), variationapplication.CreateCommand{
		Name:       request.Name,
		Definition: request.Definition,
		Value:      request.Value,
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(201).JSON(variation)
}

// findAll handles variation listing endpoints.
func (h *Handler) findAll(ctx corehttp.Context) error {
	variations, err := h.service.List(ctx.Context())
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(listResponse{Data: variations})
}

// findOne handles variation-by-id retrieval endpoints.
func (h *Handler) findOne(ctx corehttp.Context) error {
	variation, err := h.service.Get(ctx.Context(), ctx.Params("id"))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(variation)
}

// update handles variation update endpoints.
func (h *Handler) update(ctx corehttp.Context) error {
	var request updateRequest
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	variation, err := h.service.Update(ctx.Context(), ctx.Params("id"), variationapplication.UpdateCommand{
		Name:       request.Name,
		Definition: request.Definition,
		Value:      request.Value,
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(variation)
}

// remove handles variation delete endpoints.
func (h *Handler) remove(ctx corehttp.Context) error {
	if err := h.service.Delete(ctx.Context(), ctx.Params("id")); err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(deleteResponse{Status: "deleted"})
}

// protect wraps endpoint handlers with optional auth checks.
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
	if errors.Is(err, variationport.ErrNotFound) {
		return corehttp.NewAppError(404, "variation_not_found", err)
	}
	if errors.Is(err, variationapplication.ErrInvalidID) {
		return corehttp.NewAppError(400, "invalid_variation_id", err)
	}
	if isDomainValidationError(err) {
		return corehttp.NewAppError(400, "invalid_variation", err)
	}

	return corehttp.NewAppError(500, "internal_server_error", err)
}

// isDomainValidationError reports variation domain invariant errors.
func isDomainValidationError(err error) bool {
	return errors.Is(err, variationdomain.ErrNameRequired) ||
		errors.Is(err, variationdomain.ErrValueRequired) ||
		errors.Is(err, variationdomain.ErrDefinitionRequired) ||
		errors.Is(err, variationdomain.ErrInvalidDefinition)
}
