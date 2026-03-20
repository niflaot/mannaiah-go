package clickhouse

import (
	"strconv"
	"strings"

	"mannaiah/module/analytics/domain"
)

// buildSegmentWhereFromClauses constructs a WHERE clause from raw DSL clauses with include/exclude support.
func buildSegmentWhereFromClauses(filter domain.SegmentFilter, topSpenderIDs []string) (string, []any) {
	conditions := []string{"1 = 1"}
	args := make([]any, 0, 32)
	includedStatuses, excludedStatuses := collectOrderStatusScopes(filter)

	for _, clause := range filter.Clauses {
		clauseType := strings.TrimSpace(strings.ToLower(clause.Type))
		if clauseType == "" || clauseType == "order_status" {
			continue
		}

		condition, clauseArgs := buildClauseCondition(clause, includedStatuses, excludedStatuses, topSpenderIDs)
		if condition == "" {
			continue
		}
		if clause.Exclude {
			condition = "NOT (" + condition + ")"
		}

		conditions = append(conditions, condition)
		args = append(args, clauseArgs...)
	}

	return strings.Join(conditions, " AND "), args
}

// collectOrderStatusScopes resolves include and exclude status scopes from order_status clauses.
func collectOrderStatusScopes(filter domain.SegmentFilter) ([]string, []string) {
	if len(filter.Clauses) == 0 {
		return append([]string{}, filter.OrderStatuses...), nil
	}

	included := make([]string, 0, 4)
	excluded := make([]string, 0, 4)
	seenIncluded := map[string]struct{}{}
	seenExcluded := map[string]struct{}{}

	for _, clause := range filter.Clauses {
		if strings.TrimSpace(strings.ToLower(clause.Type)) != "order_status" {
			continue
		}
		statuses, ok := clauseAsStringSlice(clauseParameter(clause, "statuses"))
		if !ok || len(statuses) == 0 {
			continue
		}
		if clause.Exclude {
			for _, status := range statuses {
				trimmed := strings.TrimSpace(status)
				if trimmed == "" {
					continue
				}
				if _, exists := seenExcluded[trimmed]; exists {
					continue
				}
				seenExcluded[trimmed] = struct{}{}
				excluded = append(excluded, trimmed)
			}
			continue
		}
		for _, status := range statuses {
			trimmed := strings.TrimSpace(status)
			if trimmed == "" {
				continue
			}
			if _, exists := seenIncluded[trimmed]; exists {
				continue
			}
			seenIncluded[trimmed] = struct{}{}
			included = append(included, trimmed)
		}
	}

	return included, excluded
}

