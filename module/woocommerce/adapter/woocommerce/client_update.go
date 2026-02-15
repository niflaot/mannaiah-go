package woocommerce

import (
	"context"
	"fmt"
	"strings"

	wc "github.com/jmolboy/woocommerce-go"
	wcentity "github.com/jmolboy/woocommerce-go/entity"
	"mannaiah/module/woocommerce/port"
)

// UpdateOrderFromMainstream updates WooCommerce order mutable values from mainstream-origin payloads.
func (c *Client) UpdateOrderFromMainstream(ctx context.Context, command port.MainstreamOrderUpdateCommand) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	orderID, err := parseWooOrderID(command.Identifier)
	if err != nil {
		return err
	}

	request, err := c.toUpdateOrderRequest(ctx, command)
	if err != nil {
		return err
	}
	if _, err := c.client.Services.Order.Update(orderID, request); err != nil {
		return fmt.Errorf("update woocommerce order %d: %w", orderID, err)
	}

	return nil
}

// toUpdateOrderRequest maps mainstream order update commands to WooCommerce SDK requests.
func (c *Client) toUpdateOrderRequest(ctx context.Context, command port.MainstreamOrderUpdateCommand) (wc.UpdateOrderRequest, error) {
	lineItems, feeLines, err := c.resolveOrderItemsForUpdate(ctx, command.Items)
	if err != nil {
		return wc.UpdateOrderRequest{}, err
	}

	request := wc.UpdateOrderRequest{
		LineItems:     lineItems,
		ShippingLines: mapShippingLinesForUpdate(command.ShippingCharges),
		FeeLines:      feeLines,
	}

	if command.ShippingAddress != nil {
		shipping := mapShippingAddressForUpdate(*command.ShippingAddress)
		request.Shipping = &shipping

		billing := mapBillingAddressForUpdate(*command.ShippingAddress)
		request.Billing = &billing
	}

	return request, nil
}

// resolveOrderItemsForUpdate resolves line-item and fee-line payload values.
func (c *Client) resolveOrderItemsForUpdate(ctx context.Context, items []port.OrderSyncItem) ([]wcentity.LineItem, []wcentity.FeeLine, error) {
	if len(items) == 0 {
		return nil, nil, nil
	}

	lineItems := make([]wcentity.LineItem, 0, len(items))
	feeLines := make([]wcentity.FeeLine, 0, len(items))
	productIDsBySKU := map[string]int{}

	for _, item := range items {
		sku := strings.TrimSpace(item.SKU)
		name := strings.TrimSpace(item.Name)
		quantity := item.Quantity
		if quantity <= 0 {
			quantity = 1
		}
		value := item.Value
		if value < 0 {
			value = 0
		}

		productID := 0
		if sku != "" {
			if resolved, ok := productIDsBySKU[sku]; ok {
				productID = resolved
			} else {
				resolved, resolveErr := c.resolveWooProductIDBySKU(ctx, sku)
				if resolveErr != nil {
					return nil, nil, resolveErr
				}
				productIDsBySKU[sku] = resolved
				productID = resolved
			}
		}
		if productID > 0 {
			lineItems = append(lineItems, wcentity.LineItem{
				ProductId: productID,
				Quantity:  quantity,
				Total:     value,
				SKU:       sku,
			})
			continue
		}

		fallbackName := name
		if fallbackName == "" {
			fallbackName = sku
		}
		if fallbackName == "" {
			continue
		}

		feeLines = append(feeLines, wcentity.FeeLine{
			Name:  fallbackName,
			Total: value,
		})
	}

	return lineItems, feeLines, nil
}

// resolveWooProductIDBySKU resolves WooCommerce product IDs by SKU values.
func (c *Client) resolveWooProductIDBySKU(ctx context.Context, sku string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	resolvedID, err := c.resolveWooProductIDBySKURaw(ctx, strings.TrimSpace(sku))
	if err != nil {
		return 0, fmt.Errorf("resolve woocommerce product by sku %q: %w", sku, err)
	}

	return resolvedID, nil
}

// mapShippingLinesForUpdate maps shipping charge payload values to WooCommerce shipping-line values.
func mapShippingLinesForUpdate(values []port.OrderSyncShippingCharge) []wcentity.ShippingLine {
	if len(values) == 0 {
		return nil
	}

	lines := make([]wcentity.ShippingLine, 0, len(values))
	for _, value := range values {
		methodID := strings.TrimSpace(value.MethodID)
		methodTitle := strings.TrimSpace(value.MethodTitle)
		price := value.Price
		if price < 0 {
			price = 0
		}
		if methodID == "" && methodTitle == "" && price == 0 {
			continue
		}
		lines = append(lines, wcentity.ShippingLine{
			MethodId:    methodID,
			MethodTitle: methodTitle,
			Total:       price,
		})
	}
	if len(lines) == 0 {
		return nil
	}

	return lines
}

// mapShippingAddressForUpdate maps shipping-address payload values to WooCommerce shipping values.
func mapShippingAddressForUpdate(value port.OrderSyncShippingAddress) wcentity.Shipping {
	return wcentity.Shipping{
		Address1: strings.TrimSpace(value.Address),
		Address2: strings.TrimSpace(value.Address2),
		City:     strings.TrimSpace(value.CityCode),
	}
}

// mapBillingAddressForUpdate maps shipping-address payload values to WooCommerce billing values.
func mapBillingAddressForUpdate(value port.OrderSyncShippingAddress) wcentity.Billing {
	return wcentity.Billing{
		Address1: strings.TrimSpace(value.Address),
		Address2: strings.TrimSpace(value.Address2),
		City:     strings.TrimSpace(value.CityCode),
		Phone:    strings.TrimSpace(value.Phone),
	}
}
