package clickhouse

import (
	"context"
	"fmt"
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
	query := "SELECT DISTINCT cs.contact_id FROM contacts_snapshot cs FINAL WHERE " + whereSQL + " ORDER BY cs.contact_id ASC LIMIT ? OFFSET ?"
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
	query := "SELECT countDistinct(cs.contact_id) FROM contacts_snapshot cs FINAL WHERE " + whereSQL

	var count int64
	if err := s.client.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count clickhouse contact ids: %w", err)
	}

	return count, nil
}
