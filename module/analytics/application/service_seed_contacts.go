package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"mannaiah/module/analytics/port"
)

// contactSeedRow holds a contact row read from the transactional database during seed.
type contactSeedRow struct {
	// ID is the contact UUID.
	ID string `gorm:"column:id"`
	// Email is the contact email address.
	Email string `gorm:"column:email"`
	// FirstName is the contact first name.
	FirstName string `gorm:"column:first_name"`
	// LastName is the contact last name.
	LastName string `gorm:"column:last_name"`
	// LegalName is the contact legal/company name.
	LegalName string `gorm:"column:legal_name"`
	// Phone is the contact phone number.
	Phone string `gorm:"column:phone"`
	// CityCode is the contact municipality code.
	CityCode string `gorm:"column:city_code"`
	// DocumentType is the contact document type.
	DocumentType string `gorm:"column:document_type"`
	// CreatedAt is the contact creation timestamp.
	CreatedAt *time.Time `gorm:"column:created_at"`
	// UpdatedAt is the contact last-updated timestamp.
	UpdatedAt *time.Time `gorm:"column:updated_at"`
}

// contactMetadataSeedRow holds a contact metadata row read during seed.
type contactMetadataSeedRow struct {
	// ContactID is the parent contact UUID.
	ContactID string `gorm:"column:contact_id"`
	// Key is the metadata key.
	Key string `gorm:"column:key"`
	// Value is the metadata value.
	Value string `gorm:"column:value"`
}

// seedContacts reads contacts in batches from the transactional database and upserts them into the analytics store.
func (s *AnalyticsService) seedContacts(ctx context.Context, summary *SeedSummary) error {
	lastID := ""
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		rows := make([]contactSeedRow, 0, seedBatchSize)
		query := s.db.WithContext(ctx).
			Table("contacts").
			Select("id", "email", "first_name", "last_name", "legal_name", "phone", "city_code", "document_type", "created_at", "updated_at").
			Where("deleted_at IS NULL")
		if lastID != "" {
			query = query.Where("id > ?", lastID)
		}
		if err := query.Order("id ASC").Limit(seedBatchSize).Scan(&rows).Error; err != nil {
			return fmt.Errorf("seed contacts batch: %w", err)
		}
		if len(rows) == 0 {
			break
		}

		metadataByContact := s.loadContactMetadata(ctx, rows)
		payload := buildContactSnapshotPayload(rows, metadataByContact)

		if err := s.store.UpsertContacts(ctx, payload); err != nil {
			return fmt.Errorf("upsert contacts snapshot batch: %w", err)
		}
		summary.Contacts += int64(len(payload))
		lastID = rows[len(rows)-1].ID
	}

	return nil
}

// loadContactMetadata fetches metadata rows for the given contact batch and indexes them by contact ID.
func (s *AnalyticsService) loadContactMetadata(ctx context.Context, rows []contactSeedRow) map[string]map[string]string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, strings.TrimSpace(row.ID))
	}

	metadataRows := make([]contactMetadataSeedRow, 0, len(rows)*2)
	_ = s.db.WithContext(ctx).
		Table("contact_metadata").
		Select("contact_id, `key`, value").
		Where("contact_id IN ?", ids).
		Scan(&metadataRows).Error

	result := map[string]map[string]string{}
	for _, row := range metadataRows {
		contactID := strings.TrimSpace(row.ContactID)
		if contactID == "" {
			continue
		}
		if _, exists := result[contactID]; !exists {
			result[contactID] = map[string]string{}
		}
		key := strings.TrimSpace(row.Key)
		if key == "" {
			continue
		}
		result[contactID][key] = row.Value
	}

	return result
}

// buildContactSnapshotPayload maps contact seed rows into analytics snapshot payloads.
func buildContactSnapshotPayload(rows []contactSeedRow, metadataByContact map[string]map[string]string) []port.ContactSnapshot {
	now := time.Now().UTC()
	payload := make([]port.ContactSnapshot, 0, len(rows))
	for _, row := range rows {
		createdAt := now
		updatedAt := now
		if row.CreatedAt != nil {
			createdAt = row.CreatedAt.UTC()
		}
		if row.UpdatedAt != nil {
			updatedAt = row.UpdatedAt.UTC()
		}
		if createdAt.IsZero() {
			createdAt = now
		}
		if updatedAt.IsZero() {
			updatedAt = createdAt
		}
		contactID := strings.TrimSpace(row.ID)
		payload = append(payload, port.ContactSnapshot{
			ContactID:    contactID,
			Email:        strings.TrimSpace(row.Email),
			FirstName:    strings.TrimSpace(row.FirstName),
			LastName:     strings.TrimSpace(row.LastName),
			LegalName:    strings.TrimSpace(row.LegalName),
			Phone:        strings.TrimSpace(row.Phone),
			CityCode:     strings.TrimSpace(row.CityCode),
			DocumentType: strings.TrimSpace(row.DocumentType),
			Metadata:     metadataByContact[contactID],
			CreatedAt:    createdAt,
			UpdatedAt:    updatedAt,
		})
	}

	return payload
}
