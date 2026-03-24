package http

import (
	"strconv"
	"strings"

	corehttp "mannaiah/module/core/http"
	quotationservice "mannaiah/module/shipping/application/quotation/service"
	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

// quotationRequest defines quotation request payload values.
type quotationRequest struct {
	// OrderID defines optional order identifier values.
	OrderID string `json:"orderId"`
	// CarrierID defines carrier identifier values.
	CarrierID string `json:"carrierId"`
	// OriginCityCode defines origin city-code values.
	OriginCityCode string `json:"originCityCode"`
	// DestCityCode defines destination city-code values.
	DestCityCode string `json:"destCityCode"`
	// DeclaredValue defines declared shipment value amounts.
	DeclaredValue float64 `json:"declaredValue"`
	// CollectOnDeliveryAmount defines requested cash-on-delivery collection amounts.
	CollectOnDeliveryAmount float64 `json:"collectOnDeliveryAmount"`
	// Units defines package-unit values.
	Units []quotationUnitRequest `json:"units"`
	// ShipmentMode defines the delivery mode for this quotation (parcel or express).
	ShipmentMode domain.ShipmentMode `json:"shipmentMode"`
}

// quotationUnitRequest defines quotation package-unit payload values.
type quotationUnitRequest struct {
	// Description defines package description values.
	Description string `json:"description"`
	// PackageType defines package-type values.
	PackageType string `json:"packageType"`
	// Dimensions defines package dimensions.
	Dimensions domain.Dimensions `json:"dimensions"`
}

// quotationListResponse defines quotation list response values.
type quotationListResponse struct {
	// Data defines quotation rows.
	Data []port.QuotationRecord `json:"data"`
	// Total defines total row count values.
	Total int `json:"total"`
}

// createQuotation handles shipping quotation creation requests.
func (h *Handler) createQuotation(ctx corehttp.Context) error {
	request := quotationRequest{}
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}
	units := make([]domain.PackageUnit, 0, len(request.Units))
	for _, unit := range request.Units {
		units = append(units, domain.PackageUnit{
			Description: strings.TrimSpace(unit.Description),
			PackageType: strings.TrimSpace(unit.PackageType),
			Dimensions:  unit.Dimensions,
		})
	}
	result, err := h.quotations.Quote(ctx.Context(), quotationservice.QuoteCommand{
		OrderID:                 strings.TrimSpace(request.OrderID),
		CarrierID:               strings.TrimSpace(request.CarrierID),
		OriginCityCode:          strings.TrimSpace(request.OriginCityCode),
		DestCityCode:            strings.TrimSpace(request.DestCityCode),
		Units:                   units,
		DeclaredValue:           request.DeclaredValue,
		CollectOnDeliveryAmount: request.CollectOnDeliveryAmount,
		ShipmentMode:            request.ShipmentMode,
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(201).JSON(result)
}

// listQuotations handles shipping quotation list requests.
func (h *Handler) listQuotations(ctx corehttp.Context) error {
	rows, err := h.quotations.ListByOrderID(ctx.Context(), strings.TrimSpace(ctx.Query("orderID")))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(quotationListResponse{Data: rows, Total: len(rows)})
}

// parsePageLimit parses page/limit query values.
func parsePageLimit(ctx corehttp.Context, defaultLimit int) (int, int) {
	page, _ := strconv.Atoi(strings.TrimSpace(ctx.Query("page", "1")))
	if page <= 0 {
		page = 1
	}
	limit, _ := strconv.Atoi(strings.TrimSpace(ctx.Query("limit", strconv.Itoa(defaultLimit))))
	if limit <= 0 {
		limit = defaultLimit
	}

	return page, limit
}
