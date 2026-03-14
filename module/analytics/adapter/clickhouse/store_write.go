package clickhouse

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"mannaiah/module/analytics/port"
)

// UpsertContacts upserts contact snapshot rows.
func (s *StoreAdapter) UpsertContacts(ctx context.Context, rows []port.ContactSnapshot) error {
	if len(rows) == 0 || s == nil || s.client == nil || s.client.db == nil {
		return nil
	}

	query := `INSERT INTO contacts_snapshot (contact_id, email, first_name, last_name, legal_name, phone, city_code, document_type, metadata_json, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	return withPreparedInsert(ctx, s.client.db, query, len(rows), func(stmt *sql.Stmt, idx int) error {
		row := rows[idx]
		metadataJSON := encodeMetadata(row.Metadata)
		_, err := stmt.ExecContext(
			ctx,
			strings.TrimSpace(row.ContactID),
			strings.TrimSpace(row.Email),
			strings.TrimSpace(row.FirstName),
			strings.TrimSpace(row.LastName),
			strings.TrimSpace(row.LegalName),
			strings.TrimSpace(row.Phone),
			strings.TrimSpace(row.CityCode),
			strings.TrimSpace(row.DocumentType),
			metadataJSON,
			row.CreatedAt.UTC(),
			row.UpdatedAt.UTC(),
		)
		if err != nil {
			return fmt.Errorf("insert contacts_snapshot row: %w", err)
		}

		return nil
	})
}

// UpsertOrders upserts order fact rows.
func (s *StoreAdapter) UpsertOrders(ctx context.Context, rows []port.OrderFact) error {
	if len(rows) == 0 || s == nil || s.client == nil || s.client.db == nil {
		return nil
	}

	query := `INSERT INTO orders_fact (order_id, identifier, realm, contact_id, current_status, total_value, item_count, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	return withPreparedInsert(ctx, s.client.db, query, len(rows), func(stmt *sql.Stmt, idx int) error {
		row := rows[idx]
		_, err := stmt.ExecContext(
			ctx,
			strings.TrimSpace(row.OrderID),
			strings.TrimSpace(row.Identifier),
			strings.TrimSpace(row.Realm),
			strings.TrimSpace(row.ContactID),
			strings.TrimSpace(row.CurrentStatus),
			row.TotalValue,
			maxInt(row.ItemCount, 0),
			row.CreatedAt.UTC(),
			row.UpdatedAt.UTC(),
		)
		if err != nil {
			return fmt.Errorf("insert orders_fact row: %w", err)
		}

		return nil
	})
}

// UpsertOrderItems appends order item fact rows.
func (s *StoreAdapter) UpsertOrderItems(ctx context.Context, rows []port.OrderItemFact) error {
	if len(rows) == 0 || s == nil || s.client == nil || s.client.db == nil {
		return nil
	}

	query := `INSERT INTO order_items_fact (order_id, contact_id, sku, alternate_name, product_id, quantity, value, resolution_source, order_created_at, order_updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	return withPreparedInsert(ctx, s.client.db, query, len(rows), func(stmt *sql.Stmt, idx int) error {
		row := rows[idx]
		_, err := stmt.ExecContext(
			ctx,
			strings.TrimSpace(row.OrderID),
			strings.TrimSpace(row.ContactID),
			strings.TrimSpace(row.SKU),
			strings.TrimSpace(row.AlternateName),
			strings.TrimSpace(row.ProductID),
			maxInt(row.Quantity, 0),
			row.Value,
			strings.TrimSpace(row.ResolutionSource),
			row.OrderCreatedAt.UTC(),
			row.OrderUpdatedAt.UTC(),
		)
		if err != nil {
			return fmt.Errorf("insert order_items_fact row: %w", err)
		}

		return nil
	})
}

// InsertMembershipEvents inserts membership event rows.
func (s *StoreAdapter) InsertMembershipEvents(ctx context.Context, rows []port.MembershipEvent) error {
	if len(rows) == 0 || s == nil || s.client == nil || s.client.db == nil {
		return nil
	}

	query := `INSERT INTO membership_events (contact_id, channel, action, source, occurred_at) VALUES (?, ?, ?, ?, ?)`
	return withPreparedInsert(ctx, s.client.db, query, len(rows), func(stmt *sql.Stmt, idx int) error {
		row := rows[idx]
		_, err := stmt.ExecContext(
			ctx,
			strings.TrimSpace(row.ContactID),
			strings.TrimSpace(row.Channel),
			strings.TrimSpace(row.Action),
			strings.TrimSpace(row.Source),
			row.OccurredAt.UTC(),
		)
		if err != nil {
			return fmt.Errorf("insert membership_events row: %w", err)
		}

		return nil
	})
}

// InsertCampaignEvents inserts campaign event rows.
func (s *StoreAdapter) InsertCampaignEvents(ctx context.Context, rows []port.CampaignEvent) error {
	if len(rows) == 0 || s == nil || s.client == nil || s.client.db == nil {
		return nil
	}

	query := `INSERT INTO campaign_events (campaign_id, contact_id, channel, status, template_version, occurred_at) VALUES (?, ?, ?, ?, ?, ?)`
	return withPreparedInsert(ctx, s.client.db, query, len(rows), func(stmt *sql.Stmt, idx int) error {
		row := rows[idx]
		_, err := stmt.ExecContext(
			ctx,
			strings.TrimSpace(row.CampaignID),
			strings.TrimSpace(row.ContactID),
			strings.TrimSpace(row.Channel),
			strings.TrimSpace(row.Status),
			maxInt(row.TemplateVersion, 0),
			row.OccurredAt.UTC(),
		)
		if err != nil {
			return fmt.Errorf("insert campaign_events row: %w", err)
		}

		return nil
	})
}

func withPreparedInsert(ctx context.Context, db *sql.DB, query string, size int, fn func(stmt *sql.Stmt, idx int) error) error {
	return withTx(ctx, db, func(tx *sql.Tx) error {
		statement, err := tx.PrepareContext(ctx, query)
		if err != nil {
			return fmt.Errorf("prepare clickhouse insert: %w", err)
		}
		defer func() {
			_ = statement.Close()
		}()

		for index := 0; index < size; index++ {
			if err := fn(statement, index); err != nil {
				return err
			}
		}

		return nil
	})
}

func encodeMetadata(metadata map[string]string) string {
	if len(metadata) == 0 {
		return "{}"
	}

	payload, err := json.Marshal(metadata)
	if err != nil {
		return "{}"
	}

	return string(payload)
}

func maxInt(value int, minimum int) int {
	if value < minimum {
		return minimum
	}

	return value
}
