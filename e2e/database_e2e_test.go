package e2e_test

import (
	"context"
	"testing"

	coredatabase "mannaiah/module/core/database"
)

// catalogEntity defines a database model used by CRUD and pagination E2E scenarios.
type catalogEntity struct {
	// Model defines shared primary key and timestamp fields.
	coredatabase.Model
	// Name defines catalog entity names.
	Name string
}

// TestDatabaseCRUDPaginationE2E verifies generic DB CRUD and pagination behavior end-to-end.
func TestDatabaseCRUDPaginationE2E(t *testing.T) {
	tracer := newStepTracer(t)

	tracer.Step("open sqlite database")
	db, err := coredatabase.Open(coredatabase.Config{Driver: "sqlite", DSN: "file::memory:?cache=shared"}, tracer.logger)
	if err != nil {
		t.Fatalf("coredatabase.Open() error = %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}
	defer func() {
		_ = sqlDB.Close()
	}()

	tracer.Step("migrate catalog schema")
	if err := db.AutoMigrate(&catalogEntity{}); err != nil {
		t.Fatalf("db.AutoMigrate() error = %v", err)
	}

	tracer.Step("initialize generic crud service")
	service, err := coredatabase.NewService[catalogEntity](db)
	if err != nil {
		t.Fatalf("coredatabase.NewService() error = %v", err)
	}

	ctx := context.Background()

	tracer.Step("create catalog entities")
	first := &catalogEntity{Name: "first"}
	second := &catalogEntity{Name: "second"}
	third := &catalogEntity{Name: "third"}
	if err := service.Create(ctx, first); err != nil {
		t.Fatalf("service.Create(first) error = %v", err)
	}
	if err := service.Create(ctx, second); err != nil {
		t.Fatalf("service.Create(second) error = %v", err)
	}
	if err := service.Create(ctx, third); err != nil {
		t.Fatalf("service.Create(third) error = %v", err)
	}

	tracer.Step("read entity")
	readResult, err := service.Read(ctx, first.ID)
	if err != nil {
		t.Fatalf("service.Read() error = %v", err)
	}
	if readResult.Name != "first" {
		t.Fatalf("readResult.Name = %q, want %q", readResult.Name, "first")
	}

	tracer.Step("update entity")
	if err := service.Update(ctx, second.ID, map[string]any{"name": "second-updated"}); err != nil {
		t.Fatalf("service.Update() error = %v", err)
	}

	tracer.Step("paginate with exclusion")
	pageResult, err := service.Paginate(ctx, coredatabase.Query{
		Page:       1,
		PageSize:   2,
		Order:      "id asc",
		ExcludeIDs: []uint{first.ID},
	})
	if err != nil {
		t.Fatalf("service.Paginate() error = %v", err)
	}
	if pageResult.Total != 2 {
		t.Fatalf("pageResult.Total = %d, want %d", pageResult.Total, 2)
	}
	if len(pageResult.Data) != 2 {
		t.Fatalf("len(pageResult.Data) = %d, want %d", len(pageResult.Data), 2)
	}

	tracer.Step("delete entity")
	if err := service.Delete(ctx, third.ID); err != nil {
		t.Fatalf("service.Delete() error = %v", err)
	}

	tracer.Step("assert e2e trace logs")
	tracer.AssertStepCount(8)
}
