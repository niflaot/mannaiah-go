package http

import (
	"context"
	"errors"
	"strings"
	"time"

	corehttp "mannaiah/module/core/http"
	"mannaiah/module/membership/application"
	"mannaiah/module/membership/domain"
	"mannaiah/module/membership/port"
)

var (
	// ErrNilService is returned when service dependencies are nil.
	ErrNilService = errors.New("membership service must not be nil")
)

// Authorizer defines authentication and authorization behavior required by membership endpoints.
type Authorizer interface {
	// Require authenticates and authorizes requests using required permissions.
	Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
	// IsUnauthorized reports authentication errors.
	IsUnauthorized(err error) bool
	// IsForbidden reports authorization errors.
	IsForbidden(err error) bool
}

// Service defines membership use-case behavior required by HTTP handlers.
type Service interface {
	// Stamp persists membership stamps and updates latest status snapshots.
	Stamp(ctx context.Context, command port.StampCommand) (*domain.Status, error)
	// GetStatus retrieves one current status by contact and channel.
	GetStatus(ctx context.Context, contactID string, channel domain.Channel) (*domain.Status, error)
	// ListStamps retrieves stamps by contact and channel filters.
	ListStamps(ctx context.Context, contactID string, channel domain.Channel, limit int) ([]domain.Stamp, error)
	// MigrateFromContactMetadata migrates legacy contact metadata values to membership stamps.
	MigrateFromContactMetadata(ctx context.Context, pageSize int) (*application.MigrateSummary, error)
}

// Handler defines HTTP route handlers for membership endpoints.
type Handler struct {
	// service defines membership use-case dependencies.
	service Service
	// authorizer defines optional auth dependencies.
	authorizer Authorizer
}

// stampRequest defines stamp endpoint request payload values.
type stampRequest struct {
	// ContactID defines optional contact identifier values.
	ContactID string `json:"contactId"`
	// Email defines optional lookup email values.
	Email string `json:"email"`
	// Channel defines channel values.
	Channel string `json:"channel"`
	// Action defines action values.
	Action string `json:"action"`
	// Source defines source values.
	Source string `json:"source"`
	// OccurredAt defines optional action timestamp values.
	OccurredAt *time.Time `json:"occurredAt"`
}

// actionRequest defines opt-in/opt-out endpoint request payload values.
type actionRequest struct {
	// ContactID defines optional contact identifier values.
	ContactID string `json:"contactId"`
	// Email defines optional lookup email values.
	Email string `json:"email"`
	// Channel defines channel values.
	Channel string `json:"channel"`
	// Source defines source values.
	Source string `json:"source"`
	// OccurredAt defines optional action timestamp values.
	OccurredAt *time.Time `json:"occurredAt"`
}

// migrateRequest defines migration endpoint request payload values.
type migrateRequest struct {
	// PageSize defines batch size values used in migration iteration.
	PageSize int `json:"pageSize"`
}

// NewHandler creates membership HTTP handlers.
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

// RegisterRoutes registers membership routes.
func (h *Handler) RegisterRoutes(router corehttp.Router) {
	router.Post("/membership/optin", h.protect("marketing:manage", h.optIn))
	router.Post("/membership/optout", h.protect("marketing:manage", h.optOut))
	router.Post("/membership/stamp", h.protect("marketing:manage", h.stamp))
	router.Get("/membership/status/:contactId", h.protect("marketing:manage", h.status))
	router.Get("/membership/status/:contactId/:channel", h.protect("marketing:manage", h.statusByChannel))
	router.Get("/membership/status/:contactId/stamps", h.protect("marketing:manage", h.stamps))
	router.Get("/membership/stamps/:contactId/:channel", h.protect("marketing:manage", h.stampsByChannel))
	router.Post("/membership/migrate", h.protect("marketing:manage", h.migrate))
}

// optIn handles membership opt-in requests.
func (h *Handler) optIn(ctx corehttp.Context) error {
	return h.action(ctx, domain.ActionOptIn)
}

