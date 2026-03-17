package clickhouse

import (
	"strings"

	"mannaiah/module/analytics/domain"
)

// buildSegmentWhere constructs a parameterised WHERE clause from segment filter values.
func buildSegmentWhere(filter domain.SegmentFilter, topSpenderIDs []string) (string, []any) {
	conditions := []string{"1 = 1"}
	args := make([]any, 0, 16)

	appendCityCodeCondition(&conditions, &args, filter)
	appendMembershipCondition(&conditions, &args, filter)
	appendOptInChannelCondition(&conditions, &args, filter)
	appendMinSpendCondition(&conditions, &args, filter)
	appendPurchasedSKUCondition(&conditions, &args, filter)
	appendCategoryCondition(&conditions, &args, filter)
	appendOrderRecencyCondition(&conditions, &args, filter)
	appendNoOrderRecencyCondition(&conditions, &args, filter)
	appendFirstPurchaseCondition(&conditions, &args, filter)
	appendSubscribedNoBuyCondition(&conditions, &args, filter)
	appendTopSpenderCondition(&conditions, &args, topSpenderIDs)
	appendMetadataCondition(&conditions, &args, filter)
	appendRFMScoreRangeCondition(&conditions, &args, filter)
	appendRFMRangeCondition(&conditions, &args, filter)
	appendTagAffinityCondition(&conditions, &args, filter)
	appendCategoryAffinityCondition(&conditions, &args, filter)
	appendVariationAffinityCondition(&conditions, &args, filter)

	return strings.Join(conditions, " AND "), args
}

// appendCityCodeCondition appends a city code IN filter when city codes are set.
func appendCityCodeCondition(conditions *[]string, args *[]any, filter domain.SegmentFilter) {
	if len(filter.CityCodes) == 0 {
		return
	}
	placeholders := makePlaceholders(len(filter.CityCodes))
	*conditions = append(*conditions, "cs.city_code IN ("+placeholders+")")
	for _, cityCode := range filter.CityCodes {
		*args = append(*args, strings.TrimSpace(cityCode))
	}
}

// appendMembershipCondition appends an email opt-in/opt-out membership EXISTS filter.
func appendMembershipCondition(conditions *[]string, args *[]any, filter domain.SegmentFilter) {
	if filter.RequireEmailOptIn == nil {
		return
	}
	action := "opt_out"
	if *filter.RequireEmailOptIn {
		action = "opt_in"
	}
	*conditions = append(*conditions, `EXISTS (
		SELECT 1 FROM (
			SELECT contact_id, argMax(action, occurred_at) AS latest_action
			FROM membership_events
			WHERE channel = 'email'
			GROUP BY contact_id
		) ms
		WHERE ms.contact_id = cs.contact_id AND ms.latest_action = ?
	)`)
	*args = append(*args, action)
}

// appendOptInChannelCondition appends a channel-specific opt-in EXISTS filter.
func appendOptInChannelCondition(conditions *[]string, args *[]any, filter domain.SegmentFilter) {
	if strings.TrimSpace(filter.OptInChannel) == "" {
		return
	}
	action := strings.TrimSpace(filter.OptInAction)
	if action == "" {
		action = "opt_in"
	}
	*conditions = append(*conditions, `EXISTS (
		SELECT 1 FROM (
			SELECT contact_id, argMax(action, occurred_at) AS latest_action
			FROM membership_events
			WHERE channel = ?
			GROUP BY contact_id
		) ms
		WHERE ms.contact_id = cs.contact_id AND ms.latest_action = ?
	)`)
	*args = append(*args, strings.TrimSpace(filter.OptInChannel), action)
}

// appendMinSpendCondition appends a minimum total spend EXISTS filter.
func appendMinSpendCondition(conditions *[]string, args *[]any, filter domain.SegmentFilter) {
	if filter.MinTotalSpend == nil {
		return
	}
	*conditions = append(*conditions, `EXISTS (
		SELECT 1 FROM orders_fact of FINAL
		WHERE of.contact_id = cs.contact_id`+orderStatusFragment(filter.OrderStatuses, "of")+`
		GROUP BY of.contact_id
		HAVING sum(of.total_value) >= ?
	)`)
	*args = appendOrderStatusArgs(*args, filter.OrderStatuses)
	*args = append(*args, *filter.MinTotalSpend)
}

// appendPurchasedSKUCondition appends a purchased SKU IN (...) EXISTS filter.
func appendPurchasedSKUCondition(conditions *[]string, args *[]any, filter domain.SegmentFilter) {
	if len(filter.PurchasedSKUs) == 0 {
		return
	}
	orderStatusNested := ""
	if len(filter.OrderStatuses) > 0 {
		orderStatusNested = `
		AND EXISTS (
			SELECT 1 FROM orders_fact of FINAL
			WHERE of.order_id = oi.order_id AND of.current_status IN (` + makePlaceholders(len(filter.OrderStatuses)) + `)
		)`
	}
	*conditions = append(*conditions, `EXISTS (
		SELECT 1 FROM order_items_fact oi FINAL
		WHERE oi.contact_id = cs.contact_id AND oi.sku IN (`+makePlaceholders(len(filter.PurchasedSKUs))+`)`+orderStatusNested+`
	)`)
	for _, sku := range filter.PurchasedSKUs {
		*args = append(*args, strings.TrimSpace(sku))
	}
	*args = appendOrderStatusArgs(*args, filter.OrderStatuses)
}

