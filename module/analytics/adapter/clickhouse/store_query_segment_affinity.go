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
		*conditions = append(*conditions, `EXISTS (
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
		*conditions = append(*conditions, `EXISTS (
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
		*conditions = append(*conditions, `EXISTS (
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
		*args = append(*args, name, value, af.MinScorePct)
	}
}
