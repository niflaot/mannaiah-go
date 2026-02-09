package database

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
)

// temporalModel is a sample domain model extending the shared base model.
type temporalModel struct {
	// Model provides ID/timestamps/soft-delete fields.
	Model
	// Name defines the model display name.
	Name string
	// Status defines lifecycle state.
	Status string
}

// temporalModelService demonstrates domain-specific service extension.
type temporalModelService struct {
	// Service embeds generic CRUD behavior for temporalModel.
	*Service[temporalModel]
}

// newTemporalModelService creates an extended service over the generic implementation.
func newTemporalModelService(db *gorm.DB) (*temporalModelService, error) {
	base, err := NewService[temporalModel](db)
	if err != nil {
		return nil, err
	}

	return &temporalModelService{Service: base}, nil
}

// FindByStatus returns temporal models filtered by status.
func (s *temporalModelService) FindByStatus(ctx context.Context, status string) ([]temporalModel, error) {
	return s.Find(ctx, Query{
		Where: "status = ?",
		Args:  []any{status},
		Order: "id asc",
	})
}

// TestNewServiceRejectsNilDB verifies constructor validation for nil DB dependencies.
func TestNewServiceRejectsNilDB(t *testing.T) {
	_, err := NewService[temporalModel](nil)
	if !errors.Is(err, ErrNilDB) {
		t.Fatalf("NewService() error = %v, want ErrNilDB", err)
	}
}

// TestCreateReadUpdateDeleteLifecycle verifies generic CRUD behavior with soft-delete.
func TestCreateReadUpdateDeleteLifecycle(t *testing.T) {
	db := newTestDB(t)
	service := newTemporalServiceForTest(t, db)

	entity := &temporalModel{Name: "alpha", Status: "active"}
	if err := service.Create(context.Background(), entity); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if entity.ID == 0 {
		t.Fatalf("expected created entity ID to be assigned")
	}
	if entity.CreatedAt.IsZero() || entity.UpdatedAt.IsZero() {
		t.Fatalf("expected timestamps to be populated")
	}

	record, err := service.Read(context.Background(), entity.ID)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if record.Name != "alpha" {
		t.Fatalf("Read().Name = %q, want %q", record.Name, "alpha")
	}

	if err := service.Update(context.Background(), entity.ID, map[string]any{"status": "archived"}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	updated, err := service.Read(context.Background(), entity.ID)
	if err != nil {
		t.Fatalf("Read() error after update = %v", err)
	}
	if updated.Status != "archived" {
		t.Fatalf("updated.Status = %q, want %q", updated.Status, "archived")
	}

	if err := service.Delete(context.Background(), entity.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := service.Read(context.Background(), entity.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Read() error after delete = %v, want ErrNotFound", err)
	}
}

// TestFindByQuery verifies filtered find behavior and pagination controls.
func TestFindByQuery(t *testing.T) {
	db := newTestDB(t)
	service := newTemporalServiceForTest(t, db)

	seedTemporal(t, service, []temporalModel{
		{Name: "a", Status: "active"},
		{Name: "b", Status: "active"},
		{Name: "c", Status: "pending"},
	})

	results, err := service.Find(context.Background(), Query{
		Where:  "status = ?",
		Args:   []any{"active"},
		Order:  "id asc",
		Limit:  1,
		Offset: 1,
	})
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Find() count = %d, want %d", len(results), 1)
	}
	if results[0].Name != "b" {
		t.Fatalf("Find()[0].Name = %q, want %q", results[0].Name, "b")
	}
}

// TestFindSupportsUnscoped verifies optional retrieval of soft-deleted records.
func TestFindSupportsUnscoped(t *testing.T) {
	db := newTestDB(t)
	service := newTemporalServiceForTest(t, db)

	entity := &temporalModel{Name: "soft", Status: "active"}
	if err := service.Create(context.Background(), entity); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := service.Delete(context.Background(), entity.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	scoped, err := service.Find(context.Background(), Query{Where: "name = ?", Args: []any{"soft"}})
	if err != nil {
		t.Fatalf("Find() scoped error = %v", err)
	}
	if len(scoped) != 0 {
		t.Fatalf("expected scoped find to hide soft-deleted rows, got %d", len(scoped))
	}

	unscoped, err := service.Find(context.Background(), Query{
		Where:    "name = ?",
		Args:     []any{"soft"},
		Unscoped: true,
	})
	if err != nil {
		t.Fatalf("Find() unscoped error = %v", err)
	}
	if len(unscoped) != 1 {
		t.Fatalf("expected unscoped find to include soft-deleted row, got %d", len(unscoped))
	}
}

// TestCreateRejectsNilEntity verifies create validation for nil entity pointers.
func TestCreateRejectsNilEntity(t *testing.T) {
	db := newTestDB(t)
	service := newTemporalServiceForTest(t, db)

	if err := service.Create(context.Background(), nil); !errors.Is(err, ErrNilEntity) {
		t.Fatalf("Create() error = %v, want ErrNilEntity", err)
	}
}

// TestReadValidationAndNotFound verifies read validation and missing-record behavior.
func TestReadValidationAndNotFound(t *testing.T) {
	db := newTestDB(t)
	service := newTemporalServiceForTest(t, db)

	if _, err := service.Read(context.Background(), 0); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("Read() error = %v, want ErrInvalidID", err)
	}
	if _, err := service.Read(context.Background(), 999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Read() error = %v, want ErrNotFound", err)
	}
}

// TestUpdateValidationAndNotFound verifies update validation and missing-record behavior.
func TestUpdateValidationAndNotFound(t *testing.T) {
	db := newTestDB(t)
	service := newTemporalServiceForTest(t, db)

	if err := service.Update(context.Background(), 0, map[string]any{"status": "x"}); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("Update() error = %v, want ErrInvalidID", err)
	}
	if err := service.Update(context.Background(), 1, map[string]any{}); !errors.Is(err, ErrEmptyUpdates) {
		t.Fatalf("Update() error = %v, want ErrEmptyUpdates", err)
	}
	if err := service.Update(context.Background(), 999, map[string]any{"status": "x"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update() error = %v, want ErrNotFound", err)
	}
}

// TestDeleteValidationAndNotFound verifies delete validation and missing-record behavior.
func TestDeleteValidationAndNotFound(t *testing.T) {
	db := newTestDB(t)
	service := newTemporalServiceForTest(t, db)

	if err := service.Delete(context.Background(), 0); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("Delete() error = %v, want ErrInvalidID", err)
	}
	if err := service.Delete(context.Background(), 999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Delete() error = %v, want ErrNotFound", err)
	}
}

