package store

import (
	"context"
	"fmt"
	"testing"

	"mannaiah/module/contacts/domain"
	"mannaiah/module/contacts/port"
	coredb "mannaiah/module/core/database"
	coredbmigration "mannaiah/module/core/database/migration"

	"gorm.io/gorm"
)

// BenchmarkRepositoryCreate measures repository write throughput for contact creation.
func BenchmarkRepositoryCreate(b *testing.B) {
	repository := newRepositoryForBenchmark(b)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for index := 0; index < b.N; index++ {
		entity := &domain.Contact{
			Email:          fmt.Sprintf("bench-create-%d@example.com", index),
			FirstName:      "Bench",
			LastName:       "Create",
			DocumentType:   domain.DocumentTypeCC,
			DocumentNumber: fmt.Sprintf("create-%d", index),
		}
		if err := repository.Create(ctx, entity); err != nil {
			b.Fatalf("Create() error = %v", err)
		}
	}
}

// BenchmarkRepositoryList measures repository paginated query throughput under moderate dataset sizes.
func BenchmarkRepositoryList(b *testing.B) {
	repository := newRepositoryForBenchmark(b)
	ctx := context.Background()

	for index := 0; index < 2000; index++ {
		entity := &domain.Contact{
			Email:          fmt.Sprintf("bench-list-%d@example.com", index),
			FirstName:      "Bench",
			LastName:       "List",
			DocumentType:   domain.DocumentTypeTI,
			DocumentNumber: fmt.Sprintf("list-%d", index),
		}
		if err := repository.Create(ctx, entity); err != nil {
			b.Fatalf("Create() seed error = %v", err)
		}
	}

	query := port.ListQuery{
		Page:     5,
		Limit:    25,
		OrderBy:  "createdAt",
		OrderDir: "desc",
	}

	b.ReportAllocs()
	b.ResetTimer()

	for index := 0; index < b.N; index++ {
		if _, _, err := repository.List(ctx, query); err != nil {
			b.Fatalf("List() error = %v", err)
		}
	}
}

// newRepositoryForBenchmark creates a schema-ready repository for benchmark runs.
func newRepositoryForBenchmark(b *testing.B) *Repository {
	b.Helper()

	db := newDBForBenchmark(b)
	if err := coredbmigration.Apply(context.Background(), db, coredbmigration.Config{Enabled: true, Driver: "sqlite", Table: "schema_migrations"}, nil); err != nil {
		b.Fatalf("coredbmigration.Apply() error = %v", err)
	}
	repository, err := NewRepository(db)
	if err != nil {
		b.Fatalf("NewRepository() error = %v", err)
	}
	if err := repository.EnsureSchema(context.Background()); err != nil {
		b.Fatalf("EnsureSchema() error = %v", err)
	}

	return repository
}

// newDBForBenchmark creates an in-memory sqlite DB for benchmark runs.
func newDBForBenchmark(b *testing.B) *gorm.DB {
	b.Helper()

	db, err := coredb.Open(coredb.Config{Driver: "sqlite", DSN: "file::memory:?cache=shared"}, nil)
	if err != nil {
		b.Fatalf("Open() error = %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		b.Fatalf("DB() error = %v", err)
	}
	b.Cleanup(func() {
		_ = sqlDB.Close()
	})

	return db
}
