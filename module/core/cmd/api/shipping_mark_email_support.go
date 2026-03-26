package main

import (
	"context"
	"net/url"
	"sort"
	"strings"

	campaigntemplate "mannaiah/module/campaign/application/template"
	contactdomain "mannaiah/module/contacts/domain"
	ordersdomain "mannaiah/module/orders/domain"
	productdomain "mannaiah/module/products/domain/product"
	shippingdomain "mannaiah/module/shipping/domain"
)

const (
	// shippingTemplateRealmDefault defines the realm used for transactional shipping product rendering.
	shippingTemplateRealmDefault = "default"
)

// shippingDispatchedTemplateData defines transactional template values for shipping dispatched emails.
type shippingDispatchedTemplateData struct {
	// ShippingNumber defines mark/shipping document identifier values.
	ShippingNumber string
	// OrderNumber defines public order identifier values.
	OrderNumber string
	// FirstName defines recipient first-name values.
	FirstName string
	// CarrierName defines carrier display-name values.
	CarrierName string
	// CarrierID defines carrier identifier values.
	CarrierID string
	// TrackingNumber defines carrier tracking-number values.
	TrackingNumber string
	// TrackingURL defines tracking call-to-action URL values.
	TrackingURL string
	// HelpURL defines help call-to-action URL values.
	HelpURL string
	// Billing defines billing-address values.
	Billing shippingDispatchedAddressData
	// Shipping defines shipping-address values.
	Shipping shippingDispatchedAddressData
	// PaymentMethod defines payment method values.
	PaymentMethod string
	// Items defines ordered item values.
	Items []shippingDispatchedItemData
}

// shippingDispatchedAddressData defines address payload values used in shipping transactional templates.
type shippingDispatchedAddressData struct {
	// Name defines full name values.
	Name string
	// Address1 defines address line 1 values.
	Address1 string
	// Address2 defines address line 2 values.
	Address2 string
	// City defines city values.
	City string
	// Department defines department/state values.
	Department string
}

// shippingDispatchedItemData defines product item payload values used in shipping transactional templates.
type shippingDispatchedItemData struct {
	// ProductName defines product display-name values.
	ProductName string
	// SKU defines item sku values.
	SKU string
	// Variation defines variation label values.
	Variation string
	// Quantity defines ordered quantity values.
	Quantity int
	// ImageURL defines product image URL values.
	ImageURL string
}

// shippingDispatchedRenderMeta defines computed render metadata values.
type shippingDispatchedRenderMeta struct {
	// OrderNumber defines public order identifier values.
	OrderNumber string
	// CarrierName defines carrier display-name values.
	CarrierName string
	// TrackingNumber defines mark tracking-number values.
	TrackingNumber string
	// ShippingNumber defines mark document reference values.
	ShippingNumber string
}

// buildShippingDispatchedTemplateData resolves full template data for shipping dispatched transactional emails.
func buildShippingDispatchedTemplateData(
	ctx context.Context,
	deps shippingEmailConsumerDependencies,
	order ordersdomain.Order,
	contact contactdomain.Contact,
	mark shippingdomain.ShippingMark,
	meta shippingDispatchedRenderMeta,
) shippingDispatchedTemplateData {
	contactName := resolveContactDisplayName(&contact)
	firstName := firstNonEmpty(campaigntemplate.ExtractFirstName(contactName), contactName, contact.Email)
	shippingAddress := resolveShippingDispatchedAddress(contact, order, true)
	billingAddress := resolveShippingDispatchedAddress(contact, order, false)

	return shippingDispatchedTemplateData{
		ShippingNumber: firstNonEmpty(meta.ShippingNumber, meta.TrackingNumber, mark.ID),
		OrderNumber:    firstNonEmpty(meta.OrderNumber, order.ID),
		FirstName:      firstName,
		CarrierName:    firstNonEmpty(meta.CarrierName, mark.CarrierID),
		CarrierID:      strings.TrimSpace(mark.CarrierID),
		TrackingNumber: firstNonEmpty(meta.TrackingNumber, mark.TrackingNumber, mark.DocumentRef, mark.ID),
		TrackingURL:    buildShippingTrackingURL(firstNonEmpty(meta.TrackingNumber, mark.TrackingNumber, mark.DocumentRef, mark.ID), mark.CarrierID, firstNonEmpty(meta.OrderNumber, order.ID)),
		HelpURL:        buildShippingHelpURL(firstNonEmpty(meta.OrderNumber, order.ID)),
		Billing:        billingAddress,
		Shipping:       shippingAddress,
		PaymentMethod:  strings.ToUpper(firstNonEmpty(order.PaymentMethod, "NO ESPECIFICADO")),
		Items:          buildShippingDispatchedItems(ctx, deps, order.Items),
	}
}

