package store

import (
	"context"
	"errors"
	"fmt"
	"sort"
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

// SaveStamp persists immutable stamps and resolves latest effective status values.
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
		latestStatus, latestErr := r.selectLatestStatusTx(
			tx.Clauses(clause.Locking{Strength: "UPDATE"}),
			trimmedContactID,
			input.Channel,
		)
		if latestErr != nil && !errors.Is(latestErr, domain.ErrStatusNotFound) {
			return latestErr
		}
		if latestStatus != nil && latestStatus.Action == input.Action {
			result = port.StampResult{Status: *latestStatus, Created: false}
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

		currentStatus, statusErr := r.selectLatestStatusTx(tx, trimmedContactID, input.Channel)
		if statusErr != nil {
			return statusErr
		}

		result = port.StampResult{Status: *currentStatus, Created: true}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// GetStatus retrieves latest effective status by contact and channel.
func (r *Repository) GetStatus(ctx context.Context, contactID string, channel domain.Channel) (*domain.Status, error) {
	return r.selectLatestStatusTx(r.db.WithContext(ctx), strings.TrimSpace(contactID), channel)
}

// GetStatuses retrieves effective statuses for every contact channel.
func (r *Repository) GetStatuses(ctx context.Context, contactID string) ([]domain.Status, error) {
	trimmedContactID := strings.TrimSpace(contactID)
	if trimmedContactID == "" {
		return nil, domain.ErrInvalidContactID
	}

	rows := make([]string, 0, 8)
	err := r.db.WithContext(ctx).
		Table((stampModel{}).TableName()).
		Distinct("channel").
		Where("contact_id = ? AND channel <> ?", trimmedContactID, string(domain.ChannelAll)).
		Scan(&rows).
		Error
	if err != nil {
		return nil, fmt.Errorf("select membership status channels: %w", err)
	}

	channelSet := map[domain.Channel]struct{}{
		domain.ChannelEmail: {},
	}
	for _, row := range rows {
		resolved := domain.Channel(strings.TrimSpace(row))
		if !resolved.IsValid() || resolved == domain.ChannelAll {
			continue
		}
		channelSet[resolved] = struct{}{}
	}

	channels := make([]domain.Channel, 0, len(channelSet))
	for channel := range channelSet {
		channels = append(channels, channel)
	}
	sort.Slice(channels, func(left int, right int) bool {
		return channels[left] < channels[right]
	})

	statuses := make([]domain.Status, 0, len(channels))
	for _, channel := range channels {
		status, statusErr := r.selectLatestStatusTx(r.db.WithContext(ctx), trimmedContactID, channel)
		if errors.Is(statusErr, domain.ErrStatusNotFound) {
			continue
		}
		if statusErr != nil {
			return nil, statusErr
		}
		statuses = append(statuses, *status)
	}
	if len(statuses) == 0 {
		return nil, domain.ErrStatusNotFound
	}

	return statuses, nil
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

// selectLatestStatusTx resolves latest effective status from immutable stamp rows.
func (r *Repository) selectLatestStatusTx(tx *gorm.DB, contactID string, channel domain.Channel) (*domain.Status, error) {
	query := tx.Where("contact_id = ?", contactID)
	if channel == domain.ChannelAll {
		query = query.Where("channel = ?", string(domain.ChannelAll))
	} else {
		query = query.Where("(channel = ? OR channel = ?)", string(channel), string(domain.ChannelAll))
	}

	row := stampModel{}
	err := query.Order("occurred_at DESC, id DESC").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrStatusNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select membership latest status: %w", err)
	}

	resolvedChannel := channel
	if resolvedChannel == "" {
		resolvedChannel = domain.Channel(strings.TrimSpace(row.Channel))
	}

	return &domain.Status{
		ContactID:  strings.TrimSpace(row.ContactID),
		Channel:    resolvedChannel,
		Action:     domain.Action(strings.TrimSpace(row.Action)),
		Source:     strings.TrimSpace(row.Source),
		OccurredAt: row.OccurredAt.UTC(),
		UpdatedAt:  row.CreatedAt.UTC(),
	}, nil
}
