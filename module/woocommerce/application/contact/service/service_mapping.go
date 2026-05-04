package service

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"mannaiah/module/woocommerce/internal/citycode"
	"mannaiah/module/woocommerce/port"
)

const (
	syncMetadataSourceKey                 = "integration.source"
	syncMetadataSourceValue               = "woocommerce"
	syncMetadataOldestOrderIDKey          = "integration.woocommerce.oldest_order_id"
	syncMetadataOldestOrderAtKey          = "integration.woocommerce.oldest_order_created_at"
	checkerMetadataPrefix                 = "flock_checker_"
	checkerMetadataAcceptedAtSuffix       = "_accepted_at"
	checkerMetadataAcceptedAtUTCSuffix    = "_accepted_at_utc"
	checkerMetadataRejectedAtSuffix       = "_rejected_at"
	checkerMetadataRejectedAtUTCSuffix    = "_rejected_at_utc"
	checkerMetadataAcceptedAtLocalLayout  = "2006-01-02 15:04:05"
	checkerMetadataAcceptedAtLocalZone    = "America/Bogota"
	checkerMetadataAcceptedAtFixedZone    = "UTC-05"
	checkerMetadataAcceptedAtFixedOffset  = -5 * 60 * 60
	checkerMetadataAcceptedValueConfirmed = "yes"
	checkerMetadataRejectedValueConfirmed = "no"
	circleOptInMetadataKey                = "flock_checker_circle_optin"
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
		CityCode:       citycode.Resolve(order.BillingCity),
		DocumentType:   documentType,
		DocumentNumber: documentNumber,
		CreatedAt:      createdAt,
		Metadata:       buildSyncMetadata(order.ID, createdAt, order.Metadata),
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
func buildSyncMetadata(orderID int, createdAt *time.Time, orderMetadata map[string]string) map[string]string {
	metadata := map[string]string{
		syncMetadataSourceKey: syncMetadataSourceValue,
	}
	if orderID > 0 {
		metadata[syncMetadataOldestOrderIDKey] = strconv.Itoa(orderID)
	}
	if createdAt != nil {
		metadata[syncMetadataOldestOrderAtKey] = createdAt.UTC().Format(time.RFC3339)
	}
	mergeCheckerMetadata(metadata, orderMetadata, createdAt)

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

		rejectedAtKey := baseKey + checkerMetadataRejectedAtSuffix
		rejectedAtUTCKey := baseKey + checkerMetadataRejectedAtUTCSuffix
		rejectedAt := strings.TrimSpace(source[rejectedAtKey])
		rejectedAtUTC := strings.TrimSpace(source[rejectedAtUTCKey])

		if decision == checkerMetadataAcceptedValueConfirmed {
			fallbackAcceptedAt, fallbackAcceptedAtUTC := checkerAcceptedAtFallback(createdAt)
			if acceptedAt == "" {
				acceptedAt = fallbackAcceptedAt
			}
			if acceptedAtUTC == "" {
				acceptedAtUTC = fallbackAcceptedAtUTC
			}
			delete(target, rejectedAtKey)
			delete(target, rejectedAtUTCKey)
		} else if decision == checkerMetadataRejectedValueConfirmed && baseKey == circleOptInMetadataKey {
			fallbackAcceptedAt, fallbackAcceptedAtUTC := checkerAcceptedAtFallback(createdAt)
			if rejectedAt == "" {
				if acceptedAt != "" {
					rejectedAt = acceptedAt
				} else {
					rejectedAt = fallbackAcceptedAt
				}
			}
			if rejectedAtUTC == "" {
				if acceptedAtUTC != "" {
					rejectedAtUTC = acceptedAtUTC
				} else {
					rejectedAtUTC = fallbackAcceptedAtUTC
				}
			}
			acceptedAt = ""
			acceptedAtUTC = ""
		}

		if acceptedAt != "" {
			target[acceptedAtKey] = acceptedAt
		} else if baseKey == circleOptInMetadataKey {
			delete(target, acceptedAtKey)
		}
		if acceptedAtUTC != "" {
			target[acceptedAtUTCKey] = acceptedAtUTC
		} else if baseKey == circleOptInMetadataKey {
			delete(target, acceptedAtUTCKey)
		}
		if rejectedAt != "" {
			target[rejectedAtKey] = rejectedAt
		}
		if rejectedAtUTC != "" {
			target[rejectedAtUTCKey] = rejectedAtUTC
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
		if strings.HasSuffix(trimmedKey, checkerMetadataAcceptedAtSuffix) ||
			strings.HasSuffix(trimmedKey, checkerMetadataAcceptedAtUTCSuffix) ||
			strings.HasSuffix(trimmedKey, checkerMetadataRejectedAtSuffix) ||
			strings.HasSuffix(trimmedKey, checkerMetadataRejectedAtUTCSuffix) {
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

// normalizeCircleOptInMetadata normalizes circle opt-in metadata transitions in merged metadata maps.
func normalizeCircleOptInMetadata(metadata map[string]string) {
	if len(metadata) == 0 {
		return
	}

	decision := strings.ToLower(strings.TrimSpace(metadata[circleOptInMetadataKey]))
	acceptedAtKey := circleOptInMetadataKey + checkerMetadataAcceptedAtSuffix
	acceptedAtUTCKey := circleOptInMetadataKey + checkerMetadataAcceptedAtUTCSuffix
	rejectedAtKey := circleOptInMetadataKey + checkerMetadataRejectedAtSuffix
	rejectedAtUTCKey := circleOptInMetadataKey + checkerMetadataRejectedAtUTCSuffix

	switch decision {
	case checkerMetadataAcceptedValueConfirmed:
		delete(metadata, rejectedAtKey)
		delete(metadata, rejectedAtUTCKey)
	case checkerMetadataRejectedValueConfirmed:
		fallbackRejectedAt := strings.TrimSpace(metadata[acceptedAtKey])
		fallbackRejectedAtUTC := strings.TrimSpace(metadata[acceptedAtUTCKey])
		delete(metadata, acceptedAtKey)
		delete(metadata, acceptedAtUTCKey)
		if strings.TrimSpace(metadata[rejectedAtKey]) == "" && fallbackRejectedAt != "" {
			metadata[rejectedAtKey] = fallbackRejectedAt
		}
		if strings.TrimSpace(metadata[rejectedAtUTCKey]) == "" && fallbackRejectedAtUTC != "" {
			metadata[rejectedAtUTCKey] = fallbackRejectedAtUTC
		}
	}
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
