package service

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"mannaiah/module/woocommerce/port"
)

const (
	syncMetadataSourceKey                 = "integration.source"
	syncMetadataSourceValue               = "woocommerce"
	syncCommentAuthor                     = "system"
	checkerMetadataPrefix                 = "flock_checker_"
	checkerMetadataAcceptedAtSuffix       = "_accepted_at"
	checkerMetadataAcceptedAtUTCSuffix    = "_accepted_at_utc"
	checkerMetadataAcceptedAtLocalLayout  = "2006-01-02 15:04:05"
	checkerMetadataAcceptedAtLocalZone    = "America/Bogota"
	checkerMetadataAcceptedAtFixedZone    = "UTC-05"
	checkerMetadataAcceptedAtFixedOffset  = -5 * 60 * 60
	checkerMetadataAcceptedValueConfirmed = "yes"
)

// mapOrderSkipReason defines reason codes for skipped order rows.
type mapOrderSkipReason string

const (
	// skipReasonMissingContactEmail is used when billing email values are unavailable.
	skipReasonMissingContactEmail mapOrderSkipReason = "missing_contact_email"
	// skipReasonMissingContactName is used when neither personal names nor company names are available.
	skipReasonMissingContactName mapOrderSkipReason = "missing_contact_name"
	// skipReasonMissingIdentifier is used when order identifiers cannot be resolved.
	skipReasonMissingIdentifier mapOrderSkipReason = "missing_identifier"
	// skipReasonMissingSupportedItems is used when no supported items can be mapped.
	skipReasonMissingSupportedItems mapOrderSkipReason = "missing_supported_items"
)

// mapOrderToCommand maps WooCommerce orders into order upsert command values.
func mapOrderToCommand(order port.WooOrder) (port.OrderSyncCommand, bool, mapOrderSkipReason) {
	contactCommand, ok, reason := mapOrderToContactSyncCommand(order)
	if !ok {
		return port.OrderSyncCommand{}, false, reason
	}

	identifier := strings.TrimSpace(strconv.Itoa(order.ID))
	if order.ID <= 0 {
		identifier = strings.TrimSpace(order.Metadata["integration.woocommerce.order_id"])
	}
	if identifier == "" {
		return port.OrderSyncCommand{}, false, skipReasonMissingIdentifier
	}

	items := mapOrderItems(order.Items)
	if len(items) == 0 {
		return port.OrderSyncCommand{}, false, skipReasonMissingSupportedItems
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
		ShippingCharges: mapShippingCharges(order.ShippingCharges),
		Metadata:        nil,
		Comments:        mapOrderComments(order),
	}

	return command, true, ""
}

// mapOrderToContactSyncCommand maps WooCommerce order values to contact sync commands.
func mapOrderToContactSyncCommand(order port.WooOrder) (port.ContactSyncCommand, bool, mapOrderSkipReason) {
	email := strings.ToLower(strings.TrimSpace(order.BillingEmail))
	if email == "" {
		return port.ContactSyncCommand{}, false, skipReasonMissingContactEmail
	}

	firstName := strings.TrimSpace(order.BillingFirstName)
	lastName := strings.TrimSpace(order.BillingLastName)
	company := strings.TrimSpace(order.BillingCompany)
	hasPersonalName := firstName != "" && lastName != ""
	if !hasPersonalName && company == "" {
		return port.ContactSyncCommand{}, false, skipReasonMissingContactName
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
	}, true, ""
}

// mapOrderItems maps WooCommerce order items to sync command items.
func mapOrderItems(items []port.WooOrderItem) []port.OrderSyncItem {
	result := make([]port.OrderSyncItem, 0, len(items))
	for _, item := range items {
		sku := strings.TrimSpace(item.SKU)
		name := strings.TrimSpace(item.Name)
		if sku == "" && name == "" {
			continue
		}
		quantity := item.Quantity
		if quantity <= 0 {
			quantity = 1
		}

		result = append(result, port.OrderSyncItem{
			SKU:      sku,
			Name:     name,
			Quantity: quantity,
			Value:    item.Value,
		})
	}

	return result
}

// mapShippingAddress maps WooCommerce shipping values and falls back to billing snapshots.
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
		if billing.Address == "" && billing.Address2 == "" && billing.Phone == "" && billing.CityCode == "" {
			return nil
		}
		return &billing
	}
	if shipping == billing {
		return &billing
	}

	return &shipping
}