// appendCategoryCondition appends a category pattern EXISTS filter.
func appendCategoryCondition(conditions *[]string, args *[]any, filter domain.SegmentFilter) {
	if strings.TrimSpace(filter.CategoryPattern) == "" {
		return
	}
	orderStatusNested := ""
	if len(filter.OrderStatuses) > 0 {
		orderStatusNested = `
		AND EXISTS (
			SELECT 1 FROM orders_fact of FINAL
			WHERE of.order_id = oi.order_id AND of.current_status IN (` + makePlaceholders(len(filter.OrderStatuses)) + `)
		)`
	}
	*conditions = append(*conditions, `EXISTS (
		SELECT 1 FROM order_items_fact oi FINAL
		WHERE oi.contact_id = cs.contact_id AND (lower(oi.sku) LIKE lower(?) OR lower(oi.alternate_name) LIKE lower(?))`+orderStatusNested+`
	)`)
	pattern := "%" + strings.TrimSpace(filter.CategoryPattern) + "%"
	*args = append(*args, pattern, pattern)
	*args = appendOrderStatusArgs(*args, filter.OrderStatuses)
}

// appendOrderRecencyCondition appends an order recency EXISTS filter.
func appendOrderRecencyCondition(conditions *[]string, args *[]any, filter domain.SegmentFilter) {
	if filter.OrderRecencyDays == nil || *filter.OrderRecencyDays <= 0 {
		return
	}
	*conditions = append(*conditions, `EXISTS (
		SELECT 1 FROM orders_fact of FINAL
		WHERE of.contact_id = cs.contact_id
		  AND of.created_at >= (now64(3) - toIntervalDay(?))`+orderStatusFragment(filter.OrderStatuses, "of")+`
	)`)
	*args = append(*args, *filter.OrderRecencyDays)
	*args = appendOrderStatusArgs(*args, filter.OrderStatuses)
}

// appendNoOrderRecencyCondition appends a no-order recency NOT EXISTS filter.
func appendNoOrderRecencyCondition(conditions *[]string, args *[]any, filter domain.SegmentFilter) {
	if filter.NoOrderRecencyDays == nil || *filter.NoOrderRecencyDays <= 0 {
		return
	}
	*conditions = append(*conditions, `NOT EXISTS (
		SELECT 1 FROM orders_fact of FINAL
		WHERE of.contact_id = cs.contact_id
		  AND of.created_at >= (now64(3) - toIntervalDay(?))`+orderStatusFragment(filter.OrderStatuses, "of")+`
	)`)
	*args = append(*args, *filter.NoOrderRecencyDays)
	*args = appendOrderStatusArgs(*args, filter.OrderStatuses)
}

// appendFirstPurchaseCondition appends a single-order HAVING EXISTS filter.
func appendFirstPurchaseCondition(conditions *[]string, args *[]any, filter domain.SegmentFilter) {
	if !filter.FirstPurchaseOnly {
		return
	}
	*conditions = append(*conditions, `EXISTS (
		SELECT 1 FROM orders_fact of FINAL
		WHERE of.contact_id = cs.contact_id`+orderStatusFragment(filter.OrderStatuses, "of")+`
		GROUP BY of.contact_id
		HAVING countDistinct(of.order_id) = 1
	)`)
	*args = appendOrderStatusArgs(*args, filter.OrderStatuses)
}

// appendSubscribedNoBuyCondition appends opted-in but never ordered double condition.
func appendSubscribedNoBuyCondition(conditions *[]string, args *[]any, filter domain.SegmentFilter) {
	if !filter.SubscribedNoBuy {
		return
	}
	*conditions = append(*conditions, `EXISTS (
		SELECT 1 FROM (
			SELECT contact_id, argMax(action, occurred_at) AS latest_action
			FROM membership_events
			WHERE channel = 'email'
			GROUP BY contact_id
		) ms
		WHERE ms.contact_id = cs.contact_id AND ms.latest_action = 'opt_in'
	)`)
	*conditions = append(*conditions, `NOT EXISTS (
		SELECT 1 FROM orders_fact of FINAL
		WHERE of.contact_id = cs.contact_id`+orderStatusFragment(filter.OrderStatuses, "of")+`
	)`)
	*args = appendOrderStatusArgs(*args, filter.OrderStatuses)
}

// appendTopSpenderCondition appends a contact_id IN filter from pre-resolved top spender IDs.
func appendTopSpenderCondition(conditions *[]string, args *[]any, topSpenderIDs []string) {
	if topSpenderIDs == nil {
		return
	}
	if len(topSpenderIDs) == 0 {
		*conditions = append(*conditions, "1 = 0")
		return
	}
	placeholders := makePlaceholders(len(topSpenderIDs))
	*conditions = append(*conditions, "cs.contact_id IN ("+placeholders+")")
	for _, contactID := range topSpenderIDs {
		*args = append(*args, strings.TrimSpace(contactID))
	}
}

// appendMetadataCondition appends a JSON metadata key/value EXISTS filter.
func appendMetadataCondition(conditions *[]string, args *[]any, filter domain.SegmentFilter) {
	if strings.TrimSpace(filter.MetadataKey) == "" {
		return
	}
	if strings.TrimSpace(filter.MetadataValue) == "" {
		*conditions = append(*conditions, "JSONExtractString(cs.metadata_json, ?) != ''")
		*args = append(*args, strings.TrimSpace(filter.MetadataKey))
	} else {
		*conditions = append(*conditions, "JSONExtractString(cs.metadata_json, ?) = ?")
		*args = append(*args, strings.TrimSpace(filter.MetadataKey), strings.TrimSpace(filter.MetadataValue))
	}
}
