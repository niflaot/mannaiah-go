package main

import (
	"context"
	"errors"
	"strconv"
	"strings"

	contactdomain "mannaiah/module/contacts/domain"
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

// shippingOrderQuotationContactService defines contact lookup behavior required by the shipping quotation order source adapter.
type shippingOrderQuotationContactService interface {
	// Get resolves one contact by identifier.
	Get(ctx context.Context, id string) (*contactdomain.Contact, error)
}

// shippingProductQuotationService defines product lookup behavior required by the shipping quotation product source adapter.
type shippingProductQuotationService interface {
	// Get retrieves a product by internal ID.
	Get(ctx context.Context, id string) (*productdomain.Product, error)
	// GetBySKU retrieves a product by product-level or variant-level SKU.
	GetBySKU(ctx context.Context, sku string) (*productdomain.Product, error)
}

// shippingOrderQuotationSourceAdapter adapts the orders service to the shipping OrderQuotationSource port.
type shippingOrderQuotationSourceAdapter struct {
	// orders defines order lookup dependencies.
	orders shippingOrderQuotationService
	// contacts defines optional contact lookup dependencies used for recipient enrichment/fallback.
	contacts shippingOrderQuotationContactService
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

	var contact *contactdomain.Contact
	if a.contacts != nil {
		contactID := strings.TrimSpace(order.ContactID)
		if contactID != "" {
			resolvedContact, contactErr := a.contacts.Get(ctx, contactID)
			if contactErr == nil {
				contact = resolvedContact
			}
		}
	}

	var totalValue float64
	items := make([]shippingport.OrderQuotationItem, 0, len(order.Items))
	for _, item := range order.Items {
		sku := strings.TrimSpace(item.SKU)
		productID := strings.TrimSpace(item.ProductID)
		if sku == "" && productID == "" {
			continue
		}
		totalValue += item.Value * float64(item.Quantity)
		items = append(items, shippingport.OrderQuotationItem{
			SKU:       sku,
			ProductID: productID,
			Quantity:  item.Quantity,
		})
	}
	collectOnDeliveryAmount := resolveOrderCollectOnDeliveryAmount(totalValue, order.PaymentMethod)
	shippingAddressLine := strings.TrimSpace(strings.Join([]string{
		strings.TrimSpace(order.ShippingAddress.Address),
	}, " "))
	shippingAddressLine = strings.Join(strings.Fields(shippingAddressLine), " ")
	shippingAddressLine2 := strings.Join(strings.Fields(strings.TrimSpace(order.ShippingAddress.Address2)), " ")
	contactAddressLine := strings.TrimSpace(strings.Join([]string{
		firstNonEmptyTrimmed(contactAddress(contact)),
	}, " "))
	contactAddressLine = strings.Join(strings.Fields(contactAddressLine), " ")
	contactAddressLine2 := strings.Join(strings.Fields(firstNonEmptyTrimmed(contactAddressExtra(contact))), " ")
	recipientCity := firstNonEmptyTrimmed(order.ShippingAddress.CityCode, contactCityCode(contact))
	recipientName := firstNonEmptyTrimmed(resolveContactDisplayName(contact), contactEmail(contact), "Cliente")

	return &shippingport.OrderQuotationData{
		OrderID:                 order.ID,
		OrderIdentifier:         order.Identifier,
		DestCityCode:            recipientCity,
		RecipientCity:           recipientCity,
		TotalValue:              totalValue,
		CollectOnDeliveryAmount: collectOnDeliveryAmount,
		RecipientName:           recipientName,
		RecipientID:             contactDocumentNumber(contact),
		RecipientIDType:         contactDocumentType(contact),
		RecipientAddressLine:    firstNonEmptyTrimmed(shippingAddressLine, contactAddressLine),
		RecipientAddressLine2:   firstNonEmptyTrimmed(shippingAddressLine2, contactAddressLine2),
		RecipientPhone:          firstNonEmptyTrimmed(order.ShippingAddress.Phone, contactPhone(contact)),
		RecipientEmail:          contactEmail(contact),
		Items:                   items,
	}, nil
}

// isOrderNotFound reports whether an error indicates a missing order record.
func isOrderNotFound(err error) bool {
	if err == nil {
		return false
	}

	return errors.Is(err, ordersport.ErrNotFound) || strings.Contains(strings.ToLower(err.Error()), "not found")
}

// resolveOrderCollectOnDeliveryAmount resolves COD amount from order totals and payment method.
func resolveOrderCollectOnDeliveryAmount(totalValue float64, paymentMethod string) float64 {
	if totalValue <= 0 {
		return 0
	}
	if !isCashOnDeliveryPaymentMethod(paymentMethod) {
		return 0
	}

	return totalValue
}

// isCashOnDeliveryPaymentMethod reports whether a payment method maps to COD semantics.
func isCashOnDeliveryPaymentMethod(paymentMethod string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(paymentMethod))
	if trimmed == "" {
		return false
	}
	compacted := strings.NewReplacer("-", "", "_", "", " ", "").Replace(trimmed)
	switch compacted {
	case "cod", "cashondelivery", "payondelivery", "contraentrega", "contrareembolso":
		return true
	}

	return strings.Contains(trimmed, "cash on delivery") || strings.Contains(trimmed, "pay on delivery") || strings.Contains(trimmed, "contra entrega")
}