// buildClauseCondition builds one SQL clause and positional args for one filter clause.
func buildClauseCondition(clause domain.SegmentClause, includedStatuses []string, excludedStatuses []string, topSpenderIDs []string) (string, []any) {
	clauseType := strings.TrimSpace(strings.ToLower(clause.Type))

	switch clauseType {
	case "__always_true__":
		return "1 = 1", nil
	case "city_code_in":
		codes, ok := clauseAsStringSlice(clause.Value)
		if !ok || len(codes) == 0 {
			return "", nil
		}

		args := make([]any, 0, len(codes))
		for _, code := range codes {
			args = append(args, strings.TrimSpace(code))
		}

		return "cs.city_code IN (" + makePlaceholders(len(codes)) + ")", args
	case "city":
		codes, ok := clauseAsStringSlice(clauseParameter(clause, "codes"))
		if !ok || len(codes) == 0 {
			return "", nil
		}

		args := make([]any, 0, len(codes))
		for _, code := range codes {
			args = append(args, strings.TrimSpace(code))
		}

		return "cs.city_code IN (" + makePlaceholders(len(codes)) + ")", args
	case "min_total_spend":
		minSpend, ok := clauseAsFloat64(clause.Value)
		if !ok {
			return "", nil
		}

		condition := `EXISTS (
		SELECT 1 FROM orders_fact of FINAL
		WHERE of.contact_id = cs.contact_id` + orderStatusFragmentWithExclusions(includedStatuses, excludedStatuses, "of") + `
		GROUP BY of.contact_id
		HAVING sum(of.total_value) >= ?
	)`
		args := appendOrderStatusArgsWithExclusions(make([]any, 0, len(includedStatuses)+len(excludedStatuses)+1), includedStatuses, excludedStatuses)
		args = append(args, minSpend)

		return condition, args
	case "email_opt_in":
		required, ok := clauseAsBool(clause.Value)
		if !ok {
			return "", nil
		}
		action := "opt_out"
		if required {
			action = "opt_in"
		}

		return `EXISTS (
		SELECT 1 FROM (
			SELECT contact_id, argMax(action, occurred_at) AS latest_action
			FROM membership_events
			WHERE channel = 'email'
			GROUP BY contact_id
		) ms
		WHERE ms.contact_id = cs.contact_id AND ms.latest_action = ?
	)`, []any{action}
	case "purchased_sku":
		skus, ok := clauseAsStringSlice(clauseParameter(clause, "skus"))
		if (!ok || len(skus) == 0) && strings.TrimSpace(clauseAsStringValue(clause.Value)) != "" {
			skus = []string{strings.TrimSpace(clauseAsStringValue(clause.Value))}
		}
		if len(skus) == 0 {
			return "", nil
		}

		orderStatusNested := ""
		if len(includedStatuses) > 0 || len(excludedStatuses) > 0 {
			orderStatusNested = `
		AND EXISTS (
			SELECT 1 FROM orders_fact of FINAL
			WHERE of.order_id = oi.order_id` + orderStatusFragmentWithExclusions(includedStatuses, excludedStatuses, "of") + `
		)`
		}

		condition := `EXISTS (
		SELECT 1 FROM order_items_fact oi FINAL
		WHERE oi.contact_id = cs.contact_id AND oi.sku IN (` + makePlaceholders(len(skus)) + `)` + orderStatusNested + `
	)`
		args := make([]any, 0, len(skus)+len(includedStatuses)+len(excludedStatuses))
		for _, sku := range skus {
			args = append(args, strings.TrimSpace(sku))
		}
		args = appendOrderStatusArgsWithExclusions(args, includedStatuses, excludedStatuses)

		return condition, args
	case "order_recency":
		days, ok := clauseAsInt(clauseParameter(clause, "days"))
		if !ok || days <= 0 {
			return "", nil
		}

		condition := `EXISTS (
		SELECT 1 FROM orders_fact of FINAL
		WHERE of.contact_id = cs.contact_id
		  AND of.created_at >= (now64(3) - toIntervalDay(?))` + orderStatusFragmentWithExclusions(includedStatuses, excludedStatuses, "of") + `
	)`
		args := append([]any{days}, appendOrderStatusArgsWithExclusions(make([]any, 0, len(includedStatuses)+len(excludedStatuses)), includedStatuses, excludedStatuses)...)

		return condition, args
	case "no_order_recency":
		days, ok := clauseAsInt(clauseParameter(clause, "days"))
		if !ok || days <= 0 {
			return "", nil
		}

		condition := `NOT EXISTS (
		SELECT 1 FROM orders_fact of FINAL
		WHERE of.contact_id = cs.contact_id
		  AND of.created_at >= (now64(3) - toIntervalDay(?))` + orderStatusFragmentWithExclusions(includedStatuses, excludedStatuses, "of") + `
	)`
		args := append([]any{days}, appendOrderStatusArgsWithExclusions(make([]any, 0, len(includedStatuses)+len(excludedStatuses)), includedStatuses, excludedStatuses)...)

		return condition, args
	case "category":
		pattern, ok := clauseAsString(clauseParameter(clause, "pattern"))
		if !ok {
			return "", nil
		}

		orderStatusNested := ""
		if len(includedStatuses) > 0 || len(excludedStatuses) > 0 {
			orderStatusNested = `
		AND EXISTS (
			SELECT 1 FROM orders_fact of FINAL
			WHERE of.order_id = oi.order_id` + orderStatusFragmentWithExclusions(includedStatuses, excludedStatuses, "of") + `
		)`
		}

		condition := `EXISTS (
		SELECT 1 FROM order_items_fact oi FINAL
		WHERE oi.contact_id = cs.contact_id AND (lower(oi.sku) LIKE lower(?) OR lower(oi.alternate_name) LIKE lower(?))` + orderStatusNested + `
	)`
		args := []any{"%" + pattern + "%", "%" + pattern + "%"}
		args = appendOrderStatusArgsWithExclusions(args, includedStatuses, excludedStatuses)

		return condition, args
	case "top_spenders":
		if topSpenderIDs == nil {
			return "", nil
		}
		if len(topSpenderIDs) == 0 {
			return "1 = 0", nil
		}

		args := make([]any, 0, len(topSpenderIDs))
		for _, contactID := range topSpenderIDs {
			args = append(args, strings.TrimSpace(contactID))
		}

		return "cs.contact_id IN (" + makePlaceholders(len(topSpenderIDs)) + ")", args
	case "first_purchase_only":
		enabled := true
		if value, ok := clauseAsBool(clauseParameter(clause, "enabled")); ok {
			enabled = value
		}
		if !enabled {
			return "", nil
		}

		condition := `EXISTS (
		SELECT 1 FROM orders_fact of FINAL
		WHERE of.contact_id = cs.contact_id` + orderStatusFragmentWithExclusions(includedStatuses, excludedStatuses, "of") + `
		GROUP BY of.contact_id
		HAVING countDistinct(of.order_id) = 1
	)`
		args := appendOrderStatusArgsWithExclusions(make([]any, 0, len(includedStatuses)+len(excludedStatuses)), includedStatuses, excludedStatuses)

		return condition, args
	case "subscribed_no_buy":
		enabled := true
		if value, ok := clauseAsBool(clauseParameter(clause, "enabled")); ok {
			enabled = value
		}
		if !enabled {
			return "", nil
		}

		condition := `(EXISTS (
		SELECT 1 FROM (
			SELECT contact_id, argMax(action, occurred_at) AS latest_action
			FROM membership_events
			WHERE channel = 'email'
			GROUP BY contact_id
		) ms
		WHERE ms.contact_id = cs.contact_id AND ms.latest_action = 'opt_in'
	) AND NOT EXISTS (
		SELECT 1 FROM orders_fact of FINAL
		WHERE of.contact_id = cs.contact_id` + orderStatusFragmentWithExclusions(includedStatuses, excludedStatuses, "of") + `
	))`
		args := appendOrderStatusArgsWithExclusions(make([]any, 0, len(includedStatuses)+len(excludedStatuses)), includedStatuses, excludedStatuses)

		return condition, args
	case "opt_in_status":
		channel, ok := clauseAsString(clauseParameter(clause, "channel"))
		if !ok {
			return "", nil
		}
		action, ok := clauseAsString(clauseParameter(clause, "status"))
		if !ok {
			action = "opt_in"
		}

		return `EXISTS (
		SELECT 1 FROM (
			SELECT contact_id, argMax(action, occurred_at) AS latest_action
			FROM membership_events
			WHERE channel = ?
			GROUP BY contact_id
		) ms
		WHERE ms.contact_id = cs.contact_id AND ms.latest_action = ?
	)`, []any{channel, action}
	case "metadata":
		key, ok := clauseAsString(clauseParameter(clause, "key"))
		if !ok {
			return "", nil
		}
		value, hasValue := clauseAsString(clauseParameter(clause, "value"))
		if !hasValue {
			return "JSONExtractString(cs.metadata_json, ?) != ''", []any{key}
		}

		return "JSONExtractString(cs.metadata_json, ?) = ?", []any{key, value}
	case "rfm_score":
		minScore, hasMin := clauseAsInt(clauseParameter(clause, "min"))
		maxScore, hasMax := clauseAsInt(clauseParameter(clause, "max"))
		if !hasMin && !hasMax {
			return "", nil
		}

		subWhere := "contact_id = cs.contact_id"
		args := make([]any, 0, 2)
		if hasMin {
			subWhere += " AND (r_score + f_score + m_score) >= ?"
			args = append(args, minScore)
		}
		if hasMax {
			subWhere += " AND (r_score + f_score + m_score) <= ?"
			args = append(args, maxScore)
		}

		return "EXISTS (SELECT 1 FROM rfm_scores_computed_v WHERE " + subWhere + ")", args
	case "rfm_range":
		rMin, hasRMin := clauseAsInt(clauseParameter(clause, "rMin"))
		rMax, hasRMax := clauseAsInt(clauseParameter(clause, "rMax"))
		fMin, hasFMin := clauseAsInt(clauseParameter(clause, "fMin"))
		fMax, hasFMax := clauseAsInt(clauseParameter(clause, "fMax"))
		mMin, hasMMin := clauseAsInt(clauseParameter(clause, "mMin"))
		mMax, hasMMax := clauseAsInt(clauseParameter(clause, "mMax"))
		if !hasRMin && !hasRMax && !hasFMin && !hasFMax && !hasMMin && !hasMMax {
			return "", nil
		}

		subWhere := "contact_id = cs.contact_id"
		args := make([]any, 0, 6)
		if hasRMin {
			subWhere += " AND r_score >= ?"
			args = append(args, rMin)
		}
		if hasRMax {
			subWhere += " AND r_score <= ?"
			args = append(args, rMax)
		}
		if hasFMin {
			subWhere += " AND f_score >= ?"
			args = append(args, fMin)
		}
		if hasFMax {
			subWhere += " AND f_score <= ?"
			args = append(args, fMax)
		}
		if hasMMin {
			subWhere += " AND m_score >= ?"
			args = append(args, mMin)
		}
		if hasMMax {
			subWhere += " AND m_score <= ?"
			args = append(args, mMax)
		}

		return "EXISTS (SELECT 1 FROM rfm_scores_computed_v WHERE " + subWhere + ")", args
	case "min_order_count":
		count, ok := clauseAsInt(clauseParameter(clause, "count"))
		if !ok || count <= 0 {
			count, ok = clauseAsInt(clause.Value)
		}
		if !ok || count <= 0 {
			return "", nil
		}

		condition := `EXISTS (
		SELECT 1 FROM orders_fact of FINAL
		WHERE of.contact_id = cs.contact_id` + orderStatusFragmentWithExclusions(includedStatuses, excludedStatuses, "of") + `
		GROUP BY of.contact_id
		HAVING countDistinct(of.order_id) >= ?
	)`
		args := appendOrderStatusArgsWithExclusions(make([]any, 0, len(includedStatuses)+len(excludedStatuses)+1), includedStatuses, excludedStatuses)
		args = append(args, count)

		return condition, args
	case "tag_affinity":
		rows := clauseAffinityTagFilters(clauseParameter(clause, "tags"))
		if len(rows) == 0 {
			return "", nil
		}

		parts := make([]string, 0, len(rows))
		args := make([]any, 0, len(rows)*4)
		for _, row := range rows {
			tags := affinityTagScope(row)
			if len(tags) == 0 {
				continue
			}
			parts = append(parts, `EXISTS (
			SELECT 1 FROM (
				SELECT ta.tag, ta.score, tm.max_score
				FROM (
					SELECT tag, sum(affinity_score) AS score
					FROM tag_affinity_mv FINAL
					WHERE contact_id = cs.contact_id
					GROUP BY tag
				) ta
				CROSS JOIN (
					SELECT max(score) AS max_score
					FROM (
						SELECT sum(affinity_score) AS score
						FROM tag_affinity_mv FINAL
						WHERE contact_id = cs.contact_id
						GROUP BY tag
					)
				) tm
			) ta
			WHERE ta.tag IN (`+makePlaceholders(len(tags))+`) AND if(ta.max_score = 0, 0, (ta.score * 100.0 / ta.max_score)) >= ?
		)`)
			for _, tag := range tags {
				args = append(args, tag)
			}
			args = append(args, row.MinScorePct)
		}
		if len(parts) == 0 {
			return "", nil
		}

		return "(" + strings.Join(parts, " AND ") + ")", args
	case "category_affinity":
		rows := clauseAffinityCategoryFilters(clauseParameter(clause, "categories"))
		if len(rows) == 0 {
			return "", nil
		}

		parts := make([]string, 0, len(rows))
		args := make([]any, 0, len(rows)*2)
		for _, row := range rows {
			parts = append(parts, `EXISTS (
			SELECT 1 FROM (
				SELECT ca.category_id, ca.score, cm.max_score
				FROM (
					SELECT category_id, sum(affinity_score) AS score
					FROM category_affinity_mv FINAL
					WHERE contact_id = cs.contact_id
					GROUP BY category_id
				) ca
				CROSS JOIN (
					SELECT max(score) AS max_score
					FROM (
						SELECT sum(affinity_score) AS score
						FROM category_affinity_mv FINAL
						WHERE contact_id = cs.contact_id
						GROUP BY category_id
					)
				) cm
			) ca
			WHERE ca.category_id = ? AND if(ca.max_score = 0, 0, (ca.score * 100.0 / ca.max_score)) >= ?
		)`)
			args = append(args, row.CategoryID, row.MinScorePct)
		}

		return "(" + strings.Join(parts, " AND ") + ")", args
	case "variation_affinity":
		rows := clauseAffinityVariationFilters(clauseParameter(clause, "variations"))
		if len(rows) == 0 {
			return "", nil
		}

		parts := make([]string, 0, len(rows))
		args := make([]any, 0, len(rows)*3)
		for _, row := range rows {
			parts = append(parts, `EXISTS (
			SELECT 1 FROM (
				SELECT va.variation_name, va.variation_value, va.score, vm.max_score
				FROM (
					SELECT variation_name, variation_value, sum(affinity_score) AS score
					FROM variation_affinity_mv FINAL
					WHERE contact_id = cs.contact_id
					GROUP BY variation_name, variation_value
				) va
				CROSS JOIN (
					SELECT max(score) AS max_score
					FROM (
						SELECT sum(affinity_score) AS score
						FROM variation_affinity_mv FINAL
						WHERE contact_id = cs.contact_id
						GROUP BY variation_name, variation_value
					)
				) vm
			) va
			WHERE va.variation_name = ? AND va.variation_value = ? AND if(va.max_score = 0, 0, (va.score * 100.0 / va.max_score)) >= ?
		)`)
			args = append(args, row.Name, row.Value, row.MinScorePct)
		}

		return "(" + strings.Join(parts, " AND ") + ")", args
	default:
		return "", nil
	}
}

