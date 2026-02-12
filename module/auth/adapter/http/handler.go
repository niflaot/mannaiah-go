package http

import (
	"context"
	"errors"

	"mannaiah/module/auth/application"
	corehttp "mannaiah/module/core/http"
)

var (
	// ErrNilService is returned when service dependencies are nil.
	ErrNilService = errors.New("auth service must not be nil")
)

// Service defines auth behavior required by HTTP endpoints.
type Service interface {
	// Require authenticates request headers and validates required permissions.
	Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
}

// Handler defines HTTP route handlers for auth endpoints.
type Handler struct {
	// service defines auth use-case dependencies.
	service Service
}

// checkAuthResponse defines check-auth response payload values.
type checkAuthResponse struct {
	// Status defines authentication status values.
	Status string `json:"status"`
}

// NewHandler creates auth HTTP handlers.
func NewHandler(service Service) (*Handler, error) {
	if service == nil {
		return nil, ErrNilService
	}

	return &Handler{service: service}, nil
}

// RegisterRoutes registers auth endpoints.
func (h *Handler) RegisterRoutes(router corehttp.Router) {
	router.Get("/check-auth", h.checkAuth)
}

// checkAuth validates JWT authentication and returns authenticated status when valid.
func (h *Handler) checkAuth(ctx corehttp.Context) error {
	err := h.service.Require(ctx.Context(), ctx.GetHeader("Authorization"))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(checkAuthResponse{Status: "authenticated"})
}

// mapError maps auth errors to HTTP-layer app errors.
func (h *Handler) mapError(err error) error {
	if errors.Is(err, application.ErrUnauthorized) || errors.Is(err, application.ErrForbidden) {
		return corehttp.NewAppError(401, "unauthorized", err)
	}

	return corehttp.NewAppError(500, "internal_server_error", err)
}
