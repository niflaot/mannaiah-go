package clickhouse

import (
	"strings"

	"mannaiah/module/analytics/domain"
)

// appendTagAffinityCondition appends one EXISTS filter per tag affinity constraint (ANDed).
func appendTagAffinityCondition(conditions *[]string, args *[]any, filter domain.SegmentFilter) {
	for _, af := range filter.AffinityTags {
		tag := strings.TrimSpace(af.Tag)
		if tag == "" {
			continue
		}
		*conditions = append(*conditions, `EXISTS (
			SELECT 1 FROM (
				SELECT contact_id, tag, sum(affinity_score) AS score
				FROM tag_affinity_mv FINAL
				WHERE tag = ?
				GROUP BY contact_id, tag
			) ta
			WHERE ta.contact_id = cs.contact_id AND ta.score >= ?
		)`)
		*args = append(*args, tag, af.MinScore)
	}
}

// appendCategoryAffinityCondition appends one EXISTS filter per category affinity constraint (ANDed).
func appendCategoryAffinityCondition(conditions *[]string, args *[]any, filter domain.SegmentFilter) {
	for _, af := range filter.AffinityCategories {
		catID := strings.TrimSpace(af.CategoryID)
		if catID == "" {
			continue
		}
		*conditions = append(*conditions, `EXISTS (
			SELECT 1 FROM (
				SELECT contact_id, category_id, sum(affinity_score) AS score
				FROM category_affinity_mv FINAL
				WHERE category_id = ?
				GROUP BY contact_id, category_id
			) ca
			WHERE ca.contact_id = cs.contact_id AND ca.score >= ?
		)`)
		*args = append(*args, catID, af.MinScore)
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
		*conditions = append(*conditions, `EXISTS (
			SELECT 1 FROM (
				SELECT contact_id, variation_name, variation_value, sum(affinity_score) AS score
				FROM variation_affinity_mv FINAL
				WHERE variation_name = ? AND variation_value = ?
				GROUP BY contact_id, variation_name, variation_value
			) va
			WHERE va.contact_id = cs.contact_id AND va.score >= ?
		)`)
		*args = append(*args, name, value, af.MinScore)
	}
}
