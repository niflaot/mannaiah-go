package application

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"

	"gorm.io/gorm"
	analyticsdomain "mannaiah/module/analytics/domain"
	analyticsport "mannaiah/module/analytics/port"
	"mannaiah/module/segment/domain"
	"mannaiah/module/segment/port"
)

var (
	// ErrNilRepository is returned when nil repository dependencies are provided.
	ErrNilRepository = errors.New("segment repository must not be nil")
	// ErrNilDB is returned when nil db dependencies are provided.
	ErrNilDB = errors.New("segment db must not be nil")
)

var slugPattern = regexp.MustCompile(`^[a-z0-9-]+$`)

// CreateCommand defines segment creation payload values.
type CreateCommand struct {
	// Name defines segment names.
	Name string
	// Slug defines segment slugs.
	Slug string
	// Channel defines target channel values.
	Channel string
	// Filters defines filter DSL values.
	Filters []domain.Filter
}

// UpdateCommand defines segment update payload values.
type UpdateCommand struct {
	// Name defines optional segment names.
	Name *string
	// Slug defines optional segment slugs.
	Slug *string
	// Channel defines optional target channel values.
	Channel *string
	// Filters defines optional filter DSL values.
	Filters *[]domain.Filter
}

// ListResult defines paged segment query output values.
type ListResult struct {
	// Data defines segment rows in the current page.
	Data []domain.Segment `json:"data"`
	// Page defines current page number.
	Page int `json:"page"`
	// Limit defines current page size.
	Limit int `json:"limit"`
	// Total defines total matching rows.
	Total int64 `json:"total"`
	// TotalPages defines total available pages.
	TotalPages int `json:"totalPages"`
}

// ResolveResult defines segment resolution output values.
type ResolveResult struct {
	// SegmentID defines resolved segment identifier values.
	SegmentID string `json:"segmentId"`
	// ContactIDs defines resolved contact identifier values.
	ContactIDs []string `json:"contactIds"`
}

// Service defines segment use-case behavior.
type Service interface {
	// Create persists segment rows.
	Create(ctx context.Context, command CreateCommand) (*domain.Segment, error)
	// Get retrieves one segment by id.
	Get(ctx context.Context, id string) (*domain.Segment, error)
	// List retrieves paged segment rows.
	List(ctx context.Context, page int, limit int) (*ListResult, error)
	// Update persists segment row updates.
	Update(ctx context.Context, id string, command UpdateCommand) (*domain.Segment, error)
	// Delete removes one segment by id.
	Delete(ctx context.Context, id string) error
	// Resolve resolves contact ids for one segment.
	Resolve(ctx context.Context, id string, page int, limit int) (*ResolveResult, error)
	// Count resolves contact count for one segment.
	Count(ctx context.Context, id string) (int64, error)
}

// SegmentService implements segment use-cases.
type SegmentService struct {
	// repository defines segment persistence dependencies.
	repository port.Repository
	// resolver defines optional analytics resolver dependencies.
	resolver analyticsport.Resolver
	// db defines transactional database dependencies for fallback resolution.
	db *gorm.DB
}

// NewService creates segment services.
func NewService(repository port.Repository, resolver analyticsport.Resolver, db *gorm.DB) (*SegmentService, error) {
	if repository == nil {
		return nil, ErrNilRepository
	}
	if db == nil {
		return nil, ErrNilDB
	}

	return &SegmentService{repository: repository, resolver: resolver, db: db}, nil
}

// Create persists segment rows.
func (s *SegmentService) Create(ctx context.Context, command CreateCommand) (*domain.Segment, error) {
	name := strings.TrimSpace(command.Name)
	if name == "" {
		return nil, domain.ErrInvalidName
	}
	slug := strings.TrimSpace(strings.ToLower(command.Slug))
	if slug == "" || !slugPattern.MatchString(slug) {
		return nil, domain.ErrInvalidSlug
	}

	segment := &domain.Segment{Name: name, Slug: slug, Channel: strings.TrimSpace(command.Channel), Filters: normalizeFilters(command.Filters)}
	if err := s.repository.Create(ctx, segment); err != nil {
		return nil, fmt.Errorf("create segment: %w", err)
	}

	return segment, nil
}

// Get retrieves one segment by id.
func (s *SegmentService) Get(ctx context.Context, id string) (*domain.Segment, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, domain.ErrInvalidID
	}

	segment, err := s.repository.GetByID(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("get segment: %w", err)
	}

	return segment, nil
}