// orderStatusFragmentWithExclusions builds status SQL fragments for include and exclude status scopes.
func orderStatusFragmentWithExclusions(included []string, excluded []string, alias string) string {
	fragment := ""
	if len(included) > 0 {
		fragment += orderStatusFragment(included, alias)
	}
	if len(excluded) > 0 {
		column := "current_status"
		if alias != "" {
			column = alias + ".current_status"
		}
		fragment += " AND " + column + " NOT IN (" + makePlaceholders(len(excluded)) + ")"
	}

	return fragment
}

// appendOrderStatusArgsWithExclusions appends include and exclude status args in SQL placeholder order.
func appendOrderStatusArgsWithExclusions(args []any, included []string, excluded []string) []any {
	args = appendOrderStatusArgs(args, included)
	for _, status := range excluded {
		args = append(args, strings.TrimSpace(status))
	}

	return args
}

// clauseParameter resolves one clause parameter value.
func clauseParameter(clause domain.SegmentClause, key string) any {
	if len(clause.Parameters) == 0 {
		return nil
	}

	return clause.Parameters[strings.TrimSpace(key)]
}

// clauseAsString converts clause values to non-empty trimmed string values.
func clauseAsString(value any) (string, bool) {
	text, ok := value.(string)
	if !ok {
		return "", false
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", false
	}

	return trimmed, true
}

