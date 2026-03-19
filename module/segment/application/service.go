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
	// ErrRFMGroupRepositoryUnavailable is returned when rfm-group lookup dependencies are unavailable.
	ErrRFMGroupRepositoryUnavailable = errors.New("rfm group repository is not configured")
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
	// rfmGroupRepository defines optional RFM group lookup dependencies.
	rfmGroupRepository analyticsport.RFMGroupRepository
}

// NewService creates segment services.
func NewService(
	repository port.Repository,
	resolver analyticsport.Resolver,
	rfmGroupRepositories ...analyticsport.RFMGroupRepository,
) (*SegmentService, error) {
	if repository == nil {
		return nil, ErrNilRepository
	}
	var rfmGroupRepository analyticsport.RFMGroupRepository
	if len(rfmGroupRepositories) > 0 {
		rfmGroupRepository = rfmGroupRepositories[0]
	}
	return &SegmentService{
		repository:         repository,
		resolver:           resolver,
		rfmGroupRepository: rfmGroupRepository,
	}, nil
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

	filter, err := s.buildAnalyticsFilter(ctx, segment.Filters)
	if err != nil {
		return nil, err
	}
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

	filter, err := s.buildAnalyticsFilter(ctx, segment.Filters)
	if err != nil {
		return 0, err
	}
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

	filter, err := s.buildAnalyticsFilter(ctx, filters)
	if err != nil {
		return 0, err
	}
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

// buildAnalyticsFilter maps DSL filters and expands rfm_group clauses into concrete RFM ranges.
func (s *SegmentService) buildAnalyticsFilter(ctx context.Context, filters []domain.Filter) (analyticsdomain.SegmentFilter, error) {
	if err := validateFilters(filters); err != nil {
		return analyticsdomain.SegmentFilter{}, err
	}
	mapped := toAnalyticsFilter(filters)
	expanded, err := s.expandRFMGroupClauses(ctx, mapped)
	if err != nil {
		return analyticsdomain.SegmentFilter{}, err
	}

	return expanded, nil
}

// expandRFMGroupClauses resolves rfm_group slugs and rewrites them into rfm_range clauses.
func (s *SegmentService) expandRFMGroupClauses(ctx context.Context, filter analyticsdomain.SegmentFilter) (analyticsdomain.SegmentFilter, error) {
	if len(filter.Clauses) == 0 {
		return filter, nil
	}
	hasRFMGroupClause := false
	for _, clause := range filter.Clauses {
		if strings.TrimSpace(strings.ToLower(clause.Type)) == "rfm_group" {
			hasRFMGroupClause = true
			break
		}
	}
	if !hasRFMGroupClause {
		return filter, nil
	}
	if s.rfmGroupRepository == nil {
		return analyticsdomain.SegmentFilter{}, ErrRFMGroupRepositoryUnavailable
	}

	groupsBySlug, err := s.fetchRFMGroupsBySlug(ctx)
	if err != nil {
		return analyticsdomain.SegmentFilter{}, err
	}

	expandedClauses := make([]analyticsdomain.SegmentClause, 0, len(filter.Clauses))
	for _, clause := range filter.Clauses {
		if strings.TrimSpace(strings.ToLower(clause.Type)) != "rfm_group" {
			expandedClauses = append(expandedClauses, clause)
			continue
		}

		slug, ok := asString(segmentClauseParameter(clause, "slug"))
		if !ok {
			return analyticsdomain.SegmentFilter{}, domain.ErrInvalidFilter
		}
		group, exists := groupsBySlug[strings.ToLower(slug)]
		if !exists {
			return analyticsdomain.SegmentFilter{}, domain.ErrInvalidFilter
		}

		rangeParameters := rfmGroupConditionsToRangeParameters(group.Conditions)
		if len(rangeParameters) == 0 {
			if clause.Exclude {
				expandedClauses = append(expandedClauses, analyticsdomain.SegmentClause{
					Type:    "__always_true__",
					Exclude: true,
				})
			}
			continue
		}

		expandedClauses = append(expandedClauses, analyticsdomain.SegmentClause{
			Type:       "rfm_range",
			Exclude:    clause.Exclude,
			Parameters: rangeParameters,
		})
	}

	filter.Clauses = expandedClauses
	filter.RFMGroup = ""

	return filter, nil
}

// fetchRFMGroupsBySlug loads all RFM groups and indexes them by normalized slug.
func (s *SegmentService) fetchRFMGroupsBySlug(ctx context.Context) (map[string]analyticsdomain.RFMGroup, error) {
	rows, err := s.rfmGroupRepository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list rfm groups: %w", err)
	}

	result := make(map[string]analyticsdomain.RFMGroup, len(rows))
	for _, row := range rows {
		slug := strings.TrimSpace(strings.ToLower(row.Slug))
		if slug == "" {
			continue
		}
		result[slug] = row
	}

	return result, nil
}