// List retrieves paged segment rows.
func (s *SegmentService) List(ctx context.Context, page int, limit int) (*ListResult, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	rows, total, err := s.repository.List(ctx, page, limit)
	if err != nil {
		return nil, fmt.Errorf("list segments: %w", err)
	}
	pages := 0
	if total > 0 {
		pages = int(math.Ceil(float64(total) / float64(limit)))
	}

	return &ListResult{Data: rows, Page: page, Limit: limit, Total: total, TotalPages: pages}, nil
}

// Update persists segment row updates.
func (s *SegmentService) Update(ctx context.Context, id string, command UpdateCommand) (*domain.Segment, error) {
	segment, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if command.Name != nil {
		name := strings.TrimSpace(*command.Name)
		if name == "" {
			return nil, domain.ErrInvalidName
		}
		segment.Name = name
	}
	if command.Slug != nil {
		slug := strings.TrimSpace(strings.ToLower(*command.Slug))
		if slug == "" || !slugPattern.MatchString(slug) {
			return nil, domain.ErrInvalidSlug
		}
		segment.Slug = slug
	}
	if command.Channel != nil {
		segment.Channel = strings.TrimSpace(*command.Channel)
	}
	if command.Filters != nil {
		segment.Filters = normalizeFilters(*command.Filters)
	}

	if err := s.repository.Update(ctx, segment); err != nil {
		return nil, fmt.Errorf("update segment: %w", err)
	}

	return segment, nil
}

// Delete removes one segment by id.
func (s *SegmentService) Delete(ctx context.Context, id string) error {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return domain.ErrInvalidID
	}

	if err := s.repository.Delete(ctx, trimmedID); err != nil {
		return fmt.Errorf("delete segment: %w", err)
	}

	return nil
}

// Resolve resolves contact ids for one segment.
func (s *SegmentService) Resolve(ctx context.Context, id string, page int, limit int) (*ResolveResult, error) {
	segment, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 1000
	}

	filter := toAnalyticsFilter(segment.Filters)
	contactIDs, resolveErr := s.resolveWithAnalytics(ctx, filter, page, limit)
	if resolveErr != nil {
		return nil, resolveErr
	}

	return &ResolveResult{SegmentID: segment.ID, ContactIDs: contactIDs}, nil
}

// Count resolves contact count for one segment.
func (s *SegmentService) Count(ctx context.Context, id string) (int64, error) {
	segment, err := s.Get(ctx, id)
	if err != nil {
		return 0, err
	}

	filter := toAnalyticsFilter(segment.Filters)
	count, resolveErr := s.resolveCountWithAnalytics(ctx, filter)
	if resolveErr != nil {
		return 0, resolveErr
	}

	return count, nil
}

// resolveWithAnalytics resolves contact ids using analytics resolver with SQL fallback.
func (s *SegmentService) resolveWithAnalytics(ctx context.Context, filter analyticsdomain.SegmentFilter, page int, limit int) ([]string, error) {
	if s.resolver != nil {
		rows, err := s.resolver.ResolveContacts(ctx, filter, page, limit)
		if err == nil {
			return rows, nil
		}
	}

	return s.resolveFallback(ctx, filter, page, limit)
}

// resolveCountWithAnalytics resolves contact counts using analytics resolver with SQL fallback.
func (s *SegmentService) resolveCountWithAnalytics(ctx context.Context, filter analyticsdomain.SegmentFilter) (int64, error) {
	if s.resolver != nil {
		var (
			total int64
			page  = 1
			limit = 1000
		)
		for {
			rows, err := s.resolver.ResolveContacts(ctx, filter, page, limit)
			if err != nil {
				break
			}

			total += int64(len(rows))
			if len(rows) < limit {
				return total, nil
			}
			page++
		}
	}

	return s.resolveCountFallback(ctx, filter)
}

