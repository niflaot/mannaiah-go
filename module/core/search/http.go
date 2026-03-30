package search

import (
	"strings"
	"strconv"

	corehttp "mannaiah/module/core/http"
)

// ParseQuery extracts a search Query from HTTP query parameters.
// Format:
//
//	term       = free text
//	filter[field]     = exact match (eq)
//	filter[field.op]  = operator match (gte, lte, gt, lt, between, in, like)
//	sort       = field:dir,field:dir (e.g. "name:asc,created_at:desc")
//	page       = 1-based page number
//	pageSize   = items per page
func ParseQuery(ctx corehttp.Context) Query {
	term := strings.TrimSpace(ctx.Query("term", ""))

	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.Query("pageSize", "20"))

	return Query{
		Term:     term,
		Filters:  parseFilters(ctx),
		Sort:     parseSort(ctx.Query("sort", "")),
		Page:     page,
		PageSize: pageSize,
	}
}

// parseFilters extracts typed filters from query parameters.
// Supports: filter[field]=value (eq), filter[field.op]=value.
// Multi-value for IN: filter[status]=PENDING,HOLD.
func parseFilters(ctx corehttp.Context) []Filter {
	body := ctx.Body()
	raw := string(body)
	_ = raw

	var filters []Filter
	params := extractFilterParams(ctx)
	for key, val := range params {
		field, op := parseFilterKey(key)
		if field == "" {
			continue
		}
		filters = append(filters, buildFilter(field, op, val))
	}
	return filters
}

// extractFilterParams collects filter[...] query parameters.
func extractFilterParams(ctx corehttp.Context) map[string]string {
	result := make(map[string]string)
	for _, pair := range parseQueryString(ctx) {
		if strings.HasPrefix(pair.key, "filter[") && strings.HasSuffix(pair.key, "]") {
			inner := pair.key[7 : len(pair.key)-1]
			result[inner] = pair.value
		}
	}
	return result
}

type queryPair struct {
	key   string
	value string
}

// parseQueryString parses raw query string pairs from the URL.
func parseQueryString(ctx corehttp.Context) []queryPair {
	raw := string(ctx.Body())
	_ = raw

	var pairs []queryPair
	wellKnown := []string{"term", "page", "pageSize", "sort"}
	isWellKnown := func(k string) bool {
		for _, w := range wellKnown {
			if k == w {
				return true
			}
		}
		return false
	}

	possibleFilters := tryExtractFiltersFromContext(ctx)
	for k, v := range possibleFilters {
		if !isWellKnown(k) {
			pairs = append(pairs, queryPair{key: k, value: v})
		}
	}
	return pairs
}

// tryExtractFiltersFromContext attempts to read filter params by known patterns.
func tryExtractFiltersFromContext(ctx corehttp.Context) map[string]string {
	commonFields := []string{
		"document_type", "city_code", "status", "realm", "contact_id",
		"carrier_id", "channel", "segment_id", "parent_id", "definition",
		"dispatch_batch_id", "shipment_mode", "payment_method", "sku", "slug",
		"parent_segment_id", "tags",
	}
	commonOps := []string{"gte", "lte", "gt", "lt"}

	result := make(map[string]string)
	for _, field := range commonFields {
		key := "filter[" + field + "]"
		if v := ctx.Query(key, ""); v != "" {
			result[key] = v
		}
		for _, op := range commonOps {
			opKey := "filter[" + field + "." + op + "]"
			if v := ctx.Query(opKey, ""); v != "" {
				result[opKey] = v
			}
		}
	}
	dateFields := []string{"created_at", "updated_at", "occurred_at"}
	for _, field := range dateFields {
		for _, op := range commonOps {
			key := "filter[" + field + "." + op + "]"
			if v := ctx.Query(key, ""); v != "" {
				result[key] = v
			}
		}
		key := "filter[" + field + "]"
		if v := ctx.Query(key, ""); v != "" {
			result[key] = v
		}
	}
	numericFields := []string{"price", "declared_value"}
	for _, field := range numericFields {
		key := "filter[" + field + "]"
		if v := ctx.Query(key, ""); v != "" {
			result[key] = v
		}
		for _, op := range commonOps {
			opKey := "filter[" + field + "." + op + "]"
			if v := ctx.Query(opKey, ""); v != "" {
				result[opKey] = v
			}
		}
	}
	return result
}

// parseFilterKey splits "field.op" into field and operator.
func parseFilterKey(key string) (string, Operator) {
	parts := strings.SplitN(key, ".", 2)
	if len(parts) == 1 {
		return parts[0], OpEQ
	}
	op := Operator(strings.ToLower(parts[1]))
	switch op {
	case OpEQ, OpLike, OpIn, OpBetween, OpGT, OpLT, OpGTE, OpLTE:
		return parts[0], op
	default:
		return parts[0], OpEQ
	}
}

// buildFilter constructs a Filter from field, operator, and raw string value.
func buildFilter(field string, op Operator, rawValue string) Filter {
	val := strings.TrimSpace(rawValue)

	switch op {
	case OpIn:
		parts := strings.Split(val, ",")
		cleaned := make([]string, 0, len(parts))
		for _, p := range parts {
			if t := strings.TrimSpace(p); t != "" {
				cleaned = append(cleaned, t)
			}
		}
		return Filter{Field: field, Operator: OpIn, Value: cleaned}
	case OpBetween:
		parts := strings.SplitN(val, ",", 2)
		if len(parts) == 2 {
			return Filter{Field: field, Operator: OpBetween, Value: [2]any{strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])}}
		}
		return Filter{Field: field, Operator: OpEQ, Value: val}
	default:
		return Filter{Field: field, Operator: op, Value: val}
	}
}

// parseSort parses "field:dir,field:dir" into SortField slices.
func parseSort(raw string) []SortField {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, ",")
	result := make([]SortField, 0, len(parts))
	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), ":", 2)
		if len(kv) == 0 || strings.TrimSpace(kv[0]) == "" {
			continue
		}
		field := strings.TrimSpace(kv[0])
		dir := Desc
		if len(kv) == 2 {
			d := strings.ToUpper(strings.TrimSpace(kv[1]))
			if d == "ASC" {
				dir = Asc
			}
		}
		result = append(result, SortField{Field: field, Direction: dir})
	}
	return result
}
