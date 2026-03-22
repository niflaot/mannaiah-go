package http

import (
	"context"
	"errors"
	"strconv"
	"strings"

	corehttp "mannaiah/module/core/http"
	"mannaiah/module/campaign/application"
	"mannaiah/module/campaign/domain"
)

var (
	// ErrNilService is returned when nil service dependencies are provided.
	ErrNilService = errors.New("campaign service must not be nil")
)

// Authorizer defines authentication and authorization behavior required by campaign endpoints.
type Authorizer interface {
	// Require authenticates and authorizes requests using required permissions.
	Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
	// IsUnauthorized reports authentication errors.
	IsUnauthorized(err error) bool
	// IsForbidden reports authorization errors.
	IsForbidden(err error) bool
}

// Service defines campaign use-case behavior required by HTTP handlers.
type Service interface {
	// Create persists campaign rows.
	Create(ctx context.Context, command application.CreateCommand) (*domain.Campaign, error)
	// Get retrieves one campaign by id.
	Get(ctx context.Context, id string) (*domain.Campaign, error)
	// List retrieves paged campaign rows.
	List(ctx context.Context, page int, limit int) (*application.ListResult, error)
	// Update persists campaign row updates.
	Update(ctx context.Context, id string, command application.UpdateCommand) (*domain.Campaign, error)
	// Delete removes one campaign by id.
	Delete(ctx context.Context, id string) error
	// Send starts asynchronous campaign fan-out and returns accepted campaign states.
	Send(ctx context.Context, id string) (*domain.Campaign, error)
	// ListDeliveries retrieves paged delivery rows for one campaign.
	ListDeliveries(ctx context.Context, id string, page int, limit int) (*application.DeliveryListResult, error)
	// TestSend renders and delivers the campaign to a single override email for preview purposes.
	TestSend(ctx context.Context, campaignID string, command application.TestSendCommand) (*application.TestSendResult, error)
}

// Handler defines HTTP route handlers for campaign endpoints.
type Handler struct {
	// service defines campaign use-case dependencies.
	service Service
	// authorizer defines optional auth dependencies.
	authorizer Authorizer
}

// productBlockRequest defines a product recommendation block in create/update requests.
type productBlockRequest struct {
	ID                  string   `json:"id"`
	BaseTag             string   `json:"baseTag"`
	BaseTags            []string `json:"baseTags"`
	BaseTagMode         string   `json:"baseTagMode"`
	UseAffinity         bool     `json:"useAffinity"`
	AffinityMinScorePct float64  `json:"affinityMinScorePct"`
	CategoryID          string   `json:"categoryId"`
	Realm               string   `json:"realm"`
	Limit               int      `json:"limit"`
	PinnedProductIDs    []string `json:"pinnedProductIds"`
	ExcludeProductIDs   []string `json:"excludeProductIds"`
	FilterVariationIDs  []string `json:"filterVariationIds"`
	PreferVariationIDs  []string `json:"preferVariationIds"`
}

// createRequest defines create request payload values.
type createRequest struct {
	// Name defines campaign names.
	Name string `json:"name"`
	// Slug defines campaign slugs.
	Slug string `json:"slug"`
	// Channel defines target channel values.
	Channel string `json:"channel"`
	// SegmentID defines target segment identifier values.
	SegmentID string `json:"segmentId"`
	// Subject defines email subject values.
	Subject string `json:"subject"`
	// HTMLBody defines html content values.
	HTMLBody string `json:"htmlBody"`
	// TextBody defines text content values.
	TextBody string `json:"textBody"`
	// TemplateVars defines campaign-level custom variable values.
	TemplateVars map[string]string `json:"templateVars"`
	// ProductBlocks defines product recommendation block configurations.
	ProductBlocks []productBlockRequest `json:"productBlocks"`
}

// updateRequest defines update request payload values.
type updateRequest struct {
	// Name defines optional campaign names.
	Name *string `json:"name"`
	// Slug defines optional campaign slugs.
	Slug *string `json:"slug"`
	// Channel defines optional target channel values.
	Channel *string `json:"channel"`
	// SegmentID defines optional target segment identifier values.
	SegmentID *string `json:"segmentId"`
	// Subject defines optional email subject values.
	Subject *string `json:"subject"`
	// HTMLBody defines optional html content values.
	HTMLBody *string `json:"htmlBody"`
	// TextBody defines optional text content values.
	TextBody *string `json:"textBody"`
	// TemplateVars defines optional campaign-level custom variable values.
	TemplateVars map[string]string `json:"templateVars"`
	// ProductBlocks defines optional product recommendation block configurations.
	ProductBlocks []productBlockRequest `json:"productBlocks"`
}

// NewHandler creates campaign HTTP handlers.
func NewHandler(service Service, authorizers ...Authorizer) (*Handler, error) {
	if service == nil {
		return nil, ErrNilService
	}

	var authorizer Authorizer
	if len(authorizers) > 0 {
		authorizer = authorizers[0]
	}

	return &Handler{service: service, authorizer: authorizer}, nil
}

// SetAuthorizer configures endpoint authentication and permission dependencies.
func (h *Handler) SetAuthorizer(authorizer Authorizer) {
	if h == nil {
		return
	}

	h.authorizer = authorizer
}