// segmentClauseParameter resolves normalized clause parameter values.
func segmentClauseParameter(clause analyticsdomain.SegmentClause, key string) any {
	if len(clause.Parameters) == 0 {
		return nil
	}

	return clause.Parameters[strings.TrimSpace(key)]
}

// rfmGroupConditionsToRangeParameters maps group conditions into rfm_range clause parameters.
func rfmGroupConditionsToRangeParameters(conditions analyticsdomain.RFMGroupConditions) map[string]any {
	result := map[string]any{}
	if conditions.RMin != nil {
		result["rMin"] = *conditions.RMin
	}
	if conditions.RMax != nil {
		result["rMax"] = *conditions.RMax
	}
	if conditions.FMin != nil {
		result["fMin"] = *conditions.FMin
	}
	if conditions.FMax != nil {
		result["fMax"] = *conditions.FMax
	}
	if conditions.MMin != nil {
		result["mMin"] = *conditions.MMin
	}
	if conditions.MMax != nil {
		result["mMax"] = *conditions.MMax
	}
	if len(result) == 0 {
		return nil
	}

	return result
}

// normalizeFilters normalizes filters and removes empty types.
func normalizeFilters(filters []domain.Filter) []domain.Filter {
	normalized := make([]domain.Filter, 0, len(filters))
	for _, filter := range filters {
		trimmedType := strings.TrimSpace(filter.Type)
		if trimmedType == "" {
			continue
		}
		normalized = append(normalized, domain.Filter{
			Type:       trimmedType,
			Exclude:    filter.Exclude,
			Value:      filter.Value,
			Parameters: cloneParameters(filter.Parameters),
		})
	}

	return normalized
}

