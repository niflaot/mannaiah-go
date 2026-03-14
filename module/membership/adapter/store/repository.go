package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"mannaiah/module/membership/domain"
	"mannaiah/module/membership/port"
)

var (
	// ErrNilDB is returned when nil db dependencies are provided.
	ErrNilDB = errors.New("membership db must not be nil")
)

// Repository defines GORM-backed membership persistence behavior.
type Repository struct {
	// db defines GORM database dependencies.
	db *gorm.DB
}

var (
	// _ ensures Repository satisfies membership repository contracts.
	_ port.Repository = (*Repository)(nil)
)

// NewRepository creates GORM-backed membership repositories.
func NewRepository(db *gorm.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	return &Repository{db: db}, nil
}

// SaveStamp persists immutable stamps and updates latest status snapshots.
func (r *Repository) SaveStamp(ctx context.Context, input port.StampInput) (*port.StampResult, error) {
	trimmedContactID := strings.TrimSpace(input.ContactID)
	if trimmedContactID == "" {
		return nil, domain.ErrInvalidContactID
	}
	if !input.Channel.IsValid() {
		return nil, domain.ErrInvalidChannel
	}
	if !input.Action.IsValid() {
		return nil, domain.ErrInvalidAction
	}

	occurredAt := input.OccurredAt.UTC()
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	var result port.StampResult
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		latest := stampModel{}
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("contact_id = ? AND channel = ?", trimmedContactID, string(input.Channel)).
			Order("occurred_at DESC, id DESC").
			First(&latest).
			Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("select latest membership stamp: %w", err)
		}
		if err == nil && strings.TrimSpace(latest.Action) == string(input.Action) {
			status, statusErr := r.selectStatusTx(tx, trimmedContactID, input.Channel)
			if statusErr != nil {
				return statusErr
			}
			result = port.StampResult{Status: *status, Created: false}
			return nil
		}

		now := time.Now().UTC()
		stampRow := stampModel{
			ID:         uuid.NewString(),
			ContactID:  trimmedContactID,
			Channel:    string(input.Channel),
			Action:     string(input.Action),
			Source:     strings.TrimSpace(input.Source),
			OccurredAt: occurredAt,
			CreatedAt:  now,
		}
		if stampRow.Source == "" {
			stampRow.Source = "api"
		}
		if createErr := tx.Create(&stampRow).Error; createErr != nil {
			return fmt.Errorf("insert membership stamp: %w", createErr)
		}

		statusRow := statusModel{
			ContactID:  trimmedContactID,
			Channel:    string(input.Channel),
			Action:     string(input.Action),
			Source:     stampRow.Source,
			OccurredAt: occurredAt,
			UpdatedAt:  now,
		}
		if upsertErr := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "contact_id"}, {Name: "channel"}},
			DoUpdates: clause.AssignmentColumns([]string{"action", "source", "occurred_at", "updated_at"}),
		}).Create(&statusRow).Error; upsertErr != nil {
			return fmt.Errorf("upsert membership status: %w", upsertErr)
		}

		result = port.StampResult{Status: domain.Status{
			ContactID:  statusRow.ContactID,
			Channel:    domain.Channel(statusRow.Channel),
			Action:     domain.Action(statusRow.Action),
			Source:     statusRow.Source,
			OccurredAt: statusRow.OccurredAt.UTC(),
			UpdatedAt:  statusRow.UpdatedAt.UTC(),
		}, Created: true}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// GetStatus retrieves latest status by contact and channel.
func (r *Repository) GetStatus(ctx context.Context, contactID string, channel domain.Channel) (*domain.Status, error) {
	return r.selectStatusTx(r.db.WithContext(ctx), strings.TrimSpace(contactID), channel)
}

// ListStamps retrieves stamps by contact and channel filters.
func (r *Repository) ListStamps(ctx context.Context, contactID string, channel domain.Channel, limit int) ([]domain.Stamp, error) {
	if limit <= 0 {
		limit = 100
	}

	rows := make([]stampModel, 0, limit)
	err := r.db.WithContext(ctx).
		Where("contact_id = ? AND channel = ?", strings.TrimSpace(contactID), string(channel)).
		Order("occurred_at DESC, id DESC").
		Limit(limit).
		Find(&rows).
		Error
	if err != nil {
		return nil, fmt.Errorf("select membership stamps: %w", err)
	}

	stamps := make([]domain.Stamp, 0, len(rows))
	for _, row := range rows {
		stamps = append(stamps, domain.Stamp{
			ID:         row.ID,
			ContactID:  row.ContactID,
			Channel:    domain.Channel(row.Channel),
			Action:     domain.Action(row.Action),
			Source:     row.Source,
			OccurredAt: row.OccurredAt.UTC(),
			CreatedAt:  row.CreatedAt.UTC(),
		})
	}

	return stamps, nil
}

// selectStatusTx retrieves status rows within tx or db contexts.
func (r *Repository) selectStatusTx(tx *gorm.DB, contactID string, channel domain.Channel) (*domain.Status, error) {
	row := statusModel{}
	err := tx.Where("contact_id = ? AND channel = ?", contactID, string(channel)).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrStatusNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select membership status: %w", err)
	}

	return &domain.Status{
		ContactID:  row.ContactID,
		Channel:    domain.Channel(row.Channel),
		Action:     domain.Action(row.Action),
		Source:     row.Source,
		OccurredAt: row.OccurredAt.UTC(),
		UpdatedAt:  row.UpdatedAt.UTC(),
	}, nil
}
