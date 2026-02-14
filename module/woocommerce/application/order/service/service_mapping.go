package service

import (
	"strconv"
	"strings"
	"time"

	"mannaiah/module/woocommerce/port"
)

const (
	syncMetadataSourceKey   = "integration.source"
	syncMetadataSourceValue = "woocommerce"
	syncMetadataOrderIDKey  = "integration.woocommerce.order_id"
	syncMetadataStatusKey   = "integration.woocommerce.status"
)

// mapOrderToCommand maps WooCommerce orders into order upsert command values.
func mapOrderToCommand(order port.WooOrder) (port.OrderSyncCommand, bool) {
	contactCommand, ok := mapOrderToContactSyncCommand(order)
	if !ok {
		return port.OrderSyncCommand{}, false
	}

	identifier := strings.TrimSpace(strconv.Itoa(order.ID))
	if order.ID <= 0 {
		identifier = strings.TrimSpace(order.Metadata["integration.woocommerce.order_id"])
	}
	if identifier == "" {
		return port.OrderSyncCommand{}, false
	}

	items := mapOrderItems(order.Items)
	if len(items) == 0 {
		return port.OrderSyncCommand{}, false
	}

	createdAt := resolveCreatedAt(order.CreatedAt)
	command := port.OrderSyncCommand{
		Identifier:      identifier,
		Realm:           "woocommerce",
		Status:          strings.TrimSpace(strings.ToLower(order.Status)),
		CreatedAt:       createdAt,
		Contact:         contactCommand,
		Items:           items,
		ShippingAddress: mapShippingAddress(order),
		Metadata:        mergeMetadata(buildOrderMetadata(order), order.Metadata),
		Comments:        mapOrderComments(order),
	}

	return command, true
}

// mapOrderToContactSyncCommand maps WooCommerce order values to contact sync commands.
func mapOrderToContactSyncCommand(order port.WooOrder) (port.ContactSyncCommand, bool) {
	email := strings.ToLower(strings.TrimSpace(order.BillingEmail))
	if email == "" {
		return port.ContactSyncCommand{}, false
	}

	firstName := strings.TrimSpace(order.BillingFirstName)
	lastName := strings.TrimSpace(order.BillingLastName)
	company := strings.TrimSpace(order.BillingCompany)
	hasPersonalName := firstName != "" && lastName != ""
	if !hasPersonalName && company == "" {
		return port.ContactSyncCommand{}, false
	}

	legalName := ""
	if !hasPersonalName {
		legalName = company
	}

	documentNumber := strings.TrimSpace(order.Metadata["_billing_document"])
	documentType := ""
	if documentNumber != "" {
		documentType = "CC"
	}
	createdAt := resolveCreatedAt(order.CreatedAt)

	return port.ContactSyncCommand{
		Email:          email,
		FirstName:      firstName,
		LastName:       lastName,
		LegalName:      legalName,
		Phone:          normalizePhone(order.BillingPhone),
		Address:        strings.TrimSpace(order.BillingAddress1),
		AddressExtra:   strings.TrimSpace(order.BillingAddress2),
		CityCode:       strings.TrimSpace(order.BillingCity),
		DocumentType:   documentType,
		DocumentNumber: documentNumber,
		CreatedAt:      createdAt,
		Metadata:       buildContactMetadata(order, createdAt),
	}, true
}

// mapOrderItems maps WooCommerce order items to sync command items.
func mapOrderItems(items []port.WooOrderItem) []port.OrderSyncItem {
	result := make([]port.OrderSyncItem, 0, len(items))
	for _, item := range items {
		sku := strings.TrimSpace(item.SKU)
		if sku == "" {
			continue
		}
		metadata := map[string]string{}
		for key, value := range item.Metadata {
			trimmedKey := strings.TrimSpace(key)
			if trimmedKey == "" {
				continue
			}
			metadata[trimmedKey] = strings.TrimSpace(value)
		}
		result = append(result, port.OrderSyncItem{
			SKU:      sku,
			Name:     strings.TrimSpace(item.Name),
			Quantity: item.Quantity,
			Metadata: metadata,
		})
	}

	return result
}

