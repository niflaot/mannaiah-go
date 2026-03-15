package application

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"

	analyticsdomain "mannaiah/module/analytics/domain"
	analyticsport "mannaiah/module/analytics/port"
	"mannaiah/module/segment/domain"
	"mannaiah/module/segment/port"
)

var (
	// ErrNilRepository is returned when nil repository dependencies are provided.
	ErrNilRepository = errors.New("segment repository must not be nil")
	// ErrResolverUnavailable is returned when analytics resolver dependencies are unavailable.
	ErrResolverUnavailable = errors.New("segment resolver is not configured")
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
	// PreviewCount resolves contact count for an unsaved filter set.
	PreviewCount(ctx context.Context, filters []domain.Filter) (int64, error)
}

// SegmentService implements segment use-cases.
type SegmentService struct {
	// repository defines segment persistence dependencies.
	repository port.Repository
	// resolver defines optional analytics resolver dependencies.
	resolver analyticsport.Resolver
}

// NewService creates segment services.
func NewService(repository port.Repository, resolver analyticsport.Resolver) (*SegmentService, error) {
	if repository == nil {
		return nil, ErrNilRepository
	}
	return &SegmentService{repository: repository, resolver: resolver}, nil
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
	if err := validateFilters(command.Filters); err != nil {
		return nil, err
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
		if err := validateFilters(*command.Filters); err != nil {
			return nil, err
		}
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

// PreviewCount resolves contact count for an unsaved filter set without persisting a segment.
func (s *SegmentService) PreviewCount(ctx context.Context, filters []domain.Filter) (int64, error) {
	if err := validateFilters(filters); err != nil {
		return 0, err
	}

	filter := toAnalyticsFilter(filters)
	count, err := s.resolveCountWithAnalytics(ctx, filter)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// resolveWithAnalytics resolves contact ids using analytics resolver with SQL fallback.
func (s *SegmentService) resolveWithAnalytics(ctx context.Context, filter analyticsdomain.SegmentFilter, page int, limit int) ([]string, error) {
	if s.resolver == nil {
		return nil, ErrResolverUnavailable
	}

	rows, err := s.resolver.ResolveContacts(ctx, filter, page, limit)
	if err != nil {
		return nil, fmt.Errorf("resolve analytics segment contacts: %w", err)
	}

	return rows, nil
}

// resolveCountWithAnalytics resolves contact counts using analytics resolver with SQL fallback.
func (s *SegmentService) resolveCountWithAnalytics(ctx context.Context, filter analyticsdomain.SegmentFilter) (int64, error) {
	if s.resolver == nil {
		return 0, ErrResolverUnavailable
	}

	count, err := s.resolver.CountContacts(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("count analytics segment contacts: %w", err)
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
		normalized = append(normalized, domain.Filter{Type: trimmedType, Value: filter.Value, Parameters: cloneParameters(filter.Parameters)})
	}

	return normalized
}

// toAnalyticsFilter maps segment filters into analytics filter payload values.
func toAnalyticsFilter(filters []domain.Filter) analyticsdomain.SegmentFilter {
	result := analyticsdomain.SegmentFilter{}
	for _, filter := range filters {
		filterType := strings.TrimSpace(strings.ToLower(filter.Type))
		switch filterType {
		case "city_code_in":
			if values, ok := asStringSlice(filter.Value); ok {
				result.CityCodes = values
			}
		case "city":
			if values, ok := asStringSlice(filterParameter(filter, "codes")); ok {
				result.CityCodes = values
			}
		case "min_total_spend":
			if value, ok := asFloat64(filter.Value); ok {
				result.MinTotalSpend = &value
			}
		case "email_opt_in":
			if value, ok := asBool(filter.Value); ok {
				result.RequireEmailOptIn = &value
			}
		case "purchased_sku":
			if value, ok := filter.Value.(string); ok {
				result.PurchasedSKU = strings.TrimSpace(value)
			}
		case "order_recency":
			if value, ok := asInt(filterParameter(filter, "days")); ok && value > 0 {
				result.OrderRecencyDays = &value
			}
		case "no_order_recency":
			if value, ok := asInt(filterParameter(filter, "days")); ok && value > 0 {
				result.NoOrderRecencyDays = &value
			}
		case "category":
			if value, ok := asString(filterParameter(filter, "pattern")); ok {
				result.CategoryPattern = value
			}
		case "top_spenders":
			if value, ok := asInt(filterParameter(filter, "limit")); ok && value > 0 {
				result.TopSpendersLimit = &value
			}
			if value, ok := asFloat64(filterParameter(filter, "percentage")); ok && value > 0 {
				result.TopSpendersPercentage = &value
			}
		case "first_purchase_only":
			if value, ok := asBool(filterParameter(filter, "enabled")); ok {
				result.FirstPurchaseOnly = value
			} else {
				result.FirstPurchaseOnly = true
			}
		case "subscribed_no_buy":
			if value, ok := asBool(filterParameter(filter, "enabled")); ok {
				result.SubscribedNoBuy = value
			} else {
				result.SubscribedNoBuy = true
			}
		case "opt_in_status":
			if channel, ok := asString(filterParameter(filter, "channel")); ok {
				result.OptInChannel = channel
			}
			if status, ok := asString(filterParameter(filter, "status")); ok {
				result.OptInAction = status
			}
		case "metadata":
			if key, ok := asString(filterParameter(filter, "key")); ok {
				result.MetadataKey = key
			}
			if value, ok := asString(filterParameter(filter, "value")); ok {
				result.MetadataValue = value
			}
		case "order_status":
			if values, ok := asStringSlice(filterParameter(filter, "statuses")); ok {
				result.OrderStatuses = values
			}
		}
	}

	return result
}

// validateFilters validates filter types and parameters.
func validateFilters(filters []domain.Filter) error {
	for _, filter := range filters {
		filterType := strings.TrimSpace(strings.ToLower(filter.Type))
		switch filterType {
		case "city_code_in", "min_total_spend", "email_opt_in", "purchased_sku":
		case "city":
			if _, ok := asStringSlice(filterParameter(filter, "codes")); !ok {
				return domain.ErrInvalidFilter
			}
		case "order_recency", "no_order_recency":
			value, ok := asInt(filterParameter(filter, "days"))
			if !ok || value <= 0 {
				return domain.ErrInvalidFilter
			}
		case "category":
			if _, ok := asString(filterParameter(filter, "pattern")); !ok {
				return domain.ErrInvalidFilter
			}
		case "top_spenders":
			_, hasLimit := asInt(filterParameter(filter, "limit"))
			_, hasPercentage := asFloat64(filterParameter(filter, "percentage"))
			if !hasLimit && !hasPercentage {
				return domain.ErrInvalidFilter
			}
		case "first_purchase_only", "subscribed_no_buy":
		case "opt_in_status":
			if _, ok := asString(filterParameter(filter, "channel")); !ok {
				return domain.ErrInvalidFilter
			}
			if _, ok := asString(filterParameter(filter, "status")); !ok {
				return domain.ErrInvalidFilter
			}
		case "metadata":
			if _, ok := asString(filterParameter(filter, "key")); !ok {
				return domain.ErrInvalidFilter
			}
		case "order_status":
			values, ok := asStringSlice(filterParameter(filter, "statuses"))
			if !ok || len(values) == 0 {
				return domain.ErrInvalidFilter
			}
		default:
			return domain.ErrInvalidFilter
		}
	}

	return nil
}

// filterParameter resolves normalized filter parameter values.
func filterParameter(filter domain.Filter, key string) any {
	if len(filter.Parameters) == 0 {
		return nil
	}

	return filter.Parameters[strings.TrimSpace(key)]
}

// cloneParameters clones parameter maps.
func cloneParameters(value map[string]any) map[string]any {
	if len(value) == 0 {
		return nil
	}

	result := make(map[string]any, len(value))
	for key, row := range value {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		result[trimmed] = row
	}
	if len(result) == 0 {
		return nil
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
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		trimmed := strings.TrimSpace(strings.ToLower(typed))
		if trimmed == "true" {
			return true, true
		}
		if trimmed == "false" {
			return false, true
		}
		return false, false
	default:
		return false, false
	}
}

// asInt converts filter values into integer values.
func asInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	case float32:
		return int(typed), true
	default:
		return 0, false
	}
}

// asString converts filter values into string values.
func asString(value any) (string, bool) {
	typed, ok := value.(string)
	if !ok {
		return "", false
	}
	trimmed := strings.TrimSpace(typed)
	if trimmed == "" {
		return "", false
	}

	return trimmed, true
}
