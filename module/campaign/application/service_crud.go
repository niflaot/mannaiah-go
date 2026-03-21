package application

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"mannaiah/module/campaign/domain"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9-]+$`)

// Create persists campaign rows.
func (s *CampaignService) Create(ctx context.Context, command CreateCommand) (*domain.Campaign, error) {
	name := strings.TrimSpace(command.Name)
	if name == "" {
		return nil, domain.ErrInvalidName
	}
	slug := strings.TrimSpace(strings.ToLower(command.Slug))
	if slug == "" || !slugPattern.MatchString(slug) {
		return nil, domain.ErrInvalidSlug
	}

	campaign := &domain.Campaign{
		Name:          name,
		Slug:          slug,
		Channel:       strings.TrimSpace(command.Channel),
		SegmentID:     strings.TrimSpace(command.SegmentID),
		Subject:       strings.TrimSpace(command.Subject),
		HTMLBody:      command.HTMLBody,
		TextBody:      command.TextBody,
		Status:        domain.StatusPlanned,
		TemplateVars:  command.TemplateVars,
		ProductBlocks: command.ProductBlocks,
	}
	if campaign.Channel == "" {
		campaign.Channel = "email"
	}

	if err := s.repository.Create(ctx, campaign); err != nil {
		return nil, fmt.Errorf("create campaign: %w", err)
	}

	return campaign, nil
}

// Get retrieves one campaign by id.
func (s *CampaignService) Get(ctx context.Context, id string) (*domain.Campaign, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, domain.ErrInvalidID
	}

	campaign, err := s.repository.GetByID(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("get campaign: %w", err)
	}

	return campaign, nil
}

// List retrieves paged campaign rows.
func (s *CampaignService) List(ctx context.Context, page int, limit int) (*ListResult, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	rows, total, err := s.repository.List(ctx, page, limit)
	if err != nil {
		return nil, fmt.Errorf("list campaigns: %w", err)
	}

	return &ListResult{Data: rows, Page: page, Limit: limit, Total: total, TotalPages: totalPages(total, limit)}, nil
}

// Update persists campaign row updates.
func (s *CampaignService) Update(ctx context.Context, id string, command UpdateCommand) (*domain.Campaign, error) {
	campaign, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if campaign.Status == domain.StatusProcessing || campaign.Status == domain.StatusSent {
		return nil, domain.ErrSendConflict
	}

	if command.Name != nil {
		name := strings.TrimSpace(*command.Name)
		if name == "" {
			return nil, domain.ErrInvalidName
		}
		campaign.Name = name
	}
	if command.Slug != nil {
		slug := strings.TrimSpace(strings.ToLower(*command.Slug))
		if slug == "" || !slugPattern.MatchString(slug) {
			return nil, domain.ErrInvalidSlug
		}
		campaign.Slug = slug
	}
	if command.Channel != nil {
		campaign.Channel = strings.TrimSpace(*command.Channel)
	}
	if command.SegmentID != nil {
		campaign.SegmentID = strings.TrimSpace(*command.SegmentID)
	}
	if command.Subject != nil {
		campaign.Subject = strings.TrimSpace(*command.Subject)
	}
	if command.HTMLBody != nil {
		campaign.HTMLBody = *command.HTMLBody
	}
	if command.TextBody != nil {
		campaign.TextBody = *command.TextBody
	}
	if command.TemplateVars != nil {
		campaign.TemplateVars = command.TemplateVars
	}
	if command.ProductBlocks != nil {
		campaign.ProductBlocks = command.ProductBlocks
	}

	if err := s.repository.Update(ctx, campaign); err != nil {
		return nil, fmt.Errorf("update campaign: %w", err)
	}

	return campaign, nil
}

// Delete removes one campaign by id.
func (s *CampaignService) Delete(ctx context.Context, id string) error {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return domain.ErrInvalidID
	}

	if err := s.repository.Delete(ctx, trimmedID); err != nil {
		return fmt.Errorf("delete campaign: %w", err)
	}

	return nil
}

// ListDeliveries retrieves paged delivery rows for one campaign.
func (s *CampaignService) ListDeliveries(ctx context.Context, id string, page int, limit int) (*DeliveryListResult, error) {
	campaign, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if s.deliveryReader == nil {
		return &DeliveryListResult{Data: []DeliveryEntry{}, Page: page, Limit: limit}, nil
	}
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 50
	}

	rows, total, err := s.deliveryReader.ListByCampaignID(ctx, campaign.ID, page, limit)
	if err != nil {
		return nil, fmt.Errorf("list campaign deliveries: %w", err)
	}

	entries := make([]DeliveryEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, DeliveryEntry{
			ContactID: row.ContactID,
			Email:     row.Email,
			Status:    row.Status,
			CreatedAt: row.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			UpdatedAt: row.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}

	return &DeliveryListResult{
		Data:       entries,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages(total, limit),
	}, nil
}