// toAnalyticsFilter maps segment filters into analytics filter payload values.
func toAnalyticsFilter(filters []domain.Filter) analyticsdomain.SegmentFilter {
	result := analyticsdomain.SegmentFilter{
		Clauses: make([]analyticsdomain.SegmentClause, 0, len(filters)),
	}
	for _, filter := range filters {
		filterType := strings.TrimSpace(strings.ToLower(filter.Type))
		if filterType != "" {
			result.Clauses = append(result.Clauses, analyticsdomain.SegmentClause{
				Type:       filterType,
				Exclude:    filter.Exclude,
				Value:      filter.Value,
				Parameters: cloneParameters(filter.Parameters),
			})
		}
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
			if skus, ok := asStringSlice(filterParameter(filter, "skus")); ok && len(skus) > 0 {
				result.PurchasedSKUs = skus
			} else if value, ok := filter.Value.(string); ok && strings.TrimSpace(value) != "" {
				result.PurchasedSKUs = []string{strings.TrimSpace(value)}
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
			if values, ok := asStringSlice(filterParameter(filter, "statuses")); ok && !filter.Exclude {
				result.OrderStatuses = values
			}
		case "rfm_group":
			if value, ok := asString(filterParameter(filter, "slug")); ok {
				result.RFMGroup = value
			}
		case "rfm_score":
			if value, ok := asInt(filterParameter(filter, "min")); ok {
				result.RFMScoreMin = &value
			}
			if value, ok := asInt(filterParameter(filter, "max")); ok {
				result.RFMScoreMax = &value
			}
		case "rfm_range":
			if v, ok := asInt(filterParameter(filter, "rMin")); ok {
				result.RFMRMin = &v
			}
			if v, ok := asInt(filterParameter(filter, "rMax")); ok {
				result.RFMRMax = &v
			}
			if v, ok := asInt(filterParameter(filter, "fMin")); ok {
				result.RFMFMin = &v
			}
			if v, ok := asInt(filterParameter(filter, "fMax")); ok {
				result.RFMFMax = &v
			}
			if v, ok := asInt(filterParameter(filter, "mMin")); ok {
				result.RFMMMin = &v
			}
			if v, ok := asInt(filterParameter(filter, "mMax")); ok {
				result.RFMMMax = &v
			}
		case "min_order_count":
			if value, ok := asInt(filterParameter(filter, "count")); ok && value > 0 {
				result.MinOrderCount = &value
			} else if value, ok := asInt(filter.Value); ok && value > 0 {
				result.MinOrderCount = &value
			}
		case "tag_affinity":
			if tags, ok := asAffinityTagFilters(filterParameter(filter, "tags")); ok {
				result.AffinityTags = tags
			}
		case "category_affinity":
			if cats, ok := asAffinityCategoryFilters(filterParameter(filter, "categories")); ok {
				result.AffinityCategories = cats
			}
		case "variation_affinity":
			if vars, ok := asAffinityVariationFilters(filterParameter(filter, "variations")); ok {
				result.AffinityVariations = vars
			}
		}
	}

	return result
}

// asAffinityTagFilters parses tag affinity filter slice values.
func asAffinityTagFilters(value any) ([]analyticsdomain.AffinityTagFilter, bool) {
	raw, ok := value.([]any)
	if !ok || len(raw) == 0 {
		return nil, false
	}
	result := make([]analyticsdomain.AffinityTagFilter, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		tag, _ := asString(m["tag"])
		if tag == "" {
			return nil, false
		}
		minScorePct, ok := asPercentage(m["minScorePct"])
		if !ok {
			return nil, false
		}
		relatedTags := []string{}
		if _, exists := m["relatedTags"]; exists {
			values, ok := asStringSlice(m["relatedTags"])
			if !ok {
				return nil, false
			}
			relatedTags = normalizeRelatedTags(tag, values)
		}
		result = append(result, analyticsdomain.AffinityTagFilter{Tag: tag, RelatedTags: relatedTags, MinScorePct: minScorePct})
	}
	if len(result) == 0 {
		return nil, false
	}

	return result, true
}

// asAffinityCategoryFilters parses category affinity filter slice values.
func asAffinityCategoryFilters(value any) ([]analyticsdomain.AffinityCategoryFilter, bool) {
	raw, ok := value.([]any)
	if !ok || len(raw) == 0 {
		return nil, false
	}
	result := make([]analyticsdomain.AffinityCategoryFilter, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		catID, _ := asString(m["categoryId"])
		if catID == "" {
			return nil, false
		}
		minScorePct, ok := asPercentage(m["minScorePct"])
		if !ok {
			return nil, false
		}
		result = append(result, analyticsdomain.AffinityCategoryFilter{CategoryID: catID, MinScorePct: minScorePct})
	}
	if len(result) == 0 {
		return nil, false
	}

	return result, true
}

