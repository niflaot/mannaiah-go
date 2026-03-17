package clickhouse

import (
	"mannaiah/module/analytics/domain"
)

// appendRFMScoreRangeCondition appends an RFM total score range EXISTS filter.
func appendRFMScoreRangeCondition(conditions *[]string, args *[]any, filter domain.SegmentFilter) {
	if filter.RFMScoreMin == nil && filter.RFMScoreMax == nil {
		return
	}

	subWhere := "contact_id = cs.contact_id"
	if filter.RFMScoreMin != nil {
		subWhere += " AND (r_score + f_score + m_score) >= ?"
		*args = append(*args, *filter.RFMScoreMin)
	}
	if filter.RFMScoreMax != nil {
		subWhere += " AND (r_score + f_score + m_score) <= ?"
		*args = append(*args, *filter.RFMScoreMax)
	}

	*conditions = append(*conditions, "EXISTS (SELECT 1 FROM rfm_scores_computed_v WHERE "+subWhere+")")
}

// appendRFMRangeCondition appends individual R/F/M band score range EXISTS filters.
func appendRFMRangeCondition(conditions *[]string, args *[]any, filter domain.SegmentFilter) {
	hasRFM := filter.RFMRMin != nil || filter.RFMRMax != nil ||
		filter.RFMFMin != nil || filter.RFMFMax != nil ||
		filter.RFMMMin != nil || filter.RFMMMax != nil
	if !hasRFM {
		return
	}

	subWhere := "contact_id = cs.contact_id"
	if filter.RFMRMin != nil {
		subWhere += " AND r_score >= ?"
		*args = append(*args, *filter.RFMRMin)
	}
	if filter.RFMRMax != nil {
		subWhere += " AND r_score <= ?"
		*args = append(*args, *filter.RFMRMax)
	}
	if filter.RFMFMin != nil {
		subWhere += " AND f_score >= ?"
		*args = append(*args, *filter.RFMFMin)
	}
	if filter.RFMFMax != nil {
		subWhere += " AND f_score <= ?"
		*args = append(*args, *filter.RFMFMax)
	}
	if filter.RFMMMin != nil {
		subWhere += " AND m_score >= ?"
		*args = append(*args, *filter.RFMMMin)
	}
	if filter.RFMMMax != nil {
		subWhere += " AND m_score <= ?"
		*args = append(*args, *filter.RFMMMax)
	}

	*conditions = append(*conditions, "EXISTS (SELECT 1 FROM rfm_scores_computed_v WHERE "+subWhere+")")
}
