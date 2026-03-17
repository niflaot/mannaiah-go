package http

import (
	"context"
	"strings"

	"mannaiah/module/analytics/domain"
	rfmapp "mannaiah/module/analytics/application/rfm"
	corehttp "mannaiah/module/core/http"
)

// RFMService defines RFM use-case behavior required by HTTP handlers.
type RFMService interface {
	// CreateGroup persists a new RFM group definition.
	CreateGroup(ctx context.Context, group domain.RFMGroup) (*domain.RFMGroup, error)
	// GetGroup retrieves one RFM group by identifier.
	GetGroup(ctx context.Context, id string) (*domain.RFMGroup, error)
	// ListGroups retrieves all RFM groups.
	ListGroups(ctx context.Context) ([]domain.RFMGroup, error)
	// UpdateGroup persists RFM group updates.
	UpdateGroup(ctx context.Context, group domain.RFMGroup) (*domain.RFMGroup, error)
	// DeleteGroup removes one RFM group by identifier.
	DeleteGroup(ctx context.Context, id string) error
	// GetBands retrieves all RFM band threshold configurations.
	GetBands(ctx context.Context) ([]domain.RFMBandConfig, error)
	// UpdateBand persists a single RFM band configuration.
	UpdateBand(ctx context.Context, cfg domain.RFMBandConfig) error
	// ScoreContact computes RFM scores for one contact.
	ScoreContact(ctx context.Context, contactID string) (*domain.RFMScore, error)
	// ScoreBatch computes RFM scores for up to 1000 contacts.
	ScoreBatch(ctx context.Context, contactIDs []string) ([]domain.RFMScore, error)
	// RefreshMV truncates and repopulates the rfm_scores_mv ClickHouse table.
	RefreshMV(ctx context.Context) error
}

// RFMHandler defines HTTP route handlers for RFM endpoints.
type RFMHandler struct {
	// service defines RFM use-case dependencies.
	service RFMService
	// authorizer defines optional auth dependencies.
	authorizer Authorizer
}

// NewRFMHandler creates RFM HTTP handlers.
func NewRFMHandler(service RFMService, authorizers ...Authorizer) *RFMHandler {
	var authorizer Authorizer
	if len(authorizers) > 0 {
		authorizer = authorizers[0]
	}

	return &RFMHandler{service: service, authorizer: authorizer}
}

// SetAuthorizer configures endpoint authentication and permission dependencies.
func (h *RFMHandler) SetAuthorizer(authorizer Authorizer) {
	if h == nil {
		return
	}
	h.authorizer = authorizer
}

// RegisterRoutes registers RFM routes on the provided router.
func (h *RFMHandler) RegisterRoutes(router corehttp.Router) {
	router.Get("/analytics/rfm/bands", h.protect("marketing:manage", h.getBands))
	router.Put("/analytics/rfm/bands/:dimension", h.protect("marketing:manage", h.updateBand))
	router.Post("/analytics/rfm/groups", h.protect("marketing:manage", h.createGroup))
	router.Get("/analytics/rfm/groups", h.protect("marketing:manage", h.listGroups))
	router.Get("/analytics/rfm/groups/:id", h.protect("marketing:manage", h.getGroup))
	router.Put("/analytics/rfm/groups/:id", h.protect("marketing:manage", h.updateGroup))
	router.Delete("/analytics/rfm/groups/:id", h.protect("marketing:manage", h.deleteGroup))
	router.Get("/analytics/rfm/contacts/:contactId/score", h.protect("marketing:manage", h.scoreContact))
	router.Post("/analytics/rfm/contacts/score-batch", h.protect("marketing:manage", h.scoreBatch))
	router.Post("/analytics/rfm/refresh", h.protect("marketing:manage", h.refreshMV))
}

// getBands handles GET /analytics/rfm/bands.
func (h *RFMHandler) getBands(ctx corehttp.Context) error {
	bands, err := h.service.GetBands(ctx.Context())
	if err != nil {
		return h.mapRFMError(err)
	}

	return ctx.Status(200).JSON(bands)
}