// TestReadUpdateDeleteErrorPaths verifies wrapped DB errors when SQL handles are closed.
func TestReadUpdateDeleteErrorPaths(t *testing.T) {
	db := newTestDB(t)
	service := newTemporalServiceForTest(t, db)

	entity := &temporalModel{Name: "will-close", Status: "active"}
	if err := service.Create(context.Background(), entity); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("DB() error = %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if _, err := service.Read(context.Background(), entity.ID); err == nil || !strings.Contains(err.Error(), "read record id") {
		t.Fatalf("Read() error = %v, expected wrapped read error", err)
	}
	if err := service.Update(context.Background(), entity.ID, map[string]any{"status": "x"}); err == nil || !strings.Contains(err.Error(), "update record id") {
		t.Fatalf("Update() error = %v, expected wrapped update error", err)
	}
	if err := service.Delete(context.Background(), entity.ID); err == nil || !strings.Contains(err.Error(), "delete record id") {
		t.Fatalf("Delete() error = %v, expected wrapped delete error", err)
	}
}

// TestTemporalServiceExtension verifies generic service extension with custom domain method.
func TestTemporalServiceExtension(t *testing.T) {
	db := newTestDB(t)
	extended, err := newTemporalModelService(db)
	if err != nil {
		t.Fatalf("newTemporalModelService() error = %v", err)
	}

	seedTemporal(t, extended.Service, []temporalModel{
		{Name: "event-a", Status: "scheduled"},
		{Name: "event-b", Status: "scheduled"},
		{Name: "event-c", Status: "done"},
	})

	scheduled, err := extended.FindByStatus(context.Background(), "scheduled")
	if err != nil {
		t.Fatalf("FindByStatus() error = %v", err)
	}
	if len(scheduled) != 2 {
		t.Fatalf("FindByStatus() count = %d, want %d", len(scheduled), 2)
	}
}

// TestFindPreloadErrorPath verifies preload errors are wrapped by find.
func TestFindPreloadErrorPath(t *testing.T) {
	db := newTestDB(t)
	service := newTemporalServiceForTest(t, db)
	seedTemporal(t, service, []temporalModel{{Name: "x", Status: "active"}})

	_, err := service.Find(context.Background(), Query{Preloads: []string{"UnknownRelation"}})
	if err == nil {
		t.Fatalf("expected preload error")
	}
	if !strings.Contains(err.Error(), "find records") {
		t.Fatalf("expected wrapped find error, got %q", err.Error())
	}
}

// newTestDB creates an in-memory SQLite database and migrates test models.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := Open(
		Config{
			Driver: "sqlite",
			DSN:    "file::memory:?cache=shared",
		},
		nil,
	)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	if migrateErr := db.AutoMigrate(&temporalModel{}); migrateErr != nil {
		t.Fatalf("AutoMigrate() error = %v", migrateErr)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("DB() error = %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	return db
}

// newTemporalServiceForTest creates a typed service and fails the test on constructor error.
func newTemporalServiceForTest(t *testing.T, db *gorm.DB) *Service[temporalModel] {
	t.Helper()

	service, err := NewService[temporalModel](db)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	return service
}

// seedTemporal inserts temporal models for test setup.
func seedTemporal(t *testing.T, service *Service[temporalModel], models []temporalModel) {
	t.Helper()

	for index := range models {
		if err := service.Create(context.Background(), &models[index]); err != nil {
			t.Fatalf("Create() seed error = %v", err)
		}
	}
}

// TestCreateContextCancellation verifies context cancellation propagates through create.
func TestCreateContextCancellation(t *testing.T) {
	db := newTestDB(t)
	service := newTemporalServiceForTest(t, db)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := service.Create(ctx, &temporalModel{Name: "ctx", Status: "cancelled"})
	if err == nil {
		t.Fatalf("expected context-cancelled create error")
	}
	if !strings.Contains(err.Error(), "create record") {
		t.Fatalf("expected wrapped create error, got %q", err.Error())
	}
}

// TestUpdatedAtChangesOnUpdate verifies UpdatedAt timestamp changes on update operations.
func TestUpdatedAtChangesOnUpdate(t *testing.T) {
	db := newTestDB(t)
	service := newTemporalServiceForTest(t, db)

	entity := &temporalModel{Name: "timed", Status: "old"}
	if err := service.Create(context.Background(), entity); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	createdUpdatedAt := entity.UpdatedAt
	time.Sleep(5 * time.Millisecond)

	if err := service.Update(context.Background(), entity.ID, map[string]any{"status": "new"}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	record, err := service.Read(context.Background(), entity.ID)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if !record.UpdatedAt.After(createdUpdatedAt) {
		t.Fatalf("expected UpdatedAt to advance after update")
	}
}