// resolveShippingDispatchedAddress resolves shipping/billing address values with contact fallbacks.
func resolveShippingDispatchedAddress(contact contactdomain.Contact, order ordersdomain.Order, shipping bool) shippingDispatchedAddressData {
	base := shippingDispatchedAddressData{
		Name:       firstNonEmpty(resolveContactDisplayName(&contact), contact.Email),
		Address1:   firstNonEmpty(contact.Address),
		Address2:   firstNonEmpty(contact.AddressExtra),
		City:       firstNonEmpty(contact.CityCode),
		Department: firstNonEmpty(contact.Metadata["department"], contact.Metadata["state"], contact.Metadata["_billing_state"]),
	}
	if shipping {
		base.Name = firstNonEmpty(base.Name)
		base.Address1 = firstNonEmpty(order.ShippingAddress.Address, base.Address1)
		base.Address2 = firstNonEmpty(order.ShippingAddress.Address2, base.Address2)
		base.City = firstNonEmpty(order.ShippingAddress.CityCode, base.City)
		base.Department = firstNonEmpty(order.Metadata["_shipping_state"], order.Metadata["shipping_state"], base.Department)
		return base
	}

	base.Department = firstNonEmpty(order.Metadata["_billing_state"], order.Metadata["billing_state"], base.Department)
	return base
}

// buildShippingDispatchedItems resolves template item values from order item rows.
func buildShippingDispatchedItems(ctx context.Context, deps shippingEmailConsumerDependencies, items []ordersdomain.Item) []shippingDispatchedItemData {
	result := make([]shippingDispatchedItemData, 0, len(items))
	for _, item := range items {
		result = append(result, buildShippingDispatchedItem(ctx, deps, item))
	}

	return result
}

// buildShippingDispatchedItem resolves one transactional shipping item payload.
func buildShippingDispatchedItem(ctx context.Context, deps shippingEmailConsumerDependencies, item ordersdomain.Item) shippingDispatchedItemData {
	data := shippingDispatchedItemData{
		ProductName: firstNonEmpty(item.AlternateName, item.SKU, "Producto"),
		SKU:         firstNonEmpty(item.SKU, "-"),
		Variation:   "-",
		Quantity:    maxInt(item.Quantity, 1),
	}
	if deps.products == nil || strings.TrimSpace(item.SKU) == "" {
		return data
	}

	product, err := deps.products.GetBySKU(ctx, strings.TrimSpace(item.SKU))
	if err != nil || product == nil {
		return data
	}

	data.ProductName = firstNonEmpty(resolveProductDisplayName(*product), data.ProductName)
	data.SKU = firstNonEmpty(strings.TrimSpace(item.SKU), strings.TrimSpace(product.SKU), data.SKU)

	variationIDs := resolveVariantIDsBySKU(*product, item.SKU)
	variationLabels := resolveVariationLabels(ctx, deps, variationIDs)
	if len(variationLabels) > 0 {
		data.Variation = strings.Join(variationLabels, " / ")
	}
	data.ImageURL = resolveProductImageURL(ctx, deps.assetResolver, *product, variationIDs)
	return data
}

// resolveProductDisplayName resolves the best product name from default-realm datasheets.
func resolveProductDisplayName(product productdomain.Product) string {
	var fallback string
	for _, datasheet := range product.Datasheets {
		if fallback == "" {
			fallback = strings.TrimSpace(datasheet.Name)
		}
		if strings.EqualFold(strings.TrimSpace(datasheet.Realm), shippingTemplateRealmDefault) && strings.TrimSpace(datasheet.Name) != "" {
			return strings.TrimSpace(datasheet.Name)
		}
	}

	return firstNonEmpty(fallback, product.SKU)
}

// resolveVariantIDsBySKU resolves ordered variant variation ids for one item sku.
func resolveVariantIDsBySKU(product productdomain.Product, itemSKU string) []string {
	normalizedSKU := strings.TrimSpace(itemSKU)
	if normalizedSKU == "" {
		return nil
	}
	for _, variant := range product.Variants {
		if !strings.EqualFold(strings.TrimSpace(variant.SKU), normalizedSKU) {
			continue
		}
		return normalizeVariationIDs(variant.VariationIDs)
	}
	if len(product.Variants) == 1 {
		return normalizeVariationIDs(product.Variants[0].VariationIDs)
	}

	return nil
}