// clauseAsStringValue converts clause values to raw trimmed string values.
func clauseAsStringValue(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(text)
}

// clauseAsStringSlice converts clause values to trimmed non-empty string slices.
func clauseAsStringSlice(value any) ([]string, bool) {
	switch typed := value.(type) {
	case []any:
		result := make([]string, 0, len(typed))
		for _, row := range typed {
			text, ok := row.(string)
			if !ok {
				continue
			}
			trimmed := strings.TrimSpace(text)
			if trimmed == "" {
				continue
			}
			result = append(result, trimmed)
		}

		return result, true
	case []string:
		result := make([]string, 0, len(typed))
		for _, row := range typed {
			trimmed := strings.TrimSpace(row)
			if trimmed == "" {
				continue
			}
			result = append(result, trimmed)
		}

		return result, true
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return []string{}, true
		}
		parts := strings.Split(trimmed, ",")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			entry := strings.TrimSpace(part)
			if entry == "" {
				continue
			}
			result = append(result, entry)
		}

		return result, true
	default:
		return nil, false
	}
}

// clauseAsFloat64 converts clause values to float64 values.
func clauseAsFloat64(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

// clauseAsInt converts clause values to integer values.
func clauseAsInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	case float32:
		return int(typed), true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

// clauseAsBool converts clause values to boolean values.
func clauseAsBool(value any) (bool, bool) {
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

// clauseAffinityTagFilters parses tag-affinity clause payloads.
func clauseAffinityTagFilters(value any) []domain.AffinityTagFilter {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}

	result := make([]domain.AffinityTagFilter, 0, len(raw))
	for _, item := range raw {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		tag, ok := clauseAsString(row["tag"])
		if !ok {
			continue
		}
		minScorePct, ok := clauseAsFloat64(row["minScorePct"])
		if !ok || minScorePct < 0 || minScorePct > 100 {
			continue
		}
		relatedTags := []string{}
		if rawRelated, exists := row["relatedTags"]; exists {
			values, ok := clauseAsStringSlice(rawRelated)
			if !ok {
				continue
			}
			relatedTags = normalizeRelatedTags(tag, values)
		}
		result = append(result, domain.AffinityTagFilter{Tag: tag, RelatedTags: relatedTags, MinScorePct: minScorePct})
	}

	return result
}

// clauseAffinityCategoryFilters parses category-affinity clause payloads.
func clauseAffinityCategoryFilters(value any) []domain.AffinityCategoryFilter {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}

	result := make([]domain.AffinityCategoryFilter, 0, len(raw))
	for _, item := range raw {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		categoryID, ok := clauseAsString(row["categoryId"])
		if !ok {
			continue
		}
		minScorePct, ok := clauseAsFloat64(row["minScorePct"])
		if !ok || minScorePct < 0 || minScorePct > 100 {
			continue
		}
		result = append(result, domain.AffinityCategoryFilter{CategoryID: categoryID, MinScorePct: minScorePct})
	}

	return result
}

// clauseAffinityVariationFilters parses variation-affinity clause payloads.
func clauseAffinityVariationFilters(value any) []domain.AffinityVariationFilter {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}

	result := make([]domain.AffinityVariationFilter, 0, len(raw))
	for _, item := range raw {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, ok := clauseAsString(row["name"])
		if !ok {
			continue
		}
		val, ok := clauseAsString(row["value"])
		if !ok {
			continue
		}
		minScorePct, ok := clauseAsFloat64(row["minScorePct"])
		if !ok || minScorePct < 0 || minScorePct > 100 {
			continue
		}
		result = append(result, domain.AffinityVariationFilter{Name: name, Value: val, MinScorePct: minScorePct})
	}

	return result
}

// affinityTagScope returns one deduplicated tag scope including primary and related tags.
func affinityTagScope(filter domain.AffinityTagFilter) []string {
	scope := make([]string, 0, len(filter.RelatedTags)+1)
	primary := strings.TrimSpace(filter.Tag)
	if primary != "" {
		scope = append(scope, primary)
	}
	scope = append(scope, filter.RelatedTags...)

	return normalizeRelatedTags("", scope)
}

// normalizeRelatedTags deduplicates tag values and removes blank entries.
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
