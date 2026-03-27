package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"mannaiah/module/analytics/domain"
)

// GetTagAffinity retrieves ranked tag affinity scores for one contact.
func (s *StoreAdapter) GetTagAffinity(ctx context.Context, contactID string, limit int, minScore float64) ([]domain.TagAffinity, error) {
	if s == nil || s.client == nil || s.client.db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}

	query := `SELECT contact_id, tag, sum(affinity_score) AS affinity_score, sum(total_spent) AS total_spent, sum(purchase_count) AS purchase_count
		FROM tag_affinity_mv FINAL
		WHERE contact_id = ?
		GROUP BY contact_id, tag
		HAVING affinity_score >= ?
		ORDER BY affinity_score DESC
		LIMIT ?`

	rows, err := s.client.db.QueryContext(ctx, query, strings.TrimSpace(contactID), minScore, limit)
	if err != nil {
		return nil, fmt.Errorf("query tag affinity: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make([]domain.TagAffinity, 0, limit)
	for rows.Next() {
		var row domain.TagAffinity
		if scanErr := rows.Scan(&row.ContactID, &row.Tag, &row.AffinityScore, &row.TotalSpent, &row.PurchaseCount); scanErr != nil {
			return nil, fmt.Errorf("scan tag affinity row: %w", scanErr)
		}
		row.ContactID = strings.TrimSpace(row.ContactID)
		row.Tag = strings.TrimSpace(row.Tag)
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tag affinity rows: %w", err)
	}

	return result, nil
}

// GetCategoryAffinity retrieves ranked category affinity scores for one contact.
func (s *StoreAdapter) GetCategoryAffinity(ctx context.Context, contactID string, limit int, minScore float64) ([]domain.CategoryAffinity, error) {
	if s == nil || s.client == nil || s.client.db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}

	query := `SELECT contact_id, category_id, any(category_name), sum(affinity_score) AS affinity_score, sum(total_spent) AS total_spent, sum(purchase_count) AS purchase_count
		FROM category_affinity_mv FINAL
		WHERE contact_id = ?
		GROUP BY contact_id, category_id
		HAVING affinity_score >= ?
		ORDER BY affinity_score DESC
		LIMIT ?`

	rows, err := s.client.db.QueryContext(ctx, query, strings.TrimSpace(contactID), minScore, limit)
	if err != nil {
		return nil, fmt.Errorf("query category affinity: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make([]domain.CategoryAffinity, 0, limit)
	for rows.Next() {
		var row domain.CategoryAffinity
		if scanErr := rows.Scan(&row.ContactID, &row.CategoryID, &row.CategoryName, &row.AffinityScore, &row.TotalSpent, &row.PurchaseCount); scanErr != nil {
			return nil, fmt.Errorf("scan category affinity row: %w", scanErr)
		}
		row.ContactID = strings.TrimSpace(row.ContactID)
		row.CategoryID = strings.TrimSpace(row.CategoryID)
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate category affinity rows: %w", err)
	}

	return result, nil
}

// GetVariationAffinity retrieves ranked variation affinity scores for one contact.
func (s *StoreAdapter) GetVariationAffinity(ctx context.Context, contactID string, limit int, minScore float64) ([]domain.VariationAffinity, error) {
	if s == nil || s.client == nil || s.client.db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}

	query := `SELECT contact_id, variation_name, variation_value, sum(affinity_score) AS affinity_score, sum(total_spent) AS total_spent, sum(purchase_count) AS purchase_count
		FROM variation_affinity_mv FINAL
		WHERE contact_id = ?
		GROUP BY contact_id, variation_name, variation_value
		HAVING affinity_score >= ?
		ORDER BY affinity_score DESC
		LIMIT ?`

	rows, err := s.client.db.QueryContext(ctx, query, strings.TrimSpace(contactID), minScore, limit)
	if err != nil {
		return nil, fmt.Errorf("query variation affinity: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make([]domain.VariationAffinity, 0, limit)
	for rows.Next() {
		var row domain.VariationAffinity
		if scanErr := rows.Scan(&row.ContactID, &row.VariationName, &row.VariationValue, &row.AffinityScore, &row.TotalSpent, &row.PurchaseCount); scanErr != nil {
			return nil, fmt.Errorf("scan variation affinity row: %w", scanErr)
		}
		row.ContactID = strings.TrimSpace(row.ContactID)
		row.VariationName = strings.TrimSpace(row.VariationName)
		row.VariationValue = strings.TrimSpace(row.VariationValue)
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate variation affinity rows: %w", err)
	}

	return result, nil
}

// GetProfile assembles a full affinity profile for one contact.
func (s *StoreAdapter) GetProfile(ctx context.Context, contactID string, limit int, minScore float64) (*domain.AffinityProfile, error) {
	if s == nil || s.client == nil || s.client.db == nil {
		return &domain.AffinityProfile{ContactID: strings.TrimSpace(contactID)}, nil
	}

	tags, err := s.GetTagAffinity(ctx, contactID, limit, minScore)
	if err != nil {
		return nil, err
	}
	cats, err := s.GetCategoryAffinity(ctx, contactID, limit, minScore)
	if err != nil {
		return nil, err
	}
	vars, err := s.GetVariationAffinity(ctx, contactID, limit, minScore)
	if err != nil {
		return nil, err
	}

	return &domain.AffinityProfile{
		ContactID:  strings.TrimSpace(contactID),
		Tags:       tags,
		Categories: cats,
		Variations: vars,
	}, nil
}

// GetPurchasedProductIDs returns unique purchased product identifiers for one contact.
func (s *StoreAdapter) GetPurchasedProductIDs(ctx context.Context, contactID string, limit int) ([]string, error) {
	if s == nil || s.client == nil || s.client.db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 2000
	}

	query := `SELECT product_id
		FROM order_items_fact oi FINAL
		WHERE oi.contact_id = ?
		  AND oi.product_id != ''
		GROUP BY product_id
		ORDER BY max(oi.order_created_at) DESC
		LIMIT ?`
	rows, err := s.client.db.QueryContext(ctx, query, strings.TrimSpace(contactID), limit)
	if err != nil {
		return nil, fmt.Errorf("query purchased product ids: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make([]string, 0, limit)
	seen := make(map[string]struct{}, limit)
	for rows.Next() {
		var productID string
		if scanErr := rows.Scan(&productID); scanErr != nil {
			return nil, fmt.Errorf("scan purchased product id row: %w", scanErr)
		}
		productID = strings.TrimSpace(productID)
		if productID == "" {
			continue
		}
		if _, exists := seen[productID]; exists {
			continue
		}
		seen[productID] = struct{}{}
		result = append(result, productID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate purchased product id rows: %w", err)
	}

	return result, nil
}

// RefreshTagMV truncates and repopulates the tag_affinity_mv table.
func (s *StoreAdapter) RefreshTagMV(ctx context.Context) error {
	if s == nil || s.client == nil || s.client.db == nil {
		return nil
	}

	return withTx(ctx, s.client.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "TRUNCATE TABLE tag_affinity_mv"); err != nil {
			return fmt.Errorf("truncate tag_affinity_mv: %w", err)
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO tag_affinity_mv
			SELECT oi.contact_id, pt.tag,
			       sum(oi.value * exp(-0.01 * dateDiff('day', oi.order_created_at, now64(3)))),
			       sum(oi.value), toUInt32(count(*))
			FROM order_items_fact oi FINAL
			INNER JOIN product_taxonomy pt FINAL ON oi.product_id = pt.product_id
			WHERE pt.tag != ''
			GROUP BY oi.contact_id, pt.tag`)
		if err != nil {
			return fmt.Errorf("repopulate tag_affinity_mv: %w", err)
		}

		return nil
	})
}

// RefreshCategoryMV truncates and repopulates the category_affinity_mv table.
func (s *StoreAdapter) RefreshCategoryMV(ctx context.Context) error {
	if s == nil || s.client == nil || s.client.db == nil {
		return nil
	}

	return withTx(ctx, s.client.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "TRUNCATE TABLE category_affinity_mv"); err != nil {
			return fmt.Errorf("truncate category_affinity_mv: %w", err)
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO category_affinity_mv
			SELECT oi.contact_id, pt.category_id, any(pt.category_name),
			       sum(oi.value * exp(-0.01 * dateDiff('day', oi.order_created_at, now64(3)))),
			       sum(oi.value), toUInt32(count(*))
			FROM order_items_fact oi FINAL
			INNER JOIN product_taxonomy pt FINAL ON oi.product_id = pt.product_id
			WHERE pt.category_id != ''
			GROUP BY oi.contact_id, pt.category_id`)
		if err != nil {
			return fmt.Errorf("repopulate category_affinity_mv: %w", err)
		}

		return nil
	})
}

// RefreshVariationMV truncates and repopulates the variation_affinity_mv table.
func (s *StoreAdapter) RefreshVariationMV(ctx context.Context) error {
	if s == nil || s.client == nil || s.client.db == nil {
		return nil
	}

	return withTx(ctx, s.client.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "TRUNCATE TABLE variation_affinity_mv"); err != nil {
			return fmt.Errorf("truncate variation_affinity_mv: %w", err)
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO variation_affinity_mv
			SELECT oi.contact_id, pvt.variation_name, pvt.variation_value,
			       sum(oi.value * exp(-0.01 * dateDiff('day', oi.order_created_at, now64(3)))),
			       sum(oi.value), toUInt32(count(*))
			FROM order_items_fact oi FINAL
			INNER JOIN product_variation_taxonomy pvt FINAL ON oi.product_id = pvt.product_id
			GROUP BY oi.contact_id, pvt.variation_name, pvt.variation_value`)
		if err != nil {
			return fmt.Errorf("repopulate variation_affinity_mv: %w", err)
		}

		return nil
	})
}