// normalizeVariationIDs normalizes and deduplicates variation ids.
func normalizeVariationIDs(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

// resolveVariationLabels resolves variation display labels by id.
func resolveVariationLabels(ctx context.Context, deps shippingEmailConsumerDependencies, variationIDs []string) []string {
	if deps.variations == nil || len(variationIDs) == 0 {
		return nil
	}
	rows := make([]string, 0, len(variationIDs))
	for _, variationID := range variationIDs {
		entity, err := deps.variations.Get(ctx, variationID)
		if err != nil || entity == nil {
			continue
		}
		rows = append(rows, firstNonEmpty(strings.TrimSpace(entity.Name), strings.TrimSpace(entity.Value)))
	}
	return rows
}

// resolveProductImageURL resolves one product image URL using default-realm visibility and variant-preferred rows.
func resolveProductImageURL(ctx context.Context, resolver analyticsAssetURLResolver, product productdomain.Product, preferVariationIDs []string) string {
	gallery := append([]productdomain.GalleryItem{}, product.Gallery...)
	sort.SliceStable(gallery, func(i int, j int) bool {
		left := normalizeGalleryPosition(gallery[i].Position)
		right := normalizeGalleryPosition(gallery[j].Position)
		if left == right {
			return strings.ToLower(strings.TrimSpace(gallery[i].AssetID)) < strings.ToLower(strings.TrimSpace(gallery[j].AssetID))
		}
		return left < right
	})
	if len(preferVariationIDs) > 0 {
		preferSet := map[string]struct{}{}
		for _, variationID := range preferVariationIDs {
			preferSet[strings.TrimSpace(variationID)] = struct{}{}
		}
		for _, galleryItem := range gallery {
			if !galleryVisibleInDefaultRealm(galleryItem) {
				continue
			}
			for _, variationID := range galleryItem.VariationIDs {
				if _, ok := preferSet[strings.TrimSpace(variationID)]; !ok {
					continue
				}
				if value := firstNonEmpty(strings.TrimSpace(resolver.ResolveURL(ctx, galleryItem.AssetID)), strings.TrimSpace(galleryItem.AssetID)); strings.HasPrefix(strings.ToLower(value), "http://") || strings.HasPrefix(strings.ToLower(value), "https://") {
					return value
				}
			}
		}
	}
	for _, galleryItem := range gallery {
		if !galleryVisibleInDefaultRealm(galleryItem) {
			continue
		}
		value := firstNonEmpty(strings.TrimSpace(resolver.ResolveURL(ctx, galleryItem.AssetID)), strings.TrimSpace(galleryItem.AssetID))
		if strings.HasPrefix(strings.ToLower(value), "http://") || strings.HasPrefix(strings.ToLower(value), "https://") {
			return value
		}
	}
	return ""
}

// normalizeGalleryPosition normalizes nullable gallery positions.
func normalizeGalleryPosition(value *int) int {
	if value == nil {
		return 0
	}
	if *value < 0 {
		return 0
	}
	return *value
}

// galleryVisibleInDefaultRealm reports whether gallery rows are visible in the default realm.
func galleryVisibleInDefaultRealm(item productdomain.GalleryItem) bool {
	if len(item.IncludedRealms) == 0 {
		return true
	}
	for _, realm := range item.IncludedRealms {
		if strings.EqualFold(strings.TrimSpace(realm), shippingTemplateRealmDefault) {
			return true
		}
	}
	return false
}

// buildShippingTrackingURL builds the tracking call-to-action URL.
func buildShippingTrackingURL(trackingNumber string, carrierID string, orderNumber string) string {
	values := url.Values{}
	if strings.TrimSpace(trackingNumber) != "" {
		values.Set("tracking", strings.TrimSpace(trackingNumber))
	}
	if strings.TrimSpace(carrierID) != "" {
		values.Set("carrier", strings.TrimSpace(carrierID))
	}
	if strings.TrimSpace(orderNumber) != "" {
		values.Set("order", strings.TrimSpace(orderNumber))
	}
	if encoded := strings.TrimSpace(values.Encode()); encoded != "" {
		return "https://rastreo.flockstore.co/?" + strings.ReplaceAll(encoded, "+", "%20")
	}

	return "https://rastreo.flockstore.co"
}

// buildShippingHelpURL builds the help call-to-action URL.
func buildShippingHelpURL(orderNumber string) string {
	message := "FLK. Solicito ayuda con mi pedido " + strings.TrimSpace(orderNumber) + ","
	return "https://wa.me/573104314990?text=" + strings.ReplaceAll(url.QueryEscape(message), "+", "%20")
}

// maxInt resolves max integer values.
func maxInt(value int, minimum int) int {
	if value < minimum {
		return minimum
	}
	return value
}
