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
			SELECT contact_id FROM (
				SELECT contact_id, tag, score, max(score) OVER (PARTITION BY contact_id) AS max_score
				FROM (
					SELECT contact_id, tag, sum(affinity_score) AS score
					FROM tag_affinity_mv FINAL
					GROUP BY contact_id, tag
				)
			) ta
			WHERE ta.tag IN (`+makePlaceholders(len(tags))+`) AND if(ta.max_score = 0, 0, (ta.score * 100.0 / ta.max_score)) >= ?
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
			SELECT contact_id FROM (
				SELECT contact_id, category_id, score, max(score) OVER (PARTITION BY contact_id) AS max_score
				FROM (
					SELECT contact_id, category_id, sum(affinity_score) AS score
					FROM category_affinity_mv FINAL
					GROUP BY contact_id, category_id
				)
			) ca
			WHERE ca.category_id = ? AND if(ca.max_score = 0, 0, (ca.score * 100.0 / ca.max_score)) >= ?
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
			SELECT contact_id FROM (
				SELECT contact_id, variation_name, variation_value, score, max(score) OVER (PARTITION BY contact_id) AS max_score
				FROM (
					SELECT contact_id, variation_name, variation_value, sum(affinity_score) AS score
					FROM variation_affinity_mv FINAL
					GROUP BY contact_id, variation_name, variation_value
				)
			) va
			WHERE va.variation_name = ? AND va.variation_value = ? AND if(va.max_score = 0, 0, (va.score * 100.0 / va.max_score)) >= ?
		)`)
		*args = append(*args, name, value, af.MinScorePct)
	}
}
