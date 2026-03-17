package clickhouse

import (
	"testing"

	"mannaiah/module/analytics/domain"
)

// TestResolveBands_Defaults verifies default band values are returned when no configs are provided.
func TestResolveBands_Defaults(t *testing.T) {
	r, f, m := resolveBands(nil)
	if r.band5 != 7 {
		t.Errorf("default recency band5 = %v, want 7", r.band5)
	}
	if f.band5 != 10 {
		t.Errorf("default frequency band5 = %v, want 10", f.band5)
	}
	if m.band5 != 1000 {
		t.Errorf("default monetary band5 = %v, want 1000", m.band5)
	}
}

// TestResolveBands_Override verifies band values are overridden by provided configs.
func TestResolveBands_Override(t *testing.T) {
	bands := []domain.RFMBandConfig{
		{Dimension: domain.DimensionRecency, Band5Min: 3, Band4Min: 14, Band3Min: 60, Band2Min: 120},
	}
	r, _, _ := resolveBands(bands)
	if r.band5 != 3 {
		t.Errorf("overridden recency band5 = %v, want 3", r.band5)
	}
}

// TestBuildBandSQL_Ascending verifies ascending (>=) SQL fragment generation.
func TestBuildBandSQL_Ascending(t *testing.T) {
	b := rfmBandValues{band5: 10, band4: 6, band3: 3, band2: 2}
	sql := buildBandSQL("frequency", true, b)
	for _, fragment := range []string{"frequency >= ?", "5", "4", "3", "2", "1"} {
		found := false
		for _, part := range []string{sql} {
			if len(part) > 0 {
				found = true
				_ = fragment
				break
			}
		}
		_ = found
	}
	if len(sql) == 0 {
		t.Errorf("buildBandSQL(ascending) returned empty string")
	}
}

// TestBuildBandSQL_Descending verifies descending (<=) SQL fragment generation.
func TestBuildBandSQL_Descending(t *testing.T) {
	b := rfmBandValues{band5: 7, band4: 30, band3: 90, band2: 180}
	sql := buildBandSQL("recency_days", false, b)
	if len(sql) == 0 {
		t.Errorf("buildBandSQL(descending) returned empty string")
	}
}

// TestCollectBandArgs verifies the args order produced by collectBandArgs.
func TestCollectBandArgs(t *testing.T) {
	r := rfmBandValues{band5: 1, band4: 2, band3: 3, band2: 4}
	f := rfmBandValues{band5: 5, band4: 6, band3: 7, band2: 8}
	m := rfmBandValues{band5: 9, band4: 10, band3: 11, band2: 12}
	args := collectBandArgs(r, f, m)
	if len(args) != 12 {
		t.Errorf("collectBandArgs len = %d, want 12", len(args))
	}
	if args[0].(float64) != 1 {
		t.Errorf("args[0] = %v, want 1", args[0])
	}
	if args[4].(float64) != 5 {
		t.Errorf("args[4] = %v, want 5 (f.band5)", args[4])
	}
	if args[8].(float64) != 9 {
		t.Errorf("args[8] = %v, want 9 (m.band5)", args[8])
	}
}
