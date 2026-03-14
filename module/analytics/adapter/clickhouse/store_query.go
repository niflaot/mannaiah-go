package clickhouse

import (
	"context"
	"fmt"
	"math"
	"strings"

	"mannaiah/module/analytics/domain"
)

// ResolveContacts resolves analytical contact IDs by filter.
func (s *StoreAdapter) ResolveContacts(ctx context.Context, filter domain.SegmentFilter, page int, limit int) ([]string, error) {
	if s == nil || s.client == nil || s.client.db == nil {
		return nil, nil
	}
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 1000
	}

	topSpenderIDs, err := s.resolveTopSpenderIDs(ctx, filter)
	if err != nil {
		return nil, err
	}
	if isImpossibleFilter(topSpenderIDs, filter) {
		return []string{}, nil
	}

	whereSQL, args := buildSegmentWhere(filter, topSpenderIDs)
	query := "SELECT DISTINCT cs.contact_id FROM contacts_snapshot FINAL cs WHERE " + whereSQL + " ORDER BY cs.contact_id ASC LIMIT ? OFFSET ?"
	args = append(args, limit, (page-1)*limit)

	rows, err := s.client.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query clickhouse contact ids: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	result := make([]string, 0, limit)
	for rows.Next() {
		var contactID string
		if scanErr := rows.Scan(&contactID); scanErr != nil {
			return nil, fmt.Errorf("scan clickhouse contact id: %w", scanErr)
		}
		result = append(result, strings.TrimSpace(contactID))
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate clickhouse contact ids: %w", rowsErr)
	}

	return result, nil
}

// CountContacts counts analytical contacts by filter.
func (s *StoreAdapter) CountContacts(ctx context.Context, filter domain.SegmentFilter) (int64, error) {
	if s == nil || s.client == nil || s.client.db == nil {
		return 0, nil
	}

	topSpenderIDs, err := s.resolveTopSpenderIDs(ctx, filter)
	if err != nil {
		return 0, err
	}
	if isImpossibleFilter(topSpenderIDs, filter) {
		return 0, nil
	}

	whereSQL, args := buildSegmentWhere(filter, topSpenderIDs)
	query := "SELECT countDistinct(cs.contact_id) FROM contacts_snapshot FINAL cs WHERE " + whereSQL

	var count int64
	if err := s.client.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count clickhouse contact ids: %w", err)
	}

	return count, nil
}

func resolveTopSpenderLimit(filter domain.SegmentFilter, distinctContacts int64) int64 {
	if filter.TopSpendersLimit != nil && *filter.TopSpendersLimit > 0 {
		return int64(*filter.TopSpendersLimit)
	}
	if filter.TopSpendersPercentage != nil && *filter.TopSpendersPercentage > 0 {
		computed := int64(math.Ceil(float64(distinctContacts) * (*filter.TopSpendersPercentage) / 100.0))
		if computed < 1 {
			computed = 1
		}
		return computed
	}

	return 0
}

