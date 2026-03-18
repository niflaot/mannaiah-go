package http

import (
	"context"
	"errors"
	"strings"

	corehttp "mannaiah/module/core/http"
	"mannaiah/module/shipping/domain"
)

var (
	// ErrNilService is returned when quote service dependencies are nil.
	ErrNilService = errors.New("shipping quote service must not be nil")
)

// Authorizer defines authentication and authorization behavior required by shipping endpoints.
type Authorizer interface {
	// Require authenticates and authorizes requests using required permissions.
	Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
	// IsUnauthorized reports authentication errors.
	IsUnauthorized(err error) bool
	// IsForbidden reports authorization errors.
	IsForbidden(err error) bool
}

// Service defines shipping quote use-case behavior required by HTTP handlers.
type Service interface {
	// Quote retrieves one shipping quote from one carrier.
	Quote(ctx context.Context, request domain.QuoteRequest) (*domain.QuoteResult, error)
}

// Handler defines HTTP route handlers for shipping quote endpoints.
type Handler struct {
	// service defines shipping quote use-case dependencies.
	service Service
	// authorizer defines optional auth dependency for protected endpoints.
	authorizer Authorizer
}

// quoteRequest defines quote request payload values.
type quoteRequest struct {
	// Carrier defines carrier identifier values.
	Carrier string `json:"carrier"`
	// BusinessUnit defines business-unit values.
	BusinessUnit string `json:"businessUnit"`
	// OriginCityCode defines origin city code values.
	OriginCityCode string `json:"originCityCode"`
	// DestinationCityCode defines destination city code values.
	DestinationCityCode string `json:"destinationCityCode"`
	// DeclaredValue defines declared merchandise value.
	DeclaredValue float64 `json:"declaredValue"`
	// Units defines package unit payload values.
	Units []quoteUnitRequest `json:"units"`
}

// quoteUnitRequest defines quote unit request payload values.
type quoteUnitRequest struct {
	// Number defines sequential unit number values.
	Number int `json:"number"`
	// RealWeight defines real weight values.
	RealWeight float64 `json:"realWeight"`
	// Height defines package height values.
	Height float64 `json:"height"`
	// Width defines package width values.
	Width float64 `json:"width"`
	// Length defines package length values.
	Length float64 `json:"length"`
}

// quoteResponse defines quote response payload values.
type quoteResponse struct {
	// CarrierMessage defines provider success message values.
	CarrierMessage string `json:"carrierMessage"`
	// QuoteValue defines quoted value payload values.
	QuoteValue float64 `json:"quoteValue"`
	// BusinessUnit defines normalized business-unit identifier values.
	BusinessUnit string `json:"businessUnit"`
}

// NewHandler creates shipping HTTP handlers.
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

// RegisterRoutes registers shipping quote endpoints.
func (h *Handler) RegisterRoutes(router corehttp.Router) {
	router.Post("/shipping/quotes", h.protect("shipping:quote", h.quote))
}

// quote handles shipping quote requests.
func (h *Handler) quote(ctx corehttp.Context) error {
	request := quoteRequest{}
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	units := make([]domain.QuoteUnit, 0, len(request.Units))
	for _, unit := range request.Units {
		units = append(units, domain.QuoteUnit{
			Number:     unit.Number,
			RealWeight: unit.RealWeight,
			Height:     unit.Height,
			Width:      unit.Width,
			Length:     unit.Length,
		})
	}

	result, err := h.service.Quote(ctx.Context(), domain.QuoteRequest{
		Carrier:             domain.NormalizeCarrier(request.Carrier),
		BusinessUnit:        domain.NormalizeBusinessUnit(request.BusinessUnit),
		OriginCityCode:      strings.TrimSpace(request.OriginCityCode),
		DestinationCityCode: strings.TrimSpace(request.DestinationCityCode),
		DeclaredValue:       request.DeclaredValue,
		Units:               units,
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(quoteResponse{
		CarrierMessage: result.CarrierMessage,
		QuoteValue:     result.QuoteValue,
		BusinessUnit:   strings.ToUpper(string(result.BusinessUnit)),
	})
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
	if errors.Is(err, domain.ErrCarrierRequired) || errors.Is(err, domain.ErrUnsupportedCarrier) ||
		errors.Is(err, domain.ErrBusinessUnitRequired) || errors.Is(err, domain.ErrInvalidBusinessUnit) ||
		errors.Is(err, domain.ErrOriginCityCodeRequired) || errors.Is(err, domain.ErrOriginCityCodeInvalid) ||
		errors.Is(err, domain.ErrDestinationCityCodeRequired) || errors.Is(err, domain.ErrDestinationCityCodeInvalid) ||
		errors.Is(err, domain.ErrDeclaredValueInvalid) || errors.Is(err, domain.ErrUnitsRequired) ||
		errors.Is(err, domain.ErrUnitNumberInvalid) || errors.Is(err, domain.ErrUnitNumberSequenceInvalid) ||
		errors.Is(err, domain.ErrUnitRealWeightInvalid) || errors.Is(err, domain.ErrUnitDimensionInvalid) {
		return corehttp.NewAppError(400, "invalid_shipping_quote", err)
	}
	if errors.Is(err, domain.ErrQuoteRejected) {
		return corehttp.NewAppError(502, "shipping_quote_rejected", err)
	}
	if errors.Is(err, domain.ErrIntegrationUnavailable) {
		return corehttp.NewAppError(503, "shipping_integration_unavailable", err)
	}

	return corehttp.NewAppError(500, "internal_server_error", err)
}
