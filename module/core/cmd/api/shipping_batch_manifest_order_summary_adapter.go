package main

import (
	"context"
	"strconv"
	"strings"

	ordersdomain "mannaiah/module/orders/domain"
	productdomain "mannaiah/module/products/domain/product"
	variationdomain "mannaiah/module/products/domain/variation"
	dispatchservice "mannaiah/module/shipping/application/dispatch/service"
	markservice "mannaiah/module/shipping/application/mark/service"
)

// shippingBatchManifestOrderLookupService defines order lookup behavior required by batch manifest summary adapters.
type shippingBatchManifestOrderLookupService interface {
	// Get resolves one order by internal identifier.
	Get(ctx context.Context, id string) (*ordersdomain.Order, error)
}

// shippingBatchManifestProductLookupService defines product lookup behavior required for SKU variation labels.
type shippingBatchManifestProductLookupService interface {
	// GetBySKU retrieves a product by product-level or variant-level SKU.
	GetBySKU(ctx context.Context, sku string) (*productdomain.Product, error)
}

// shippingBatchManifestVariationLookupService defines variation lookup behavior required for SKU variation labels.
type shippingBatchManifestVariationLookupService interface {
	// Get retrieves a variation by identifier.
	Get(ctx context.Context, id string) (*variationdomain.Variation, error)
}

// shippingBatchManifestOrderSummaryAdapter adapts orders lookup behavior for batch manifest summary rendering.
type shippingBatchManifestOrderSummaryAdapter struct {
	// orders defines order lookup dependencies.
	orders shippingBatchManifestOrderLookupService
	// products defines product lookup dependencies.
	products shippingBatchManifestProductLookupService
	// variations defines product variation lookup dependencies.
	variations shippingBatchManifestVariationLookupService
}

// ResolveBatchManifestOrderSummary resolves one batch manifest order summary by order id.
func (a shippingBatchManifestOrderSummaryAdapter) ResolveBatchManifestOrderSummary(ctx context.Context, orderID string) (*dispatchservice.BatchManifestOrderSummary, error) {
	if a.orders == nil {
		return nil, nil
	}
	order, err := a.orders.Get(ctx, strings.TrimSpace(orderID))
	if err != nil || order == nil {
		return nil, err
	}
	return &dispatchservice.BatchManifestOrderSummary{
		OrderNumber: firstNonEmpty(strings.TrimSpace(order.Identifier), strings.TrimSpace(order.ID)),
		Items:       a.itemLabels(ctx, order.Items),
	}, nil
}

// ResolveRotulusOrderSummary resolves one rotulus order summary by order id.
func (a shippingBatchManifestOrderSummaryAdapter) ResolveRotulusOrderSummary(ctx context.Context, orderID string) (*markservice.RotulusOrderSummary, error) {
	if a.orders == nil {
		return nil, nil
	}
	order, err := a.orders.Get(ctx, strings.TrimSpace(orderID))
	if err != nil || order == nil {
		return nil, err
	}

	return &markservice.RotulusOrderSummary{
		Items: a.itemLabels(ctx, order.Items),
	}, nil
}

// itemLabels resolves compact item labels from order item rows and SKU-linked variation labels.
func (a shippingBatchManifestOrderSummaryAdapter) itemLabels(ctx context.Context, items []ordersdomain.Item) []string {
	labels := make([]string, 0, len(items))
	for _, item := range items {
		label := strings.TrimSpace(firstNonEmpty(strings.TrimSpace(item.AlternateName), strings.TrimSpace(item.SKU)))
		if label == "" {
			continue
		}
		label = a.enrichItemLabelWithSKUVariations(ctx, item.SKU, label)
		labels = append(labels, formatShippingBatchManifestItemLabel(item.Quantity, label))
	}
	return labels
}

// enrichItemLabelWithSKUVariations appends variation labels linked to one variant SKU.
func (a shippingBatchManifestOrderSummaryAdapter) enrichItemLabelWithSKUVariations(ctx context.Context, sku string, label string) string {
	if a.products == nil || a.variations == nil || strings.TrimSpace(sku) == "" {
		return strings.TrimSpace(label)
	}
	product, err := a.products.GetBySKU(ctx, strings.TrimSpace(sku))
	if err != nil || product == nil {
		return strings.TrimSpace(label)
	}
	variationIDs := matchingVariantVariationIDs(*product, sku)
	if len(variationIDs) == 0 {
		return strings.TrimSpace(label)
	}
	variationLabels := make([]string, 0, len(variationIDs))
	seen := map[string]struct{}{}
	for _, variationID := range variationIDs {
		variation, variationErr := a.variations.Get(ctx, variationID)
		if variationErr != nil || variation == nil {
			continue
		}
		variationLabel := strings.TrimSpace(variation.Name)
		if variationLabel == "" || containsLabelToken(label, variationLabel) {
			continue
		}
		normalized := strings.ToLower(variationLabel)
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		variationLabels = append(variationLabels, strings.ToUpper(variationLabel))
	}
	if len(variationLabels) == 0 {
		return strings.TrimSpace(label)
	}
	return strings.TrimSpace(label + " " + strings.Join(variationLabels, " "))
}

// matchingVariantVariationIDs resolves variation identifiers for the variant whose SKU matches the order item SKU.
func matchingVariantVariationIDs(product productdomain.Product, sku string) []string {
	trimmedSKU := strings.TrimSpace(sku)
	if trimmedSKU == "" {
		return nil
	}
	for _, variant := range product.Variants {
		if strings.EqualFold(strings.TrimSpace(variant.SKU), trimmedSKU) {
			return append([]string(nil), variant.VariationIDs...)
		}
	}
	return nil
}

// containsLabelToken reports whether a label already contains a variation label.
func containsLabelToken(label string, token string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(label)), strings.ToLower(strings.TrimSpace(token)))
}

// shippingBatchManifestItemLabels resolves compact item labels from order item rows.
func shippingBatchManifestItemLabels(items []ordersdomain.Item) []string {
	return (shippingBatchManifestOrderSummaryAdapter{}).itemLabels(context.Background(), items)
}

// formatShippingBatchManifestItemLabel resolves one PDF-safe product label with quantity prefix.
func formatShippingBatchManifestItemLabel(quantity int, label string) string {
	trimmed := strings.TrimSpace(label)
	if trimmed == "" {
		return ""
	}
	if quantity <= 0 {
		quantity = 1
	}

	return "X" + strconv.Itoa(quantity) + " " + trimmed
}
