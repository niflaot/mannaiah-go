package main

import (
	"context"
	"errors"
	"strings"

	ordersapplication "mannaiah/module/orders/application"
	ordersdomain "mannaiah/module/orders/domain"
	ordersport "mannaiah/module/orders/port"
	productdomain "mannaiah/module/products/domain/product"
	shippingport "mannaiah/module/shipping/port"
)

// shippingOrderQuotationService defines order lookup behavior required by the shipping quotation order source adapter.
type shippingOrderQuotationService interface {
	// Get resolves one order by internal identifier.
	Get(ctx context.Context, id string) (*ordersdomain.Order, error)
	// List lists paginated order aggregate values.
	List(ctx context.Context, query ordersapplication.ListQuery) (*ordersapplication.ListResult, error)
}

// shippingProductQuotationService defines product lookup behavior required by the shipping quotation product source adapter.
type shippingProductQuotationService interface {
	// GetBySKU retrieves a product by product-level or variant-level SKU.
	GetBySKU(ctx context.Context, sku string) (*productdomain.Product, error)
}

// shippingOrderQuotationSourceAdapter adapts the orders service to the shipping OrderQuotationSource port.
type shippingOrderQuotationSourceAdapter struct {
	// orders defines order lookup dependencies.
	orders shippingOrderQuotationService
}

// GetByIDOrIdentifier resolves order quotation data by internal ID or external identifier.
func (a shippingOrderQuotationSourceAdapter) GetByIDOrIdentifier(ctx context.Context, identifier string) (*shippingport.OrderQuotationData, error) {
	if a.orders == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(identifier)
	if trimmed == "" {
		return nil, nil
	}

	order, err := a.orders.Get(ctx, trimmed)
	if err != nil && !errors.Is(err, ordersport.ErrNotFound) && !isOrderNotFound(err) {
		result, listErr := a.orders.List(ctx, ordersapplication.ListQuery{Identifier: trimmed, Limit: 1})
		if listErr != nil {
			return nil, listErr
		}
		if result == nil || len(result.Data) == 0 {
			return nil, nil
		}
		order = &result.Data[0]
	} else if err != nil {
		result, listErr := a.orders.List(ctx, ordersapplication.ListQuery{Identifier: trimmed, Limit: 1})
		if listErr != nil {
			return nil, listErr
		}
		if result == nil || len(result.Data) == 0 {
			return nil, nil
		}
		order = &result.Data[0]
	}

	if order == nil {
		return nil, nil
	}

	var totalValue float64
	items := make([]shippingport.OrderQuotationItem, 0, len(order.Items))
	for _, item := range order.Items {
		sku := strings.TrimSpace(item.SKU)
		if sku == "" {
			continue
		}
		totalValue += item.Value * float64(item.Quantity)
		items = append(items, shippingport.OrderQuotationItem{
			SKU:      sku,
			Quantity: item.Quantity,
		})
	}

	return &shippingport.OrderQuotationData{
		OrderID:         order.ID,
		OrderIdentifier: order.Identifier,
		DestCityCode:    order.ShippingAddress.CityCode,
		TotalValue:      totalValue,
		Items:           items,
	}, nil
}

// isOrderNotFound reports whether an error indicates a missing order record.
func isOrderNotFound(err error) bool {
	if err == nil {
		return false
	}

	return errors.Is(err, ordersport.ErrNotFound) || strings.Contains(strings.ToLower(err.Error()), "not found")
}

// shippingProductQuotationSourceAdapter adapts the products service to the shipping OrderProductSource port.
type shippingProductQuotationSourceAdapter struct {
	// products defines product lookup dependencies.
	products shippingProductQuotationService
}

// GetShippingAttributes resolves shipping dimension and packing attributes for one SKU.
func (a shippingProductQuotationSourceAdapter) GetShippingAttributes(ctx context.Context, sku string) (*shippingport.ProductShippingAttributes, error) {
	if a.products == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(sku)
	if trimmed == "" {
		return nil, nil
	}

	product, err := a.products.GetBySKU(ctx, trimmed)
	if err != nil || product == nil {
		return nil, nil
	}

	var defaultAttrs map[string]any
	for _, ds := range product.Datasheets {
		if strings.EqualFold(strings.TrimSpace(ds.Realm), "default") {
			defaultAttrs = ds.Attributes
			break
		}
	}
	if defaultAttrs == nil {
		return &shippingport.ProductShippingAttributes{SKU: trimmed, Valid: false}, nil
	}

	weightKG := extractFloat(defaultAttrs, "pweight")
	heightCM := extractFloat(defaultAttrs, "pheight")
	widthCM := extractFloat(defaultAttrs, "pwidth")
	lengthCM := extractFloat(defaultAttrs, "plength")
	price := extractFloat(defaultAttrs, "price")
	overlapped := extractBool(defaultAttrs, "overlapped", true)

	valid := weightKG > 0 && heightCM > 0 && widthCM > 0 && lengthCM > 0
	if price <= 0 {
		price = 1
	}

	return &shippingport.ProductShippingAttributes{
		SKU:        trimmed,
		WeightKG:   weightKG,
		HeightCM:   heightCM,
		WidthCM:    widthCM,
		LengthCM:   lengthCM,
		Price:      price,
		Overlapped: overlapped,
		Valid:       valid,
	}, nil
}

// extractFloat reads a float64 attribute value from a map by key.
func extractFloat(attrs map[string]any, key string) float64 {
	if attrs == nil {
		return 0
	}
	val, ok := attrs[key]
	if !ok {
		return 0
	}
	switch v := val.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}

// extractBool reads a bool attribute value from a map by key, returning defaultValue when absent.
func extractBool(attrs map[string]any, key string, defaultValue bool) bool {
	if attrs == nil {
		return defaultValue
	}
	val, ok := attrs[key]
	if !ok {
		return defaultValue
	}
	b, ok := val.(bool)
	if !ok {
		return defaultValue
	}

	return b
}
