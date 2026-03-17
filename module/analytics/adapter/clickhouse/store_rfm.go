package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"mannaiah/module/analytics/domain"
)

// ScoreContact computes RFM scores for one contact using the provided band configs.
func (s *StoreAdapter) ScoreContact(ctx context.Context, contactID string, bands []domain.RFMBandConfig) (*domain.RFMScore, error) {
	if s == nil || s.client == nil || s.client.db == nil {
		return nil, nil
	}

	rBand, fBand, mBand := resolveBands(bands)
	rSQL := buildBandSQL("recency_days", false, rBand)
	fSQL := buildBandSQL("frequency", true, fBand)
	mSQL := buildBandSQL("monetary", true, mBand)

	query := fmt.Sprintf(`SELECT contact_id, recency_days, frequency, monetary, %s, %s, %s
		FROM rfm_scores_mv FINAL
		WHERE contact_id = ?
		LIMIT 1`, rSQL, fSQL, mSQL)

	args := collectBandArgs(rBand, fBand, mBand)
	args = append(args, strings.TrimSpace(contactID))

	row := s.client.db.QueryRowContext(ctx, query, args...)
	score, err := scanRFMRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("score rfm contact: %w", err)
	}

	return score, nil
}

// ScoreBatch computes RFM scores for up to 1000 contacts.
func (s *StoreAdapter) ScoreBatch(ctx context.Context, contactIDs []string, bands []domain.RFMBandConfig) ([]domain.RFMScore, error) {
	if s == nil || s.client == nil || s.client.db == nil || len(contactIDs) == 0 {
		return nil, nil
	}
	if len(contactIDs) > 1000 {
		contactIDs = contactIDs[:1000]
	}

	rBand, fBand, mBand := resolveBands(bands)
	rSQL := buildBandSQL("recency_days", false, rBand)
	fSQL := buildBandSQL("frequency", true, fBand)
	mSQL := buildBandSQL("monetary", true, mBand)
	placeholders := makePlaceholders(len(contactIDs))

	query := fmt.Sprintf(`SELECT contact_id, recency_days, frequency, monetary, %s, %s, %s
		FROM rfm_scores_mv FINAL
		WHERE contact_id IN (%s)`, rSQL, fSQL, mSQL, placeholders)

	args := collectBandArgs(rBand, fBand, mBand)
	for _, id := range contactIDs {
		args = append(args, strings.TrimSpace(id))
	}

	rows, err := s.client.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("score rfm batch: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make([]domain.RFMScore, 0, len(contactIDs))
	for rows.Next() {
		score, scanErr := scanRFMRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan rfm score row: %w", scanErr)
		}
		result = append(result, *score)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rfm score rows: %w", err)
	}

	return result, nil
}

// RefreshMV truncates and repopulates the rfm_scores_mv table.
func (s *StoreAdapter) RefreshMV(ctx context.Context) error {
	if s == nil || s.client == nil || s.client.db == nil {
		return nil
	}

	return withTx(ctx, s.client.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "TRUNCATE TABLE rfm_scores_mv"); err != nil {
			return fmt.Errorf("truncate rfm_scores_mv: %w", err)
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO rfm_scores_mv
			SELECT contact_id,
			       toUInt32(dateDiff('day', max(created_at), now64(3))) AS recency_days,
			       toUInt32(countDistinct(order_id))                     AS frequency,
			       sum(total_value)                                       AS monetary,
			       now64(3)                                               AS updated_at
			FROM orders_fact FINAL
			GROUP BY contact_id`)
		if err != nil {
			return fmt.Errorf("repopulate rfm_scores_mv: %w", err)
		}

		return nil
	})
}

// ComputeMonetaryPercentiles returns [p20, p40, p60, p80] monetary percentile thresholds.
func (s *StoreAdapter) ComputeMonetaryPercentiles(ctx context.Context) ([4]float64, error) {
	var result [4]float64
	if s == nil || s.client == nil || s.client.db == nil {
		return result, nil
	}

	query := `SELECT quantile(0.2)(monetary), quantile(0.4)(monetary), quantile(0.6)(monetary), quantile(0.8)(monetary)
		FROM rfm_scores_mv FINAL`
	row := s.client.db.QueryRowContext(ctx, query)
	if err := row.Scan(&result[0], &result[1], &result[2], &result[3]); err != nil {
		return result, fmt.Errorf("compute monetary percentiles: %w", err)
	}

	return result, nil
}

// rfmBandValues holds threshold values for one band dimension.
type rfmBandValues struct {
	band5 float64
	band4 float64
	band3 float64
	band2 float64
}

// resolveBands extracts band threshold values for R, F, and M dimensions.
func resolveBands(bands []domain.RFMBandConfig) (r, f, m rfmBandValues) {
	r = rfmBandValues{band5: 7, band4: 30, band3: 90, band2: 180}
	f = rfmBandValues{band5: 10, band4: 6, band3: 3, band2: 2}
	m = rfmBandValues{band5: 1000, band4: 500, band3: 200, band2: 50}

	for _, b := range bands {
		switch b.Dimension {
		case domain.DimensionRecency:
			r = rfmBandValues{band5: b.Band5Min, band4: b.Band4Min, band3: b.Band3Min, band2: b.Band2Min}
		case domain.DimensionFrequency:
			f = rfmBandValues{band5: b.Band5Min, band4: b.Band4Min, band3: b.Band3Min, band2: b.Band2Min}
		case domain.DimensionMonetary:
			m = rfmBandValues{band5: b.Band5Min, band4: b.Band4Min, band3: b.Band3Min, band2: b.Band2Min}
		}
	}

	return r, f, m
}

// buildBandSQL constructs a multiIf SQL expression for one RFM dimension.
func buildBandSQL(col string, ascending bool, b rfmBandValues) string {
	if ascending {
		return fmt.Sprintf(
			"multiIf(%s >= ?, 5, %s >= ?, 4, %s >= ?, 3, %s >= ?, 2, 1)",
			col, col, col, col,
		)
	}

	return fmt.Sprintf(
		"multiIf(%s <= ?, 5, %s <= ?, 4, %s <= ?, 3, %s <= ?, 2, 1)",
		col, col, col, col,
	)
}

// collectBandArgs builds the args slice for R, F, M band multiIf expressions.
func collectBandArgs(r, f, m rfmBandValues) []any {
	return []any{
		r.band5, r.band4, r.band3, r.band2,
		f.band5, f.band4, f.band3, f.band2,
		m.band5, m.band4, m.band3, m.band2,
	}
}

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanRFMRow scans one rfm score row from a scanner.
func scanRFMRow(row rowScanner) (*domain.RFMScore, error) {
	var (
		contactID   string
		recencyDays uint32
		frequency   uint32
		monetary    float64
		rScore      int
		fScore      int
		mScore      int
	)
	if err := row.Scan(&contactID, &recencyDays, &frequency, &monetary, &rScore, &fScore, &mScore); err != nil {
		return nil, err
	}

	return &domain.RFMScore{
		ContactID:   strings.TrimSpace(contactID),
		RecencyDays: recencyDays,
		Frequency:   frequency,
		Monetary:    monetary,
		RScore:      rScore,
		FScore:      fScore,
		MScore:      mScore,
		RFMTotal:    rScore + fScore + mScore,
	}, nil
}
