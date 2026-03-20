package clickhouse

import (
	"strings"

	"mannaiah/module/analytics/domain"
)

// appendTagAffinityCondition appends one EXISTS filter per tag affinity constraint (ANDed).
func appendTagAffinityCondition(conditions *[]string, args *[]any, filter domain.SegmentFilter) {
	for _, af := range filter.AffinityTags {
		tags := affinityTagScope(af)
		if len(tags) == 0 {
			continue
		}
		*conditions = append(*conditions, `cs.contact_id IN (
			SELECT contact_id
			FROM (
				SELECT contact_id, tag, sum(affinity_score) AS score
				FROM tag_affinity_mv FINAL
				GROUP BY contact_id, tag
			)
			GROUP BY contact_id
			HAVING maxIf(score, tag IN (`+makePlaceholders(len(tags))+`)) * 100.0 / nullIf(max(score), 0) >= ?
		)`)
		for _, tag := range tags {
			*args = append(*args, strings.TrimSpace(tag))
		}
		*args = append(*args, af.MinScorePct)
	}
}

// appendCategoryAffinityCondition appends one EXISTS filter per category affinity constraint (ANDed).
func appendCategoryAffinityCondition(conditions *[]string, args *[]any, filter domain.SegmentFilter) {
	for _, af := range filter.AffinityCategories {
		catID := strings.TrimSpace(af.CategoryID)
		if catID == "" {
			continue
		}
		*conditions = append(*conditions, `cs.contact_id IN (
			SELECT contact_id
			FROM (
				SELECT contact_id, category_id, sum(affinity_score) AS score
				FROM category_affinity_mv FINAL
				GROUP BY contact_id, category_id
			)
			GROUP BY contact_id
			HAVING maxIf(score, category_id = ?) * 100.0 / nullIf(max(score), 0) >= ?
		)`)
		*args = append(*args, catID, af.MinScorePct)
	}
}

// appendVariationAffinityCondition appends one EXISTS filter per variation affinity constraint (ANDed).
func appendVariationAffinityCondition(conditions *[]string, args *[]any, filter domain.SegmentFilter) {
	for _, af := range filter.AffinityVariations {
		name := strings.TrimSpace(af.Name)
		value := strings.TrimSpace(af.Value)
		if name == "" || value == "" {
			continue
		}
		*conditions = append(*conditions, `cs.contact_id IN (
			SELECT contact_id
			FROM (
				SELECT contact_id, variation_name, variation_value, sum(affinity_score) AS score
				FROM variation_affinity_mv FINAL
				GROUP BY contact_id, variation_name, variation_value
			)
			GROUP BY contact_id
			HAVING maxIf(score, variation_name = ? AND variation_value = ?) * 100.0 / nullIf(max(score), 0) >= ?
		)`)
		*args = append(*args, name, value, af.MinScorePct)
	}
}