// mapShippingAddress maps WooCommerce shipping values and drops billing-equivalent values.
func mapShippingAddress(order port.WooOrder) *port.OrderSyncShippingAddress {
	shipping := port.OrderSyncShippingAddress{
		Address:  strings.TrimSpace(order.ShippingAddressLine1),
		Address2: strings.TrimSpace(order.ShippingAddressLine2),
		Phone:    normalizePhone(order.ShippingPhone),
		CityCode: strings.TrimSpace(order.ShippingCityCode),
	}
	billing := port.OrderSyncShippingAddress{
		Address:  strings.TrimSpace(order.BillingAddress1),
		Address2: strings.TrimSpace(order.BillingAddress2),
		Phone:    normalizePhone(order.BillingPhone),
		CityCode: strings.TrimSpace(order.BillingCity),
	}

	if shipping.Address == "" && shipping.Address2 == "" && shipping.Phone == "" && shipping.CityCode == "" {
		return nil
	}
	if shipping == billing {
		return nil
	}

	return &shipping
}

// mapOrderComments maps WooCommerce order notes into comment command values.
func mapOrderComments(order port.WooOrder) []port.OrderSyncComment {
	comments := make([]port.OrderSyncComment, 0, len(order.Comments))
	for _, comment := range order.Comments {
		author := strings.TrimSpace(comment.Author)
		description := strings.TrimSpace(comment.Description)
		if description == "" {
			continue
		}
		if author == "" {
			author = "system"
		}
		comments = append(comments, port.OrderSyncComment{
			Author:      author,
			Description: description,
			OccurredAt:  comment.OccurredAt.UTC(),
		})
	}

	return comments
}

// buildOrderMetadata resolves sync metadata values stored on synchronized orders.
func buildOrderMetadata(order port.WooOrder) map[string]string {
	metadata := map[string]string{
		syncMetadataSourceKey: syncMetadataSourceValue,
	}
	if order.ID > 0 {
		metadata[syncMetadataOrderIDKey] = strconv.Itoa(order.ID)
	}
	if status := strings.TrimSpace(strings.ToLower(order.Status)); status != "" {
		metadata[syncMetadataStatusKey] = status
	}

	return metadata
}

// buildContactMetadata resolves sync metadata values stored on synchronized contacts.
func buildContactMetadata(order port.WooOrder, createdAt *time.Time) map[string]string {
	metadata := map[string]string{
		"integration.source": "woocommerce",
	}
	if order.ID > 0 {
		metadata["integration.woocommerce.oldest_order_id"] = strconv.Itoa(order.ID)
	}
	if createdAt != nil {
		metadata["integration.woocommerce.oldest_order_created_at"] = createdAt.UTC().Format(time.RFC3339)
	}

	return metadata
}

// normalizePhone normalizes WooCommerce phone values to +57-prefixed values.
func normalizePhone(value string) string {
	normalized := strings.ReplaceAll(strings.TrimSpace(value), " ", "")
	normalized = strings.ReplaceAll(normalized, "+", "")
	normalized = strings.TrimPrefix(normalized, "57")
	if normalized == "" {
		return ""
	}

	return "+57" + normalized
}

// resolveCreatedAt resolves non-zero source timestamps to UTC pointers.
func resolveCreatedAt(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}

	resolved := value.UTC()
	return &resolved
}

// mergeMetadata merges preferred metadata values over fallback values.
func mergeMetadata(left map[string]string, right map[string]string) map[string]string {
	result := map[string]string{}
	for key, value := range right {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		result[trimmedKey] = strings.TrimSpace(value)
	}
	for key, value := range left {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		result[trimmedKey] = strings.TrimSpace(value)
	}
	if len(result) == 0 {
		return nil
	}

	return result
}
