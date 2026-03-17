package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"mannaiah/module/analytics/port"
)

// UpsertProductTaxonomy batch-upserts product taxonomy rows into the product_taxonomy table.
func (s *StoreAdapter) UpsertProductTaxonomy(ctx context.Context, rows []port.ProductTaxonomyRow) error {
	if len(rows) == 0 || s == nil || s.client == nil || s.client.db == nil {
		return nil
	}

	query := `INSERT INTO product_taxonomy (product_id, tag, category_id, category_name, updated_at) VALUES (?, ?, ?, ?, ?)`
	return withPreparedInsert(ctx, s.client.db, query, len(rows), func(stmt *sql.Stmt, idx int) error {
		row := rows[idx]
		_, err := stmt.ExecContext(
			ctx,
			strings.TrimSpace(row.ProductID),
			strings.TrimSpace(row.Tag),
			strings.TrimSpace(row.CategoryID),
			strings.TrimSpace(row.CategoryName),
			row.UpdatedAt.UTC(),
		)
		if err != nil {
			return fmt.Errorf("insert product_taxonomy row: %w", err)
		}

		return nil
	})
}

// UpsertVariationTaxonomy batch-upserts variation taxonomy rows into the product_variation_taxonomy table.
func (s *StoreAdapter) UpsertVariationTaxonomy(ctx context.Context, rows []port.VariationTaxonomyRow) error {
	if len(rows) == 0 || s == nil || s.client == nil || s.client.db == nil {
		return nil
	}

	query := `INSERT INTO product_variation_taxonomy (product_id, sku, variation_id, variation_name, variation_value, updated_at) VALUES (?, ?, ?, ?, ?, ?)`
	return withPreparedInsert(ctx, s.client.db, query, len(rows), func(stmt *sql.Stmt, idx int) error {
		row := rows[idx]
		_, err := stmt.ExecContext(
			ctx,
			strings.TrimSpace(row.ProductID),
			strings.TrimSpace(row.SKU),
			strings.TrimSpace(row.VariationID),
			strings.TrimSpace(row.VariationName),
			strings.TrimSpace(row.VariationValue),
			row.UpdatedAt.UTC(),
		)
		if err != nil {
			return fmt.Errorf("insert product_variation_taxonomy row: %w", err)
		}

		return nil
	})
}
