package orders

import (
	"strings"

	ordersapplication "mannaiah/module/orders/application"
	ordersdomain "mannaiah/module/orders/domain"
	"mannaiah/module/woocommerce/port"
)

// toCreateCommand maps order sync command values to order creation payload values.
func toCreateCommand(
	command port.OrderSyncCommand,
	contactID string,
	realm string,
	status ordersdomain.Status,
) ordersapplication.CreateCommand {
	return ordersapplication.CreateCommand{
		Identifier:      strings.TrimSpace(command.Identifier),
		Realm:           strings.TrimSpace(realm),
		ContactID:       strings.TrimSpace(contactID),
		Items:           toCreateItems(command.Items),
		InitialStatus:   &status,
		Author:          syncStatusAuthor,
		Description:     syncStatusDescription,
		ShippingAddress: toShippingAddress(command.ShippingAddress),
		ShippingCharges: toShippingCharges(command.ShippingCharges),
		PaymentMethod:   strings.TrimSpace(command.PaymentMethod),
		Metadata:        normalizeMetadata(command.Metadata),
		CreatedAt:       command.CreatedAt,
		Source:          syncStatusAuthor,
	}
}

// toCreateItems maps sync item values to create command item values.
func toCreateItems(items []port.OrderSyncItem) []ordersapplication.CreateItemCommand {
	result := make([]ordersapplication.CreateItemCommand, 0, len(items))
	for _, item := range items {
		sku := strings.TrimSpace(item.SKU)
		alternateName := strings.TrimSpace(item.Name)
		if sku == "" && alternateName == "" {
			continue
		}

		quantity := item.Quantity
		if quantity <= 0 {
			quantity = 1
		}

		result = append(result, ordersapplication.CreateItemCommand{
			SKU:           sku,
			AlternateName: alternateName,
			Quantity:      quantity,
			Value:         item.Value,
		})
	}

	return result
}

// toShippingAddress maps optional sync shipping address values to create command shipping values.
func toShippingAddress(value *port.OrderSyncShippingAddress) *ordersapplication.ShippingAddressCommand {
	if value == nil {
		return nil
	}

	command := &ordersapplication.ShippingAddressCommand{
		Address:  strings.TrimSpace(value.Address),
		Address2: strings.TrimSpace(value.Address2),
		Phone:    strings.TrimSpace(value.Phone),
		CityCode: strings.TrimSpace(value.CityCode),
	}
	if command.Address == "" && command.Address2 == "" && command.Phone == "" && command.CityCode == "" {
		return nil
	}

	return command
}

// toShippingCharges maps sync shipping charge values to create command shipping-charge values.
func toShippingCharges(values []port.OrderSyncShippingCharge) []ordersapplication.ShippingChargeCommand {
	if len(values) == 0 {
		return nil
	}

	rows := make([]ordersapplication.ShippingChargeCommand, 0, len(values))
	for _, value := range values {
		methodID := strings.TrimSpace(value.MethodID)
		methodTitle := strings.TrimSpace(value.MethodTitle)
		if methodID == "" && methodTitle == "" && value.Price == 0 {
			continue
		}
		rows = append(rows, ordersapplication.ShippingChargeCommand{
			MethodID:    methodID,
			MethodTitle: methodTitle,
			Price:       value.Price,
		})
	}
	if len(rows) == 0 {
		return nil
	}

	return rows
}

// mapOrderStatus maps WooCommerce source status values to order-domain status values.
func mapOrderStatus(value string) ordersdomain.Status {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "cancelled", "canceled", "failed":
		return ordersdomain.StatusCancelled
	case "processing", "created":
		return ordersdomain.StatusCreated
	case "on-hold", "hold":
		return ordersdomain.StatusHold
	case "pending", "pending-payment":
		return ordersdomain.StatusPending
	case "completed", "complete":
		return ordersdomain.StatusCompleted
	default:
		return ordersdomain.StatusCreated
	}
}