// asAffinityVariationFilters parses variation affinity filter slice values.
func asAffinityVariationFilters(value any) ([]analyticsdomain.AffinityVariationFilter, bool) {
	raw, ok := value.([]any)
	if !ok || len(raw) == 0 {
		return nil, false
	}
	result := make([]analyticsdomain.AffinityVariationFilter, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		name, _ := asString(m["name"])
		val, _ := asString(m["value"])
		if name == "" || val == "" {
			return nil, false
		}
		minScorePct, ok := asPercentage(m["minScorePct"])
		if !ok {
			return nil, false
		}
		result = append(result, analyticsdomain.AffinityVariationFilter{Name: name, Value: val, MinScorePct: minScorePct})
	}
	if len(result) == 0 {
		return nil, false
	}

	return result, true
}

// validateFilters validates filter types and parameters.
func validateFilters(filters []domain.Filter) error {
	for _, filter := range filters {
		filterType := strings.TrimSpace(strings.ToLower(filter.Type))
		switch filterType {
		case "city_code_in", "min_total_spend", "email_opt_in":
		case "purchased_sku":
			skus, hasSkus := asStringSlice(filterParameter(filter, "skus"))
			_, hasLegacyValue := filter.Value.(string)
			if (!hasSkus || len(skus) == 0) && !hasLegacyValue {
				return domain.ErrInvalidFilter
			}
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
			continue
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
		case "rfm_group":
			if _, ok := asString(filterParameter(filter, "slug")); !ok {
				return domain.ErrInvalidFilter
			}
		case "rfm_score":
			_, hasMin := asInt(filterParameter(filter, "min"))
			_, hasMax := asInt(filterParameter(filter, "max"))
			if !hasMin && !hasMax {
				return domain.ErrInvalidFilter
			}
		case "rfm_range":
			hasAnyRange := false
			if _, ok := asInt(filterParameter(filter, "rMin")); ok {
				hasAnyRange = true
			}
			if _, ok := asInt(filterParameter(filter, "rMax")); ok {
				hasAnyRange = true
			}
			if _, ok := asInt(filterParameter(filter, "fMin")); ok {
				hasAnyRange = true
			}
			if _, ok := asInt(filterParameter(filter, "fMax")); ok {
				hasAnyRange = true
			}
			if _, ok := asInt(filterParameter(filter, "mMin")); ok {
				hasAnyRange = true
			}
			if _, ok := asInt(filterParameter(filter, "mMax")); ok {
				hasAnyRange = true
			}
			if !hasAnyRange {
				return domain.ErrInvalidFilter
			}
		case "min_order_count":
			hasCount := false
			if v, ok := asInt(filterParameter(filter, "count")); ok && v > 0 {
				hasCount = true
			} else if v, ok := asInt(filter.Value); ok && v > 0 {
				hasCount = true
				_ = v
			}
			if !hasCount {
				return domain.ErrInvalidFilter
			}
		case "tag_affinity":
			rows, ok := asAffinityTagFilters(filterParameter(filter, "tags"))
			if !ok || len(rows) == 0 {
				return domain.ErrInvalidFilter
			}
		case "category_affinity":
			rows, ok := asAffinityCategoryFilters(filterParameter(filter, "categories"))
			if !ok || len(rows) == 0 {
				return domain.ErrInvalidFilter
			}
		case "variation_affinity":
			rows, ok := asAffinityVariationFilters(filterParameter(filter, "variations"))
			if !ok || len(rows) == 0 {
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

// asPercentage converts filter values into [0,100] percentage values.
func asPercentage(value any) (float64, bool) {
	parsed, ok := asFloat64(value)
	if !ok || parsed < 0 || parsed > 100 {
		return 0, false
	}

	return parsed, true
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

// normalizeRelatedTags deduplicates related tags and removes the primary tag.
func normalizeRelatedTags(primary string, related []string) []string {
	normalizedPrimary := strings.ToLower(strings.TrimSpace(primary))
	seen := make(map[string]struct{}, len(related)+1)
	if normalizedPrimary != "" {
		seen[normalizedPrimary] = struct{}{}
	}
	result := make([]string, 0, len(related))
	for _, row := range related {
		trimmed := strings.TrimSpace(row)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, trimmed)
	}

	return result
}