// RegisterRoutes registers campaign routes.
func (h *Handler) RegisterRoutes(router corehttp.Router) {
	router.Post("/campaigns", h.protect("marketing:manage", h.create))
	router.Get("/campaigns", h.protect("marketing:manage", h.list))
	router.Get("/campaigns/:id", h.protect("marketing:manage", h.get))
	router.Patch("/campaigns/:id", h.protect("marketing:manage", h.update))
	router.Delete("/campaigns/:id", h.protect("marketing:manage", h.remove))
	router.Post("/campaigns/:id/send", h.protect("marketing:manage", h.send))
	router.Post("/campaigns/:id/test", h.protect("marketing:manage", h.testSend))
	router.Get("/campaigns/:id/deliveries", h.protect("marketing:manage", h.listDeliveries))
}

// create handles campaign create requests.
func (h *Handler) create(ctx corehttp.Context) error {
	request := createRequest{}
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	campaign, err := h.service.Create(ctx.Context(), application.CreateCommand{
		Name:          request.Name,
		Slug:          request.Slug,
		Channel:       request.Channel,
		SegmentID:     request.SegmentID,
		Subject:       request.Subject,
		HTMLBody:      request.HTMLBody,
		TextBody:      request.TextBody,
		TemplateVars:  request.TemplateVars,
		ProductBlocks: mapProductBlockRequests(request.ProductBlocks),
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(201).JSON(campaign)
}

// list handles campaign list requests.
func (h *Handler) list(ctx corehttp.Context) error {
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	limit, _ := strconv.Atoi(ctx.Query("limit", "20"))

	result, err := h.service.List(ctx.Context(), page, limit)
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(result)
}

// get handles campaign by-id requests.
func (h *Handler) get(ctx corehttp.Context) error {
	campaign, err := h.service.Get(ctx.Context(), strings.TrimSpace(ctx.Params("id")))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(campaign)
}

// update handles campaign update requests.
func (h *Handler) update(ctx corehttp.Context) error {
	request := updateRequest{}
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	campaign, err := h.service.Update(ctx.Context(), strings.TrimSpace(ctx.Params("id")), application.UpdateCommand{
		Name:          request.Name,
		Slug:          request.Slug,
		Channel:       request.Channel,
		SegmentID:     request.SegmentID,
		Subject:       request.Subject,
		HTMLBody:      request.HTMLBody,
		TextBody:      request.TextBody,
		TemplateVars:  request.TemplateVars,
		ProductBlocks: mapProductBlockRequests(request.ProductBlocks),
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(campaign)
}

// remove handles campaign delete requests.
func (h *Handler) remove(ctx corehttp.Context) error {
	if err := h.service.Delete(ctx.Context(), strings.TrimSpace(ctx.Params("id"))); err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(map[string]string{"status": "deleted"})
}

// send handles campaign send requests.
func (h *Handler) send(ctx corehttp.Context) error {
	campaign, err := h.service.Send(ctx.Context(), strings.TrimSpace(ctx.Params("id")))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(202).JSON(campaign)
}

// protect wraps endpoint handlers with optional authentication and permission checks.
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

// mapProductBlockRequests maps HTTP product block requests to domain product blocks.
func mapProductBlockRequests(reqs []productBlockRequest) []domain.ProductBlock {
	if len(reqs) == 0 {
		return nil
	}
	blocks := make([]domain.ProductBlock, 0, len(reqs))
	for _, r := range reqs {
		blocks = append(blocks, domain.ProductBlock{
			ID:                  r.ID,
			BaseTag:             r.BaseTag,
			BaseTags:            r.BaseTags,
			BaseTagMode:         r.BaseTagMode,
			UseAffinity:         r.UseAffinity,
			AffinityMinScorePct: r.AffinityMinScorePct,
			CategoryID:          r.CategoryID,
			Realm:               r.Realm,
			Limit:               r.Limit,
			PinnedProductIDs:    r.PinnedProductIDs,
			ExcludeProductIDs:   r.ExcludeProductIDs,
			FilterVariationIDs:  r.FilterVariationIDs,
			PreferVariationIDs:  r.PreferVariationIDs,
		})
	}
	return blocks
}

// mapError maps app/auth errors to HTTP-layer app errors.
func (h *Handler) mapError(err error) error {
	if h != nil && h.authorizer != nil {
		if h.authorizer.IsUnauthorized(err) {
			return corehttp.NewAppError(401, "unauthorized", err)
		}
		if h.authorizer.IsForbidden(err) {
			return corehttp.NewAppError(403, "forbidden", err)
		}
	}
	if errors.Is(err, domain.ErrInvalidID) || errors.Is(err, domain.ErrInvalidName) || errors.Is(err, domain.ErrInvalidSlug) || errors.Is(err, domain.ErrInvalidTestEmail) {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}
	if errors.Is(err, domain.ErrInvalidTemplate) {
		return corehttp.NewAppError(400, "invalid_template", err)
	}
	if errors.Is(err, domain.ErrSenderNotConfigured) {
		return corehttp.NewAppError(503, "email_sender_not_configured", err)
	}
	if errors.Is(err, domain.ErrSenderUnavailable) {
		return corehttp.NewAppError(503, "email_sender_unavailable", err)
	}
	if errors.Is(err, domain.ErrNotFound) {
		return corehttp.NewAppError(404, "campaign_not_found", err)
	}
	if errors.Is(err, domain.ErrSendConflict) {
		return corehttp.NewAppError(409, "campaign_send_conflict", err)
	}

	return corehttp.NewAppError(500, "internal_server_error", err)
}
