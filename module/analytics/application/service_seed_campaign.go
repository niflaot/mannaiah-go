package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"mannaiah/module/analytics/port"
)

// campaignSeedRow holds a campaign delivery event row read from the transactional database during seed.
type campaignSeedRow struct {
	// ID is the delivery status history UUID.
	ID string `gorm:"column:id"`
	// ContactID is the linked contact UUID.
	ContactID string `gorm:"column:contact_id"`
	// IdempotencyKey encodes campaign ID and contact ID for deduplication.
	IdempotencyKey string `gorm:"column:idempotency_key"`
	// Status is the delivery status value.
	Status string `gorm:"column:status"`
	// OccurredAt is the status occurrence timestamp.
	OccurredAt time.Time `gorm:"column:occurred_at"`
}

// seedCampaignEvents reads campaign delivery events in batches and inserts them into the analytics store.
func (s *AnalyticsService) seedCampaignEvents(ctx context.Context, summary *SeedSummary) error {
	lastID := ""
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		rows := make([]campaignSeedRow, 0, seedBatchSize)
		query := s.db.WithContext(ctx).
			Table("email_delivery_status_history esh").
			Select("esh.id", "ed.contact_id", "ed.idempotency_key", "esh.status", "esh.occurred_at").
			Joins("INNER JOIN email_deliveries ed ON ed.id = esh.delivery_id")
		if lastID != "" {
			query = query.Where("esh.id > ?", lastID)
		}
		if err := query.Order("esh.id ASC").Limit(seedBatchSize).Scan(&rows).Error; err != nil {
			if looksLikeTableMissing(err) {
				return nil
			}
			return fmt.Errorf("seed campaign events batch: %w", err)
		}
		if len(rows) == 0 {
			break
		}

		payload := make([]port.CampaignEvent, 0, len(rows))
		for _, row := range rows {
			campaignID, fallbackContactID, ok := parseCampaignIdempotency(row.IdempotencyKey)
			if !ok {
				continue
			}
			contactID := strings.TrimSpace(row.ContactID)
			if contactID == "" {
				contactID = fallbackContactID
			}
			payload = append(payload, port.CampaignEvent{
				CampaignID:      campaignID,
				ContactID:       contactID,
				Channel:         "email",
				Status:          strings.TrimSpace(row.Status),
				TemplateVersion: 1,
				OccurredAt:      row.OccurredAt.UTC(),
			})
		}
		if err := s.store.InsertCampaignEvents(ctx, payload); err != nil {
			return fmt.Errorf("insert campaign events batch: %w", err)
		}
		summary.CampaignEvents += int64(len(payload))
		lastID = rows[len(rows)-1].ID
	}

	return nil
}

// parseCampaignIdempotency parses campaign ID and contact ID from an idempotency key.
func parseCampaignIdempotency(value string) (string, string, bool) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) < 2 {
		return "", "", false
	}

	campaignID := strings.TrimSpace(parts[0])
	contactID := strings.TrimSpace(parts[1])
	if campaignID == "" || contactID == "" {
		return "", "", false
	}

	return campaignID, contactID, true
}

// looksLikeTableMissing reports whether a database error indicates a missing table.
func looksLikeTableMissing(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))

	return strings.Contains(message, "doesn't exist") || strings.Contains(message, "no such table")
}
