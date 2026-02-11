package contact

import (
	"strings"

	"mannaiah/module/woocommerce/port"
)

// mapOrderToCommand maps WooCommerce orders into contact upsert command values.
func mapOrderToCommand(order port.WooOrder) (port.ContactSyncCommand, bool) {
	email := strings.ToLower(strings.TrimSpace(order.BillingEmail))
	if email == "" {
		return port.ContactSyncCommand{}, false
	}

	firstName := strings.TrimSpace(order.BillingFirstName)
	lastName := strings.TrimSpace(order.BillingLastName)
	if firstName == "" || lastName == "" {
		return port.ContactSyncCommand{}, false
	}

	documentNumber := mapDocumentNumber(order.Metadata)
	documentType := ""
	if documentNumber != "" {
		documentType = "CC"
	}

	return port.ContactSyncCommand{
		Email:          email,
		FirstName:      firstName,
		LastName:       lastName,
		Phone:          normalizePhone(order.BillingPhone),
		Address:        strings.TrimSpace(order.BillingAddress1),
		AddressExtra:   strings.TrimSpace(order.BillingAddress2),
		CityCode:       strings.TrimSpace(order.BillingCity),
		DocumentType:   documentType,
		DocumentNumber: documentNumber,
	}, true
}

// mapDocumentNumber resolves WooCommerce billing document metadata values.
func mapDocumentNumber(metadata map[string]string) string {
	if len(metadata) == 0 {
		return ""
	}

	return strings.TrimSpace(metadata[billingDocumentMetaKey])
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
