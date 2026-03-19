package clickhouse

import (
	"context"
	"fmt"
	"math"
	"strings"

	"mannaiah/module/analytics/domain"
)

// resolveTopSpenderIDs resolves contact IDs for top-spender segment filters.
func (s *StoreAdapter) resolveTopSpenderIDs(ctx context.Context, filter domain.SegmentFilter) ([]string, error) {
	if s == nil || s.client == nil || s.client.db == nil {
		return nil, nil
	}
	if filter.TopSpendersLimit == nil && filter.TopSpendersPercentage == nil {
		return nil, nil
	}

	includedStatuses, excludedStatuses := collectOrderStatusScopes(filter)
	statusArgs := appendOrderStatusArgsWithExclusions(make([]any, 0, len(includedStatuses)+len(excludedStatuses)), includedStatuses, excludedStatuses)
	statusWhere := ""
	if len(includedStatuses) > 0 || len(excludedStatuses) > 0 {
		statusWhere = " WHERE 1 = 1" + orderStatusFragmentWithExclusions(includedStatuses, excludedStatuses, "")
	}

	var distinctContacts int64
	if err := s.client.db.QueryRowContext(ctx,
		"SELECT countDistinct(contact_id) FROM orders_fact FINAL"+statusWhere,
		statusArgs...,
	).Scan(&distinctContacts); err != nil {
		return nil, fmt.Errorf("count distinct top spender contacts: %w", err)
	}
	if distinctContacts <= 0 {
		return []string{}, nil
	}

	limit := resolveTopSpenderLimit(filter, distinctContacts)
	if limit <= 0 {
		return []string{}, nil
	}

	topArgs := append(append([]any{}, statusArgs...), limit)
	rows, err := s.client.db.QueryContext(
		ctx,
		"SELECT contact_id FROM orders_fact FINAL"+statusWhere+" GROUP BY contact_id ORDER BY sum(total_value) DESC LIMIT ?",
		topArgs...,
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

// resolveTopSpenderLimit resolves the contact count limit for top-spender filters.
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

// isImpossibleFilter reports whether a top-spender filter resolves to an empty set.
func isImpossibleFilter(topSpenderIDs []string, filter domain.SegmentFilter) bool {
	return topSpenderIDs != nil && len(topSpenderIDs) == 0 &&
		(filter.TopSpendersLimit != nil || filter.TopSpendersPercentage != nil)
}

// orderStatusFragment returns an " AND [alias.]current_status IN (?,?,...)" SQL fragment
// when statuses is non-empty, otherwise returns an empty string.
func orderStatusFragment(statuses []string, alias string) string {
	if len(statuses) == 0 {
		return ""
	}
	col := "current_status"
	if alias != "" {
		col = alias + ".current_status"
	}
	return " AND " + col + " IN (" + makePlaceholders(len(statuses)) + ")"
}

// appendOrderStatusArgs appends trimmed status values to the given args slice.
func appendOrderStatusArgs(args []any, statuses []string) []any {
	for _, s := range statuses {
		args = append(args, strings.TrimSpace(s))
	}
	return args
}

// makePlaceholders returns a comma-separated string of count placeholder tokens.
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