// rfmBandUpdateRequest defines the update payload for one RFM band.
type rfmBandUpdateRequest struct {
	Band5Min float64 `json:"band5Min"`
	Band4Min float64 `json:"band4Min"`
	Band3Min float64 `json:"band3Min"`
	Band2Min float64 `json:"band2Min"`
	Ascending bool   `json:"ascending"`
}

// updateBand handles PUT /analytics/rfm/bands/:dimension.
func (h *RFMHandler) updateBand(ctx corehttp.Context) error {
	dimension := strings.TrimSpace(ctx.Params("dimension"))
	var req rfmBandUpdateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return corehttp.NewAppError(400, "invalid_request_body", err)
	}

	cfg := domain.RFMBandConfig{
		Dimension: domain.RFMDimension(dimension),
		Ascending: req.Ascending,
		Band5Min:  req.Band5Min,
		Band4Min:  req.Band4Min,
		Band3Min:  req.Band3Min,
		Band2Min:  req.Band2Min,
	}
	if err := h.service.UpdateBand(ctx.Context(), cfg); err != nil {
		return h.mapRFMError(err)
	}

	return ctx.Status(200).JSON(cfg)
}

// rfmGroupRequest defines create/update payload for an RFM group.
type rfmGroupRequest struct {
	Name        string                     `json:"name"`
	Slug        string                     `json:"slug"`
	Description string                     `json:"description"`
	Conditions  rfmGroupConditionsRequest  `json:"conditions"`
}

// rfmGroupConditionsRequest defines condition payload values.
type rfmGroupConditionsRequest struct {
	RMin *int `json:"rMin"`
	RMax *int `json:"rMax"`
	FMin *int `json:"fMin"`
	FMax *int `json:"fMax"`
	MMin *int `json:"mMin"`
	MMax *int `json:"mMax"`
}

// createGroup handles POST /analytics/rfm/groups.
func (h *RFMHandler) createGroup(ctx corehttp.Context) error {
	var req rfmGroupRequest
	if err := ctx.BodyParser(&req); err != nil {
		return corehttp.NewAppError(400, "invalid_request_body", err)
	}

	group := domain.RFMGroup{
		Name:        strings.TrimSpace(req.Name),
		Slug:        strings.TrimSpace(req.Slug),
		Description: strings.TrimSpace(req.Description),
		Conditions: domain.RFMGroupConditions{
			RMin: req.Conditions.RMin, RMax: req.Conditions.RMax,
			FMin: req.Conditions.FMin, FMax: req.Conditions.FMax,
			MMin: req.Conditions.MMin, MMax: req.Conditions.MMax,
		},
	}
	created, err := h.service.CreateGroup(ctx.Context(), group)
	if err != nil {
		return h.mapRFMError(err)
	}

	return ctx.Status(201).JSON(created)
}

// listGroups handles GET /analytics/rfm/groups.
func (h *RFMHandler) listGroups(ctx corehttp.Context) error {
	groups, err := h.service.ListGroups(ctx.Context())
	if err != nil {
		return h.mapRFMError(err)
	}

	return ctx.Status(200).JSON(groups)
}

// getGroup handles GET /analytics/rfm/groups/:id.
func (h *RFMHandler) getGroup(ctx corehttp.Context) error {
	id := strings.TrimSpace(ctx.Params("id"))
	group, err := h.service.GetGroup(ctx.Context(), id)
	if err != nil {
		return h.mapRFMError(err)
	}

	return ctx.Status(200).JSON(group)
}