// firstNonEmptyTrimmed resolves the first non-empty trimmed value.
func firstNonEmptyTrimmed(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}

	return ""
}

// contactEmail resolves contact email values.
func contactEmail(contact *contactdomain.Contact) string {
	if contact == nil {
		return ""
	}

	return strings.TrimSpace(contact.Email)
}

// contactPhone resolves contact phone values.
func contactPhone(contact *contactdomain.Contact) string {
	if contact == nil {
		return ""
	}

	return strings.TrimSpace(contact.Phone)
}

// contactCityCode resolves contact city code values.
func contactCityCode(contact *contactdomain.Contact) string {
	if contact == nil {
		return ""
	}

	return strings.TrimSpace(contact.CityCode)
}

// contactAddress resolves contact address line 1 values.
func contactAddress(contact *contactdomain.Contact) string {
	if contact == nil {
		return ""
	}

	return strings.TrimSpace(contact.Address)
}

// contactAddressExtra resolves contact address line 2 values.
func contactAddressExtra(contact *contactdomain.Contact) string {
	if contact == nil {
		return ""
	}

	return strings.TrimSpace(contact.AddressExtra)
}

// contactDocumentNumber resolves contact document-number values.
func contactDocumentNumber(contact *contactdomain.Contact) string {
	if contact == nil {
		return ""
	}

	return strings.TrimSpace(contact.DocumentNumber)
}

// contactDocumentType resolves contact document-type values.
func contactDocumentType(contact *contactdomain.Contact) string {
	if contact == nil {
		return ""
	}

	return strings.TrimSpace(string(contact.DocumentType))
}

// shippingProductQuotationSourceAdapter adapts the products service to the shipping OrderProductSource port.
type shippingProductQuotationSourceAdapter struct {
	// products defines product lookup dependencies.
	products shippingProductQuotationService
}

// extractProductShippingAttributes extracts shipping attributes from a resolved product's default-realm datasheet.
func extractProductShippingAttributes(product *productdomain.Product, identifier string) *shippingport.ProductShippingAttributes {
	var defaultAttrs map[string]any
	for _, ds := range product.Datasheets {
		if strings.EqualFold(strings.TrimSpace(ds.Realm), "default") {
			defaultAttrs = ds.Attributes
			break
		}
	}
	if defaultAttrs == nil {
		return &shippingport.ProductShippingAttributes{SKU: identifier, Valid: false}
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
		SKU:        identifier,
		WeightKG:   weightKG,
		HeightCM:   heightCM,
		WidthCM:    widthCM,
		LengthCM:   lengthCM,
		Price:      price,
		Overlapped: overlapped,
		Valid:      valid,
	}
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

	return extractProductShippingAttributes(product, trimmed), nil
}

// GetShippingAttributesByID resolves shipping dimension and packing attributes by internal product ID.
func (a shippingProductQuotationSourceAdapter) GetShippingAttributesByID(ctx context.Context, productID string) (*shippingport.ProductShippingAttributes, error) {
	if a.products == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(productID)
	if trimmed == "" {
		return nil, nil
	}

	product, err := a.products.Get(ctx, trimmed)
	if err != nil || product == nil {
		return nil, nil
	}

	return extractProductShippingAttributes(product, trimmed), nil
}

// extractFloat reads a float64 attribute value from a map by key.
// Handles both native numeric types and JSON-string encoded numbers (e.g. "1", "40").
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
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return 0
		}
		return f
	default:
		return 0
	}
}

// extractBool reads a bool attribute value from a map by key, returning defaultValue when absent.
// Handles both native bool and JSON-string encoded booleans (e.g. "true", "false").
func extractBool(attrs map[string]any, key string, defaultValue bool) bool {
	if attrs == nil {
		return defaultValue
	}
	val, ok := attrs[key]
	if !ok {
		return defaultValue
	}
	switch v := val.(type) {
	case bool:
		return v
	case string:
		b, err := strconv.ParseBool(strings.TrimSpace(v))
		if err != nil {
			return defaultValue
		}
		return b
	default:
		return defaultValue
	}
}