// resolveFallback resolves contact ids directly from transactional MySQL/SQLite tables.
func (s *SegmentService) resolveFallback(ctx context.Context, filter analyticsdomain.SegmentFilter, page int, limit int) ([]string, error) {
	offset := (page - 1) * limit
	db := s.db.WithContext(ctx).Table("contacts c").Select("c.id").Where("c.deleted_at IS NULL")
	if len(filter.CityCodes) > 0 {
		db = db.Where("c.city_code IN ?", filter.CityCodes)
	}
	if filter.RequireEmailOptIn {
		db = db.Where("EXISTS (SELECT 1 FROM membership_status ms WHERE ms.contact_id = c.id AND ms.channel = ? AND ms.action = ?)", "email", "opt_in")
	}
	if filter.MinTotalSpend != nil {
		db = db.Where("EXISTS (SELECT 1 FROM orders o WHERE o.contact_id = c.id GROUP BY o.contact_id HAVING SUM(o.total_value) >= ?)", *filter.MinTotalSpend)
	}
	if strings.TrimSpace(filter.PurchasedSKU) != "" {
		db = db.Where("EXISTS (SELECT 1 FROM orders o JOIN order_items oi ON oi.order_id = o.id WHERE o.contact_id = c.id AND oi.sku = ?)", strings.TrimSpace(filter.PurchasedSKU))
	}

	ids := make([]string, 0, limit)
	if err := db.Order("c.id ASC").Offset(offset).Limit(limit).Scan(&ids).Error; err != nil {
		return nil, fmt.Errorf("resolve segment fallback contacts: %w", err)
	}

	return ids, nil
}

// resolveCountFallback resolves contact counts directly from transactional MySQL/SQLite tables.
func (s *SegmentService) resolveCountFallback(ctx context.Context, filter analyticsdomain.SegmentFilter) (int64, error) {
	db := s.db.WithContext(ctx).Table("contacts c").Where("c.deleted_at IS NULL")
	if len(filter.CityCodes) > 0 {
		db = db.Where("c.city_code IN ?", filter.CityCodes)
	}
	if filter.RequireEmailOptIn {
		db = db.Where("EXISTS (SELECT 1 FROM membership_status ms WHERE ms.contact_id = c.id AND ms.channel = ? AND ms.action = ?)", "email", "opt_in")
	}
	if filter.MinTotalSpend != nil {
		db = db.Where("EXISTS (SELECT 1 FROM orders o WHERE o.contact_id = c.id GROUP BY o.contact_id HAVING SUM(o.total_value) >= ?)", *filter.MinTotalSpend)
	}
	if strings.TrimSpace(filter.PurchasedSKU) != "" {
		db = db.Where("EXISTS (SELECT 1 FROM orders o JOIN order_items oi ON oi.order_id = o.id WHERE o.contact_id = c.id AND oi.sku = ?)", strings.TrimSpace(filter.PurchasedSKU))
	}

	var count int64
	if err := db.Select("COUNT(DISTINCT c.id)").Scan(&count).Error; err != nil {
		return 0, fmt.Errorf("resolve segment fallback count: %w", err)
	}

	return count, nil
}

// normalizeFilters normalizes filters and removes empty types.
func normalizeFilters(filters []domain.Filter) []domain.Filter {
	normalized := make([]domain.Filter, 0, len(filters))
	for _, filter := range filters {
		trimmedType := strings.TrimSpace(filter.Type)
		if trimmedType == "" {
			continue
		}
		normalized = append(normalized, domain.Filter{Type: trimmedType, Value: filter.Value})
	}

	return normalized
}

// toAnalyticsFilter maps segment filters into analytics filter payload values.
func toAnalyticsFilter(filters []domain.Filter) analyticsdomain.SegmentFilter {
	result := analyticsdomain.SegmentFilter{}
	for _, filter := range filters {
		switch strings.TrimSpace(strings.ToLower(filter.Type)) {
		case "city_code_in":
			if values, ok := asStringSlice(filter.Value); ok {
				result.CityCodes = values
			}
		case "min_total_spend":
			if value, ok := asFloat64(filter.Value); ok {
				result.MinTotalSpend = &value
			}
		case "email_opt_in":
			if value, ok := asBool(filter.Value); ok {
				result.RequireEmailOptIn = value
			}
		case "purchased_sku":
			if value, ok := filter.Value.(string); ok {
				result.PurchasedSKU = strings.TrimSpace(value)
			}
		}
	}

	return result
}

// asStringSlice converts filter values into string slices.
func asStringSlice(value any) ([]string, bool) {
	raw, ok := value.([]any)
	if ok {
		result := make([]string, 0, len(raw))
		for _, row := range raw {
			if text, castOK := row.(string); castOK && strings.TrimSpace(text) != "" {
				result = append(result, strings.TrimSpace(text))
			}
		}
		return result, true
	}

	if typed, ok := value.([]string); ok {
		return typed, true
	}

	return nil, false
}

// asFloat64 converts filter values into float64 values.
func asFloat64(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	default:
		return 0, false
	}
}

// asBool converts filter values into boolean values.
func asBool(value any) (bool, bool) {
	typed, ok := value.(bool)
	return typed, ok
}
