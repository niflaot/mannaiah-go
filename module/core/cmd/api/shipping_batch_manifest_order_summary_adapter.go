package main

import (
	"context"
	"strings"

	ordersdomain "mannaiah/module/orders/domain"
	dispatchservice "mannaiah/module/shipping/application/dispatch/service"
	markservice "mannaiah/module/shipping/application/mark/service"
)

// shippingBatchManifestOrderLookupService defines order lookup behavior required by batch manifest summary adapters.
type shippingBatchManifestOrderLookupService interface {
	// Get resolves one order by internal identifier.
	Get(ctx context.Context, id string) (*ordersdomain.Order, error)
}

// shippingBatchManifestOrderSummaryAdapter adapts orders lookup behavior for batch manifest summary rendering.
type shippingBatchManifestOrderSummaryAdapter struct {
	// orders defines order lookup dependencies.
	orders shippingBatchManifestOrderLookupService
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
		Items:       shippingBatchManifestItemLabels(order.Items),
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
		Items: shippingBatchManifestItemLabels(order.Items),
	}, nil
}

// shippingBatchManifestItemLabels resolves compact item labels from order item rows.
func shippingBatchManifestItemLabels(items []ordersdomain.Item) []string {
	labels := make([]string, 0, len(items))
	for _, item := range items {
		label := strings.TrimSpace(firstNonEmpty(strings.TrimSpace(item.AlternateName), strings.TrimSpace(item.SKU)))
		if label == "" {
			continue
		}
		labels = append(labels, label)
	}
	return labels
}
