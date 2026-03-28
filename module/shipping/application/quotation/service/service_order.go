package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

const (
	// warningNoProducts is emitted when no valid package units can be resolved from the order.
	warningNoProducts = "NO_PRODUCTS"
	// warningAllOverlapped is emitted when every product is flagged as overlapped.
	warningAllOverlapped = "ALL_OVERLAPPED"
	// warningInvalidCity is emitted when the carrier rejects the destination city code.
	warningInvalidCity = "INVALID_CITY"

	// maxOverlappedPerBox defines the maximum number of overlapped items that fit in one main box.
	maxOverlappedPerBox = 3
)

// QuoteFromOrderCommand defines input values for order-based quotation requests.
type QuoteFromOrderCommand struct {
	// OrderIdentifier defines the order to quote: accepts an internal ID or external identifier.
	OrderIdentifier string
	// CarrierID defines carrier identifier values.
	CarrierID string
	// OriginCityCode defines origin city-code values.
	OriginCityCode string
	// ShipmentMode defines the delivery mode for this quotation (parcel or express).
	ShipmentMode domain.ShipmentMode
}

// packageCandidate holds resolved attributes for one product unit.
type packageCandidate struct {
	attrs  port.ProductShippingAttributes
	volume float64
}

// QuoteFromOrder builds package units from order products and requests a freight quotation.
// It reads shipping attributes from the "default" realm datasheet of each product,
// applies the overlapping box-packing algorithm, and calls Quote() with the resulting units.
func (s *Service) QuoteFromOrder(ctx context.Context, cmd QuoteFromOrderCommand) (*domain.QuotationResult, error) {
	if s.orderSource == nil {
		return nil, errors.New("order source not configured")
	}
	if s.productSource == nil {
		return nil, errors.New("product source not configured")
	}

	orderData, err := s.orderSource.GetByIDOrIdentifier(ctx, strings.TrimSpace(cmd.OrderIdentifier))
	if err != nil {
		return nil, fmt.Errorf("resolve order: %w", err)
	}
	if orderData == nil {
		return nil, domain.ErrInvalidID
	}

	candidates, warnings, buildErr := s.buildCandidates(ctx, orderData.Items)
	if buildErr != nil {
		return nil, buildErr
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no valid products for quotation: %w", domain.ErrInvalidID)
	}

	units := packBoxes(candidates, &warnings)

	result, quoteErr := s.Quote(ctx, QuoteCommand{
		OrderID:                 orderData.OrderID,
		OrderIdentifier:         orderData.OrderIdentifier,
		CarrierID:               strings.TrimSpace(cmd.CarrierID),
		OriginCityCode:          strings.TrimSpace(cmd.OriginCityCode),
		DestCityCode:            orderData.DestCityCode,
		Units:                   units,
		DeclaredValue:           orderData.TotalValue,
		CollectOnDeliveryAmount: orderData.TotalValue,
		ShipmentMode:            cmd.ShipmentMode,
	})
	if quoteErr != nil {
		if isCityError(quoteErr) {
			warnings = append(warnings, domain.QuotationWarning{
				Code:    warningInvalidCity,
				Message: "destination city code was rejected by the carrier",
			})
			if result != nil {
				result.Warnings = warnings
			}

			return result, nil
		}

		return nil, quoteErr
	}

	result.Warnings = warnings

	return result, nil
}

// buildCandidates resolves product shipping attributes for all order items and returns one candidate
// per unit (quantity expanded). Invalid products are silently skipped.
func (s *Service) buildCandidates(ctx context.Context, items []port.OrderQuotationItem) ([]packageCandidate, []domain.QuotationWarning, error) {
	candidates := make([]packageCandidate, 0)
	var warnings []domain.QuotationWarning

	for _, item := range items {
		sku := strings.TrimSpace(item.SKU)
		if sku == "" {
			continue
		}

		attrs, attrErr := s.productSource.GetShippingAttributes(ctx, sku)
		if attrErr != nil || attrs == nil || !attrs.Valid {
			continue
		}

		vol := attrs.HeightCM * attrs.WidthCM * attrs.LengthCM
		for i := 0; i < item.Quantity; i++ {
			candidates = append(candidates, packageCandidate{attrs: *attrs, volume: vol})
		}
	}

	return candidates, warnings, nil
}

// packBoxes applies the overlapping box-packing algorithm to the resolved candidates
// and returns the final set of PackageUnit values.
//
// Algorithm:
//  1. Separate candidates into non-overlapped (standalone boxes) and overlapped (nestable items).
//  2. If all candidates are overlapped, promote the largest one to a main box and emit ALL_OVERLAPPED.
//  3. Fill each main box with up to maxOverlappedPerBox overlapped items (smallest volume first).
//  4. Any remaining overlapped candidates: promote the largest as a new main box and repeat step 3.
func packBoxes(candidates []packageCandidate, warnings *[]domain.QuotationWarning) []domain.PackageUnit {
	var main, overlapped []packageCandidate
	for _, c := range candidates {
		if c.attrs.Overlapped {
			overlapped = append(overlapped, c)
		} else {
			main = append(main, c)
		}
	}

	if len(main) == 0 && len(overlapped) > 0 {
		*warnings = append(*warnings, domain.QuotationWarning{
			Code:    warningAllOverlapped,
			Message: "all products are flagged as overlapped; the largest item is used as the main box",
		})
		sort.Slice(overlapped, func(i, j int) bool { return overlapped[i].volume > overlapped[j].volume })
		main = append(main, overlapped[0])
		overlapped = overlapped[1:]
	}

	sort.Slice(overlapped, func(i, j int) bool { return overlapped[i].volume < overlapped[j].volume })

	for idx := range main {
		var assigned int
		for assigned < maxOverlappedPerBox && len(overlapped) > 0 {
			overlapped = overlapped[1:]
			assigned++
		}
		_ = idx
	}

	for len(overlapped) > 0 {
		sort.Slice(overlapped, func(i, j int) bool { return overlapped[i].volume > overlapped[j].volume })
		main = append(main, overlapped[0])
		overlapped = overlapped[1:]
		var assigned int
		for assigned < maxOverlappedPerBox && len(overlapped) > 0 {
			overlapped = overlapped[1:]
			assigned++
		}
	}

	units := make([]domain.PackageUnit, 0, len(main))
	for _, box := range main {
		units = append(units, domain.PackageUnit{
			Description: box.attrs.SKU,
			PackageType: "parcel",
			Dimensions: domain.Dimensions{
				HeightCM:         box.attrs.HeightCM,
				WidthCM:          box.attrs.WidthCM,
				DepthCM:          box.attrs.LengthCM,
				RealWeightKG:     box.attrs.WeightKG,
				DeclaredValueCOP: box.attrs.Price,
			},
		})
	}

	return units
}

// isCityError reports whether an error originated from an invalid destination city rejection.
func isCityError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())

	return strings.Contains(msg, "city") || strings.Contains(msg, "ciudad") || strings.Contains(msg, "destino")
}
