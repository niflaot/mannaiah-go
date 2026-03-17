package store

import (
	"context"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"mannaiah/module/analytics/domain"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(&rfmGroupRecord{}, &rfmGroupConditionRecord{}, &rfmBandConfigRecord{}); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}

	return db
}

// TestNewRFMGroupRepository_NilDB verifies constructor nil guard.
func TestNewRFMGroupRepository_NilDB(t *testing.T) {
	if _, err := NewRFMGroupRepository(nil); !errors.Is(err, ErrNilDB) {
		t.Errorf("NewRFMGroupRepository(nil) error = %v, want ErrNilDB", err)
	}
}

// TestRFMGroupRepository_CRUD verifies full create/get/list/update/delete lifecycle.
func TestRFMGroupRepository_CRUD(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewRFMGroupRepository(db)
	if err != nil {
		t.Fatalf("NewRFMGroupRepository() error = %v", err)
	}
	ctx := context.Background()

	rMin, rMax := 4, 5
	group := domain.RFMGroup{
		Name:        "Champions",
		Slug:        "champions",
		Description: "Best customers",
		Conditions:  domain.RFMGroupConditions{RMin: &rMin, RMax: &rMax},
	}

	if err := repo.Create(ctx, &group); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if group.ID == "" {
		t.Errorf("Create() did not assign ID")
	}

	got, err := repo.GetByID(ctx, group.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.Name != "Champions" {
		t.Errorf("GetByID().Name = %q, want Champions", got.Name)
	}
	if got.Conditions.RMin == nil || *got.Conditions.RMin != 4 {
		t.Errorf("GetByID().Conditions.RMin = %v, want 4", got.Conditions.RMin)
	}

	groups, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(groups) != 1 {
		t.Errorf("List() len = %d, want 1", len(groups))
	}

	got.Name = "VIPs"
	if err := repo.Update(ctx, got); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	updated, _ := repo.GetByID(ctx, group.ID)
	if updated.Name != "VIPs" {
		t.Errorf("after Update, Name = %q, want VIPs", updated.Name)
	}

	if err := repo.Delete(ctx, group.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if _, err := repo.GetByID(ctx, group.ID); !errors.Is(err, ErrRFMGroupNotFound) {
		t.Errorf("after Delete, GetByID error = %v, want ErrRFMGroupNotFound", err)
	}
}

// TestSeedDefaultBands_CreatesDefaults verifies that SeedDefaultBands creates three band configs.
func TestSeedDefaultBands_CreatesDefaults(t *testing.T) {
	db := openTestDB(t)
	repo, _ := NewRFMGroupRepository(db)
	ctx := context.Background()

	if err := repo.SeedDefaultBands(ctx); err != nil {
		t.Fatalf("SeedDefaultBands() error = %v", err)
	}

	bands, err := repo.GetBandConfigs(ctx)
	if err != nil {
		t.Fatalf("GetBandConfigs() error = %v", err)
	}
	if len(bands) != 3 {
		t.Errorf("SeedDefaultBands() created %d bands, want 3", len(bands))
	}
}

// TestSeedDefaultBands_Idempotent verifies SeedDefaultBands is safe to call multiple times.
func TestSeedDefaultBands_Idempotent(t *testing.T) {
	db := openTestDB(t)
	repo, _ := NewRFMGroupRepository(db)
	ctx := context.Background()

	_ = repo.SeedDefaultBands(ctx)
	_ = repo.SeedDefaultBands(ctx)

	bands, _ := repo.GetBandConfigs(ctx)
	if len(bands) != 3 {
		t.Errorf("after double seed, band count = %d, want 3", len(bands))
	}
}

// TestUpdateBandConfig verifies that UpdateBandConfig persists threshold changes.
func TestUpdateBandConfig(t *testing.T) {
	db := openTestDB(t)
	repo, _ := NewRFMGroupRepository(db)
	ctx := context.Background()

	_ = repo.SeedDefaultBands(ctx)

	if err := repo.UpdateBandConfig(ctx, domain.RFMBandConfig{
		Dimension: domain.DimensionRecency,
		Band5Min:  3,
		Band4Min:  14,
	}); err != nil {
		t.Fatalf("UpdateBandConfig() error = %v", err)
	}

	bands, _ := repo.GetBandConfigs(ctx)
	for _, b := range bands {
		if b.Dimension == domain.DimensionRecency && b.Band5Min != 3 {
			t.Errorf("after update, Band5Min = %v, want 3", b.Band5Min)
		}
	}
}