// mapShippingCharges maps WooCommerce shipping charge values to sync payload values.
func mapShippingCharges(values []port.WooOrderShippingCharge) []port.OrderSyncShippingCharge {
	if len(values) == 0 {
		return nil
	}

	rows := make([]port.OrderSyncShippingCharge, 0, len(values))
	for _, value := range values {
		methodID := strings.TrimSpace(value.MethodID)
		methodTitle := strings.TrimSpace(value.MethodTitle)
		if methodID == "" && methodTitle == "" && value.Price == 0 {
			continue
		}
		rows = append(rows, port.OrderSyncShippingCharge{
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

// mapOrderComments maps WooCommerce order notes into comment command values.
func mapOrderComments(order port.WooOrder) []port.OrderSyncComment {
	comments := make([]port.OrderSyncComment, 0, len(order.Comments))
	for _, comment := range order.Comments {
		author := strings.TrimSpace(comment.Author)
		text := strings.TrimSpace(comment.Description)
		if text == "" {
			continue
		}
		if author == "" {
			author = syncCommentAuthor
		}
		comments = append(comments, port.OrderSyncComment{
			Author:     author,
			Comment:    text,
			Internal:   comment.Internal,
			OccurredAt: comment.OccurredAt.UTC(),
		})
	}

	return comments
}

// buildContactMetadata resolves sync metadata values stored on synchronized contacts.
func buildContactMetadata(order port.WooOrder, createdAt *time.Time) map[string]string {
	metadata := map[string]string{
		syncMetadataSourceKey: syncMetadataSourceValue,
	}
	if order.ID > 0 {
		metadata["integration.woocommerce.oldest_order_id"] = strconv.Itoa(order.ID)
	}
	if createdAt != nil {
		metadata["integration.woocommerce.oldest_order_created_at"] = createdAt.UTC().Format(time.RFC3339)
	}
	mergeCheckerMetadata(metadata, order.Metadata, createdAt)

	return metadata
}

// mergeCheckerMetadata maps Woo metadata checker key groups into contact metadata payload values.
func mergeCheckerMetadata(target map[string]string, source map[string]string, createdAt *time.Time) {
	if len(source) == 0 {
		return
	}

	baseKeys := checkerMetadataKeys(source)
	for _, baseKey := range baseKeys {
		decision := normalizeCheckerDecision(source[baseKey])
		if decision == "" {
			continue
		}
		target[baseKey] = decision

		acceptedAtKey := baseKey + checkerMetadataAcceptedAtSuffix
		acceptedAtUTCKey := baseKey + checkerMetadataAcceptedAtUTCSuffix

		acceptedAt := strings.TrimSpace(source[acceptedAtKey])
		acceptedAtUTC := strings.TrimSpace(source[acceptedAtUTCKey])

		if decision == checkerMetadataAcceptedValueConfirmed {
			fallbackAcceptedAt, fallbackAcceptedAtUTC := checkerAcceptedAtFallback(createdAt)
			if acceptedAt == "" {
				acceptedAt = fallbackAcceptedAt
			}
			if acceptedAtUTC == "" {
				acceptedAtUTC = fallbackAcceptedAtUTC
			}
		}

		if acceptedAt != "" {
			target[acceptedAtKey] = acceptedAt
		}
		if acceptedAtUTC != "" {
			target[acceptedAtUTCKey] = acceptedAtUTC
		}
	}
}

// checkerMetadataKeys resolves checker decision-key metadata values from order metadata maps.
func checkerMetadataKeys(source map[string]string) []string {
	keys := make([]string, 0, len(source))
	for key := range source {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		if !strings.HasPrefix(trimmedKey, checkerMetadataPrefix) {
			continue
		}
		if strings.HasSuffix(trimmedKey, checkerMetadataAcceptedAtSuffix) || strings.HasSuffix(trimmedKey, checkerMetadataAcceptedAtUTCSuffix) {
			continue
		}

		keys = append(keys, trimmedKey)
	}
	if len(keys) == 0 {
		return nil
	}

	sort.Strings(keys)
	return keys
}

// normalizeCheckerDecision normalizes checker decision values to lowercase yes/no values when possible.
func normalizeCheckerDecision(value string) string {
	trimmed := strings.TrimSpace(value)
	if strings.EqualFold(trimmed, "yes") {
		return "yes"
	}
	if strings.EqualFold(trimmed, "no") {
		return "no"
	}

	return trimmed
}

// checkerAcceptedAtFallback resolves fallback checker accepted-at timestamps from source order creation timestamps.
func checkerAcceptedAtFallback(createdAt *time.Time) (string, string) {
	if createdAt == nil || createdAt.IsZero() {
		return "", ""
	}

	utc := createdAt.UTC()
	location, err := time.LoadLocation(checkerMetadataAcceptedAtLocalZone)
	if err != nil {
		location = time.FixedZone(checkerMetadataAcceptedAtFixedZone, checkerMetadataAcceptedAtFixedOffset)
	}

	return utc.In(location).Format(checkerMetadataAcceptedAtLocalLayout), utc.Format(time.RFC3339)
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