func (s *StoreAdapter) resolveTopSpenderIDs(ctx context.Context, filter domain.SegmentFilter) ([]string, error) {
	if s == nil || s.client == nil || s.client.db == nil {
		return nil, nil
	}
	if filter.TopSpendersLimit == nil && filter.TopSpendersPercentage == nil {
		return nil, nil
	}

	var distinctContacts int64
	if err := s.client.db.QueryRowContext(ctx, "SELECT countDistinct(contact_id) FROM orders_fact FINAL").Scan(&distinctContacts); err != nil {
		return nil, fmt.Errorf("count distinct top spender contacts: %w", err)
	}
	if distinctContacts <= 0 {
		return []string{}, nil
	}

	limit := resolveTopSpenderLimit(filter, distinctContacts)
	if limit <= 0 {
		return []string{}, nil
	}

	rows, err := s.client.db.QueryContext(
		ctx,
		"SELECT contact_id FROM orders_fact FINAL GROUP BY contact_id ORDER BY sum(total_value) DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query top spender contacts: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	ids := make([]string, 0, limit)
	for rows.Next() {
		var contactID string
		if scanErr := rows.Scan(&contactID); scanErr != nil {
			return nil, fmt.Errorf("scan top spender contact id: %w", scanErr)
		}
		ids = append(ids, strings.TrimSpace(contactID))
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate top spender contact ids: %w", rowsErr)
	}

	return ids, nil
}

func isImpossibleFilter(topSpenderIDs []string, filter domain.SegmentFilter) bool {
	if topSpenderIDs != nil && len(topSpenderIDs) == 0 && (filter.TopSpendersLimit != nil || filter.TopSpendersPercentage != nil) {
		return true
	}

	return false
}

func buildSegmentWhere(filter domain.SegmentFilter, topSpenderIDs []string) (string, []any) {
	conditions := []string{"1 = 1"}
	args := make([]any, 0, 16)

	if len(filter.CityCodes) > 0 {
		placeholders := makePlaceholders(len(filter.CityCodes))
		conditions = append(conditions, "cs.city_code IN ("+placeholders+")")
		for _, cityCode := range filter.CityCodes {
			args = append(args, strings.TrimSpace(cityCode))
		}
	}
	if filter.RequireEmailOptIn != nil {
		action := "opt_out"
		if *filter.RequireEmailOptIn {
			action = "opt_in"
		}
		conditions = append(conditions, `EXISTS (
			SELECT 1 FROM (
				SELECT contact_id, argMax(action, occurred_at) AS latest_action
				FROM membership_events
				WHERE channel = 'email'
				GROUP BY contact_id
			) ms
			WHERE ms.contact_id = cs.contact_id AND ms.latest_action = ?
		)`)
		args = append(args, action)
	}
	if strings.TrimSpace(filter.OptInChannel) != "" {
		action := strings.TrimSpace(filter.OptInAction)
		if action == "" {
			action = "opt_in"
		}
		conditions = append(conditions, `EXISTS (
			SELECT 1 FROM (
				SELECT contact_id, argMax(action, occurred_at) AS latest_action
				FROM membership_events
				WHERE channel = ?
				GROUP BY contact_id
			) ms
			WHERE ms.contact_id = cs.contact_id AND ms.latest_action = ?
		)`)
		args = append(args, strings.TrimSpace(filter.OptInChannel), action)
	}
	if filter.MinTotalSpend != nil {
		conditions = append(conditions, `EXISTS (
			SELECT 1 FROM orders_fact FINAL of
			WHERE of.contact_id = cs.contact_id
			GROUP BY of.contact_id
			HAVING sum(of.total_value) >= ?
		)`)
		args = append(args, *filter.MinTotalSpend)
	}
	if strings.TrimSpace(filter.PurchasedSKU) != "" {
		conditions = append(conditions, `EXISTS (
			SELECT 1 FROM order_items_fact FINAL oi
			WHERE oi.contact_id = cs.contact_id AND oi.sku = ?
		)`)
		args = append(args, strings.TrimSpace(filter.PurchasedSKU))
	}
	if strings.TrimSpace(filter.CategoryPattern) != "" {
		conditions = append(conditions, `EXISTS (
			SELECT 1 FROM order_items_fact FINAL oi
			WHERE oi.contact_id = cs.contact_id AND (lower(oi.sku) LIKE lower(?) OR lower(oi.alternate_name) LIKE lower(?))
		)`)
		pattern := "%" + strings.TrimSpace(filter.CategoryPattern) + "%"
		args = append(args, pattern, pattern)
	}
	if filter.OrderRecencyDays != nil && *filter.OrderRecencyDays > 0 {
		conditions = append(conditions, `EXISTS (
			SELECT 1 FROM orders_fact FINAL of
			WHERE of.contact_id = cs.contact_id
			  AND of.created_at >= (now64(3) - toIntervalDay(?))
		)`)
		args = append(args, *filter.OrderRecencyDays)
	}
	if filter.NoOrderRecencyDays != nil && *filter.NoOrderRecencyDays > 0 {
		conditions = append(conditions, `NOT EXISTS (
			SELECT 1 FROM orders_fact FINAL of
			WHERE of.contact_id = cs.contact_id
			  AND of.created_at >= (now64(3) - toIntervalDay(?))
		)`)
		args = append(args, *filter.NoOrderRecencyDays)
	}
	if filter.FirstPurchaseOnly {
		conditions = append(conditions, `EXISTS (
			SELECT 1 FROM orders_fact FINAL of
			WHERE of.contact_id = cs.contact_id
			GROUP BY of.contact_id
			HAVING countDistinct(of.order_id) = 1
		)`)
	}
	if filter.SubscribedNoBuy {
		conditions = append(conditions, `EXISTS (
			SELECT 1 FROM (
				SELECT contact_id, argMax(action, occurred_at) AS latest_action
				FROM membership_events
				WHERE channel = 'email'
				GROUP BY contact_id
			) ms
			WHERE ms.contact_id = cs.contact_id AND ms.latest_action = 'opt_in'
		)`)
		conditions = append(conditions, `NOT EXISTS (
			SELECT 1 FROM orders_fact FINAL of
			WHERE of.contact_id = cs.contact_id
		)`)
	}
	if topSpenderIDs != nil {
		if len(topSpenderIDs) == 0 {
			conditions = append(conditions, "1 = 0")
		} else {
			placeholders := makePlaceholders(len(topSpenderIDs))
			conditions = append(conditions, "cs.contact_id IN ("+placeholders+")")
			for _, contactID := range topSpenderIDs {
				args = append(args, strings.TrimSpace(contactID))
			}
		}
	}
	if strings.TrimSpace(filter.MetadataKey) != "" {
		if strings.TrimSpace(filter.MetadataValue) == "" {
			conditions = append(conditions, "JSONExtractString(cs.metadata_json, ?) != ''")
			args = append(args, strings.TrimSpace(filter.MetadataKey))
		} else {
			conditions = append(conditions, "JSONExtractString(cs.metadata_json, ?) = ?")
			args = append(args, strings.TrimSpace(filter.MetadataKey), strings.TrimSpace(filter.MetadataValue))
		}
	}

	return strings.Join(conditions, " AND "), args
}

func makePlaceholders(count int) string {
	if count <= 0 {
		return ""
	}

	placeholders := make([]string, 0, count)
	for index := 0; index < count; index++ {
		placeholders = append(placeholders, "?")
	}

	return strings.Join(placeholders, ",")
}