// optOut handles membership opt-out requests.
func (h *Handler) optOut(ctx corehttp.Context) error {
	return h.action(ctx, domain.ActionOptOut)
}

// action handles membership action requests.
func (h *Handler) action(ctx corehttp.Context, action domain.Action) error {
	var request actionRequest
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	channel := strings.TrimSpace(request.Channel)
	if channel == "" {
		channel = string(domain.ChannelEmail)
	}

	status, err := h.service.Stamp(ctx.Context(), port.StampCommand{
		ContactID: strings.TrimSpace(request.ContactID),
		Email:     strings.TrimSpace(request.Email),
		Channel:   domain.Channel(channel),
		Action:    action,
		Source:    strings.TrimSpace(request.Source),
		OccurredAt: func() *time.Time {
			if request.OccurredAt == nil {
				return nil
			}
			value := request.OccurredAt.UTC()
			return &value
		}(),
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(status)
}

// stamp handles membership stamp requests.
func (h *Handler) stamp(ctx corehttp.Context) error {
	var request stampRequest
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	status, err := h.service.Stamp(ctx.Context(), port.StampCommand{
		ContactID: strings.TrimSpace(request.ContactID),
		Email:     strings.TrimSpace(request.Email),
		Channel:   domain.Channel(strings.TrimSpace(request.Channel)),
		Action:    domain.Action(strings.TrimSpace(request.Action)),
		Source:    strings.TrimSpace(request.Source),
		OccurredAt: func() *time.Time {
			if request.OccurredAt == nil {
				return nil
			}
			value := request.OccurredAt.UTC()
			return &value
		}(),
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(status)
}

// status handles membership status-by-contact requests.
func (h *Handler) status(ctx corehttp.Context) error {
	status, err := h.service.GetStatus(
		ctx.Context(),
		strings.TrimSpace(ctx.Params("contactId")),
		domain.Channel(strings.TrimSpace(ctx.Query("channel", string(domain.ChannelEmail)))),
	)
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(status)
}

// statusByChannel handles membership status-by-contact-and-channel requests.
func (h *Handler) statusByChannel(ctx corehttp.Context) error {
	status, err := h.service.GetStatus(
		ctx.Context(),
		strings.TrimSpace(ctx.Params("contactId")),
		domain.Channel(strings.TrimSpace(ctx.Params("channel"))),
	)
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(status)
}

// stamps handles membership stamp listing requests.
func (h *Handler) stamps(ctx corehttp.Context) error {
	entries, err := h.service.ListStamps(
		ctx.Context(),
		strings.TrimSpace(ctx.Params("contactId")),
		domain.Channel(strings.TrimSpace(ctx.Query("channel", string(domain.ChannelEmail)))),
		100,
	)
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(entries)
}

// stampsByChannel handles membership stamp listing requests by explicit channel path parameters.
func (h *Handler) stampsByChannel(ctx corehttp.Context) error {
	entries, err := h.service.ListStamps(
		ctx.Context(),
		strings.TrimSpace(ctx.Params("contactId")),
		domain.Channel(strings.TrimSpace(ctx.Params("channel"))),
		100,
	)
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(entries)
}

// migrate handles legacy metadata migration requests.
func (h *Handler) migrate(ctx corehttp.Context) error {
	request := migrateRequest{}
	_ = ctx.BodyParser(&request)

	summary, err := h.service.MigrateFromContactMetadata(ctx.Context(), request.PageSize)
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(summary)
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
	if errors.Is(err, domain.ErrInvalidContactID) ||
		errors.Is(err, domain.ErrInvalidEmail) ||
		errors.Is(err, domain.ErrInvalidChannel) ||
		errors.Is(err, domain.ErrInvalidAction) {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}
	if errors.Is(err, domain.ErrContactNotFound) || errors.Is(err, domain.ErrStatusNotFound) {
		return corehttp.NewAppError(404, "membership_not_found", err)
	}

	return corehttp.NewAppError(500, "internal_server_error", err)
}
