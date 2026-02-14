package store

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"gorm.io/gorm"
	"mannaiah/module/contacts/domain"
)

// contactMetadataRecord defines contact metadata persistence schema.
type contactMetadataRecord struct {
	// ID defines surrogate row identifiers.
	ID uint `gorm:"primaryKey"`
	// ContactID defines owning contact identifiers.
	ContactID string `gorm:"size:64;not null;index;uniqueIndex:idx_contacts_metadata_contact_key,priority:1"`
	// Key defines metadata keys.
	Key string `gorm:"size:128;not null;index;uniqueIndex:idx_contacts_metadata_contact_key,priority:2"`
	// Value defines metadata values.
	Value string `gorm:"type:text;not null"`
}

// TableName defines storage table name.
func (contactMetadataRecord) TableName() string {
	return "contact_metadata"
}

// replaceContactMetadata rewrites metadata rows for target contacts.
func replaceContactMetadata(tx *gorm.DB, contactID string, metadata map[string]string) error {
	trimmedContactID := strings.TrimSpace(contactID)
	if err := tx.Where("contact_id = ?", trimmedContactID).Delete(&contactMetadataRecord{}).Error; err != nil {
		return fmt.Errorf("delete contact metadata records: %w", err)
	}

	rows, err := toMetadataRecords(trimmedContactID, metadata)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	if err := tx.Create(&rows).Error; err != nil {
		return fmt.Errorf("create contact metadata records: %w", err)
	}

	return nil
}

// loadMetadataByContactIDs loads metadata rows grouped by contact identifiers.
func loadMetadataByContactIDs(ctx context.Context, db *gorm.DB, contactIDs []string) (map[string]map[string]string, error) {
	result := make(map[string]map[string]string, len(contactIDs))
	if len(contactIDs) == 0 {
		return result, nil
	}

	rows := make([]contactMetadataRecord, 0)
	if err := db.WithContext(ctx).Model(&contactMetadataRecord{}).Where("contact_id IN ?", contactIDs).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("load contact metadata records: %w", err)
	}

	for _, contactID := range contactIDs {
		result[contactID] = map[string]string{}
	}
	for _, row := range rows {
		group := result[row.ContactID]
		if group == nil {
			group = map[string]string{}
		}
		group[row.Key] = row.Value
		result[row.ContactID] = group
	}

	return result, nil
}

// toMetadataRecords maps metadata maps into deterministic persistence rows.
func toMetadataRecords(contactID string, metadata map[string]string) ([]contactMetadataRecord, error) {
	if metadata == nil {
		return nil, nil
	}

	normalized := make(map[string]string, len(metadata))
	for key, value := range metadata {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		trimmedValue := strings.TrimSpace(value)
		if len(trimmedKey) > 128 || len(trimmedValue) > 2048 {
			return nil, domain.ErrInvalidMetadata
		}
		normalized[trimmedKey] = trimmedValue
	}

	keys := make([]string, 0, len(normalized))
	for key := range normalized {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	rows := make([]contactMetadataRecord, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, contactMetadataRecord{
			ContactID: contactID,
			Key:       key,
			Value:     normalized[key],
		})
	}

	return rows, nil
}
