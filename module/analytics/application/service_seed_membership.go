package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"mannaiah/module/analytics/port"
)

// membershipSeedRow holds a membership stamp row read from the transactional database during seed.
type membershipSeedRow struct {
	// ID is the stamp UUID.
	ID string `gorm:"column:id"`
	// ContactID is the linked contact UUID.
	ContactID string `gorm:"column:contact_id"`
	// Channel is the membership channel.
	Channel string `gorm:"column:channel"`
	// Action is the membership action (opt_in / opt_out).
	Action string `gorm:"column:action"`
	// Source is the action source identifier.
	Source string `gorm:"column:source"`
	// OccurredAt is the action occurrence timestamp.
	OccurredAt time.Time `gorm:"column:occurred_at"`
	// CreatedAt is the stamp creation timestamp used for cursor pagination.
	CreatedAt time.Time `gorm:"column:created_at"`
}

// seedMembershipEvents reads membership stamps in batches and inserts them into the analytics store.
func (s *AnalyticsService) seedMembershipEvents(ctx context.Context, summary *SeedSummary) error {
	lastCreatedAt := time.Time{}
	lastID := ""
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		rows := make([]membershipSeedRow, 0, seedBatchSize)
		query := s.db.WithContext(ctx).
			Table("membership_stamps").
			Select("id", "contact_id", "channel", "action", "source", "occurred_at", "created_at")
		if !lastCreatedAt.IsZero() {
			query = query.Where("(created_at > ?) OR (created_at = ? AND id > ?)", lastCreatedAt, lastCreatedAt, lastID)
		}
		if err := query.Order("created_at ASC").Order("id ASC").Limit(seedBatchSize).Scan(&rows).Error; err != nil {
			return fmt.Errorf("seed membership events batch: %w", err)
		}
		if len(rows) == 0 {
			break
		}

		payload := make([]port.MembershipEvent, 0, len(rows))
		for _, row := range rows {
			payload = append(payload, port.MembershipEvent{
				ContactID:  strings.TrimSpace(row.ContactID),
				Channel:    strings.TrimSpace(row.Channel),
				Action:     strings.TrimSpace(row.Action),
				Source:     strings.TrimSpace(row.Source),
				OccurredAt: row.OccurredAt.UTC(),
			})
		}
		if err := s.store.InsertMembershipEvents(ctx, payload); err != nil {
			return fmt.Errorf("insert membership events batch: %w", err)
		}
		summary.MembershipEvents += int64(len(payload))
		lastCreatedAt = rows[len(rows)-1].CreatedAt.UTC()
		lastID = rows[len(rows)-1].ID
	}

	return nil
}
