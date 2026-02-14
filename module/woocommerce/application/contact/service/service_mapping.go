package service

import (
	"strconv"
	"strings"
	"time"

	"mannaiah/module/woocommerce/port"
)

const (
	syncMetadataSourceKey        = "integration.source"
	syncMetadataSourceValue      = "woocommerce"
	syncMetadataOldestOrderIDKey = "integration.woocommerce.oldest_order_id"
	syncMetadataOldestOrderAtKey = "integration.woocommerce.oldest_order_created_at"
)

// mapOrderToCommand maps WooCommerce orders into contact upsert command values.
func mapOrderToCommand(order port.WooOrder) (port.ContactSyncCommand, bool) {
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

	documentNumber := mapDocumentNumber(order.Metadata)
	documentType := ""
	if documentNumber != "" {
		documentType = "CC"
	}

	legalName := ""
	if !hasPersonalName {
		legalName = company
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
		Metadata:       buildSyncMetadata(order.ID, createdAt),
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

// resolveCreatedAt resolves non-zero source timestamps to UTC pointers.
func resolveCreatedAt(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}

	resolved := value.UTC()
	return &resolved
}

// buildSyncMetadata resolves sync metadata values stored on synchronized contacts.
func buildSyncMetadata(orderID int, createdAt *time.Time) map[string]string {
	metadata := map[string]string{
		syncMetadataSourceKey: syncMetadataSourceValue,
	}
	if orderID > 0 {
		metadata[syncMetadataOldestOrderIDKey] = strconv.Itoa(orderID)
	}
	if createdAt != nil {
		metadata[syncMetadataOldestOrderAtKey] = createdAt.UTC().Format(time.RFC3339)
	}

	return metadata
}