// updateGroup handles PUT /analytics/rfm/groups/:id.
func (h *RFMHandler) updateGroup(ctx corehttp.Context) error {
	id := strings.TrimSpace(ctx.Params("id"))
	var req rfmGroupRequest
	if err := ctx.BodyParser(&req); err != nil {
		return corehttp.NewAppError(400, "invalid_request_body", err)
	}

	group := domain.RFMGroup{
		ID:          id,
		Name:        strings.TrimSpace(req.Name),
		Slug:        strings.TrimSpace(req.Slug),
		Description: strings.TrimSpace(req.Description),
		Conditions: domain.RFMGroupConditions{
			RMin: req.Conditions.RMin, RMax: req.Conditions.RMax,
			FMin: req.Conditions.FMin, FMax: req.Conditions.FMax,
			MMin: req.Conditions.MMin, MMax: req.Conditions.MMax,
		},
	}
	updated, err := h.service.UpdateGroup(ctx.Context(), group)
	if err != nil {
		return h.mapRFMError(err)
	}

	return ctx.Status(200).JSON(updated)
}

// deleteGroup handles DELETE /analytics/rfm/groups/:id.
func (h *RFMHandler) deleteGroup(ctx corehttp.Context) error {
	id := strings.TrimSpace(ctx.Params("id"))
	if err := h.service.DeleteGroup(ctx.Context(), id); err != nil {
		return h.mapRFMError(err)
	}

	return ctx.Status(204).JSON(nil)
}

// scoreContact handles GET /analytics/rfm/contacts/:contactId/score.
func (h *RFMHandler) scoreContact(ctx corehttp.Context) error {
	contactID := strings.TrimSpace(ctx.Params("contactId"))
	score, err := h.service.ScoreContact(ctx.Context(), contactID)
	if err != nil {
		return h.mapRFMError(err)
	}

	return ctx.Status(200).JSON(score)
}

// scoreBatchRequest defines the batch scoring request payload.
type scoreBatchRequest struct {
	ContactIDs []string `json:"contactIds"`
}

// scoreBatch handles POST /analytics/rfm/contacts/score-batch.
func (h *RFMHandler) scoreBatch(ctx corehttp.Context) error {
	var req scoreBatchRequest
	if err := ctx.BodyParser(&req); err != nil {
		return corehttp.NewAppError(400, "invalid_request_body", err)
	}

	scores, err := h.service.ScoreBatch(ctx.Context(), req.ContactIDs)
	if err != nil {
		return h.mapRFMError(err)
	}

	return ctx.Status(200).JSON(scores)
}

// refreshMV handles POST /analytics/rfm/refresh.
func (h *RFMHandler) refreshMV(ctx corehttp.Context) error {
	if err := h.service.RefreshMV(ctx.Context()); err != nil {
		return h.mapRFMError(err)
	}

	return ctx.Status(200).JSON(map[string]string{"status": "ok"})
}

// protect wraps RFM endpoint handlers with optional authentication.
func (h *RFMHandler) protect(permission string, next corehttp.Handler) corehttp.Handler {
	if h == nil || h.authorizer == nil {
		return next
	}

	return func(ctx corehttp.Context) error {
		if err := h.authorizer.Require(ctx.Context(), ctx.GetHeader("Authorization"), permission); err != nil {
			return mapAuthError(h.authorizer, err)
		}

		return next(ctx)
	}
}

// mapRFMError maps RFM service errors to HTTP-layer app errors.
func (h *RFMHandler) mapRFMError(err error) error {
	if h != nil && h.authorizer != nil {
		if err2 := mapAuthError(h.authorizer, err); err2 != err {
			return err2
		}
	}

	return mapServiceError(err)
}

// mapAuthError maps authorization errors for a given authorizer.
func mapAuthError(auth Authorizer, err error) error {
	if auth.IsUnauthorized(err) {
		return corehttp.NewAppError(401, "unauthorized", err)
	}
	if auth.IsForbidden(err) {
		return corehttp.NewAppError(403, "forbidden", err)
	}

	return err
}

// mapServiceError maps common service errors to HTTP errors.
func mapServiceError(err error) error {
	_ = rfmapp.ErrNilRFMStore
	if err == rfmapp.ErrGroupNotFound {
		return corehttp.NewAppError(404, "not_found", err)
	}

	return corehttp.NewAppError(500, "internal_server_error", err)
}
