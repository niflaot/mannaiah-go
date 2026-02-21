package store

import (
	"context"
	errorspkg "errors"
	"testing"

	"mannaiah/module/assets/domain"
	"mannaiah/module/assets/port"
	coredb "mannaiah/module/core/database"
	coredbmigration "mannaiah/module/core/database/migration"
)

// TestNewRepository validates repository constructor behavior.
func TestNewRepository(t *testing.T) {
	if _, err := NewRepository(nil); !errorspkg.Is(err, ErrNilDB) {
		t.Fatalf("NewRepository(nil) error = %v, want ErrNilDB", err)
	}
}

// TestRepositoryAssetCRUD verifies asset persistence behavior.
func TestRepositoryAssetCRUD(t *testing.T) {
	repository := newRepositoryForTest(t)
	ctx := context.Background()

	folder := &domain.Folder{ID: "f-1", Name: "Hero", Tags: []domain.Tag{{Name: "hero", Color: "#ff0000"}}}
	folder.Normalize()
	if err := repository.CreateFolder(ctx, folder); err != nil {
		t.Fatalf("CreateFolder() error = %v", err)
	}

	asset := &domain.Asset{
		ID:           "a-1",
		Key:          "assets/a-1.png",
		Name:         "Asset One",
		OriginalName: "one.png",
		FolderID:     folder.ID,
		MimeType:     "image/png",
		Size:         120,
		Tags:         []domain.Tag{{Name: "hero", Color: "#ff0000"}},
		Metadata:     map[string]string{"alt": "hero"},
	}
	if createErr := repository.Create(ctx, asset); createErr != nil {
		t.Fatalf("Create() error = %v", createErr)
	}

	loaded, getErr := repository.GetByID(ctx, "a-1")
	if getErr != nil {
		t.Fatalf("GetByID() error = %v", getErr)
	}
	if loaded.FolderID != folder.ID {
		t.Fatalf("loaded.FolderID = %q, want %q", loaded.FolderID, folder.ID)
	}
	if loaded.Metadata["alt"] != "hero" {
		t.Fatalf("loaded.Metadata[alt] = %q, want %q", loaded.Metadata["alt"], "hero")
	}

	page, listErr := repository.List(ctx, port.ListQuery{Page: 1, Limit: 10, Filters: "asset"})
	if listErr != nil {
		t.Fatalf("List() error = %v", listErr)
	}
	if page.Total != 1 {
		t.Fatalf("page.Total = %d, want %d", page.Total, 1)
	}

	name := "Asset Renamed"
	emptyFolder := ""
	tags := []domain.Tag{{Name: "cover", Color: "#111111"}}
	metadata := map[string]string{"alt": "cover"}
	updated, updateErr := repository.Update(ctx, "a-1", port.AssetUpdate{
		Name:     &name,
		FolderID: &emptyFolder,
		Tags:     &tags,
		Metadata: &metadata,
	})
	if updateErr != nil {
		t.Fatalf("Update() error = %v", updateErr)
	}
	if updated.Name != "Asset Renamed" {
		t.Fatalf("updated.Name = %q, want %q", updated.Name, "Asset Renamed")
	}
	if updated.FolderID != "" {
		t.Fatalf("updated.FolderID = %q, want empty", updated.FolderID)
	}

	if deleteErr := repository.SoftDelete(ctx, "a-1"); deleteErr != nil {
		t.Fatalf("SoftDelete() error = %v", deleteErr)
	}
	if _, getDeletedErr := repository.GetByID(ctx, "a-1"); !errorspkg.Is(getDeletedErr, port.ErrNotFound) {
		t.Fatalf("GetByID(deleted) error = %v, want port.ErrNotFound", getDeletedErr)
	}
}

// TestRepositoryFolderCRUD verifies folder persistence behavior.
func TestRepositoryFolderCRUD(t *testing.T) {
	repository := newRepositoryForTest(t)
	ctx := context.Background()

	folder := &domain.Folder{
		ID:   "f-1",
		Name: "  Hero Folder  ",
		Tags: []domain.Tag{{Name: "hero", Color: "#aa00ff"}},
	}
	folder.Normalize()
	if err := repository.CreateFolder(ctx, folder); err != nil {
		t.Fatalf("CreateFolder() error = %v", err)
	}

	loaded, err := repository.GetFolderByID(ctx, folder.ID)
	if err != nil {
		t.Fatalf("GetFolderByID() error = %v", err)
	}
	if loaded.Slug != "hero-folder" {
		t.Fatalf("loaded.Slug = %q, want %q", loaded.Slug, "hero-folder")
	}

	page, err := repository.ListFolders(ctx, port.ListQuery{Page: 1, Limit: 10, Filters: "hero"})
	if err != nil {
		t.Fatalf("ListFolders() error = %v", err)
	}
	if page.Total != 1 {
		t.Fatalf("page.Total = %d, want %d", page.Total, 1)
	}

	newName := "Catalog Root"
	newTags := []domain.Tag{{Name: "catalog", Color: "#123abc"}}
	updated, err := repository.UpdateFolder(ctx, folder.ID, port.FolderUpdate{Name: &newName, Tags: &newTags})
	if err != nil {
		t.Fatalf("UpdateFolder() error = %v", err)
	}
	if updated.Slug != "catalog-root" {
		t.Fatalf("updated.Slug = %q, want %q", updated.Slug, "catalog-root")
	}

	asset := &domain.Asset{
		ID:           "a-1",
		Key:          "assets/a-1.png",
		Name:         "Asset",
		OriginalName: "one.png",
		FolderID:     folder.ID,
		MimeType:     "image/png",
		Size:         1,
	}
	if err := repository.Create(ctx, asset); err != nil {
		t.Fatalf("Create(asset) error = %v", err)
	}

	child := &domain.Folder{
		ID:             "f-2",
		Name:           "Child",
		ParentFolderID: folder.ID,
		Tags:           []domain.Tag{{Name: "child", Color: "#00aa00"}},
	}
	child.Normalize()
	if err := repository.CreateFolder(ctx, child); err != nil {
		t.Fatalf("CreateFolder(child) error = %v", err)
	}
	grandchild := &domain.Folder{
		ID:             "f-3",
		Name:           "Grandchild",
		ParentFolderID: child.ID,
	}
	grandchild.Normalize()
	if err := repository.CreateFolder(ctx, grandchild); err != nil {
		t.Fatalf("CreateFolder(grandchild) error = %v", err)
	}

	childPage, err := repository.ListFolders(ctx, port.ListQuery{Page: 1, Limit: 10, ParentFolderID: folder.ID})
	if err != nil {
		t.Fatalf("ListFolders(parent) error = %v", err)
	}
	if childPage.Total != 1 {
		t.Fatalf("childPage.Total = %d, want %d", childPage.Total, 1)
	}
	if childPage.Data[0].ID != child.ID {
		t.Fatalf("childPage.Data[0].ID = %q, want %q", childPage.Data[0].ID, child.ID)
	}

	selfParent := folder.ID
	if _, err := repository.UpdateFolder(ctx, folder.ID, port.FolderUpdate{ParentFolderID: &selfParent}); !errorspkg.Is(err, domain.ErrFolderParentSelfReference) {
		t.Fatalf("UpdateFolder(self parent) error = %v, want domain.ErrFolderParentSelfReference", err)
	}
	parentToGrandchild := grandchild.ID
	if _, err := repository.UpdateFolder(ctx, folder.ID, port.FolderUpdate{ParentFolderID: &parentToGrandchild}); !errorspkg.Is(err, domain.ErrFolderParentCycle) {
		t.Fatalf("UpdateFolder(cycle) error = %v, want domain.ErrFolderParentCycle", err)
	}

	if err := repository.SoftDeleteFolder(ctx, folder.ID); err != nil {
		t.Fatalf("SoftDeleteFolder() error = %v", err)
	}

	exists, err := repository.ExistsFolder(ctx, folder.ID)
	if err != nil {
		t.Fatalf("ExistsFolder() error = %v", err)
	}
	if exists {
		t.Fatalf("ExistsFolder(deleted) = true, want false")
	}
	childExists, err := repository.ExistsFolder(ctx, child.ID)
	if err != nil {
		t.Fatalf("ExistsFolder(child) error = %v", err)
	}
	if childExists {
		t.Fatalf("ExistsFolder(child) = true, want false")
	}
	grandchildExists, err := repository.ExistsFolder(ctx, grandchild.ID)
	if err != nil {
		t.Fatalf("ExistsFolder(grandchild) error = %v", err)
	}
	if grandchildExists {
		t.Fatalf("ExistsFolder(grandchild) = true, want false")
	}

	loadedAsset, err := repository.GetByID(ctx, asset.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if loadedAsset.FolderID != "" {
		t.Fatalf("loadedAsset.FolderID = %q, want empty", loadedAsset.FolderID)
	}
}

// TestRepositoryNotFound verifies not-found behavior across operations.
func TestRepositoryNotFound(t *testing.T) {
	repository := newRepositoryForTest(t)
	ctx := context.Background()

	if _, err := repository.GetByID(ctx, "missing"); !errorspkg.Is(err, port.ErrNotFound) {
		t.Fatalf("GetByID(missing) error = %v, want port.ErrNotFound", err)
	}
	if _, err := repository.Update(ctx, "missing", port.AssetUpdate{}); !errorspkg.Is(err, port.ErrNotFound) {
		t.Fatalf("Update(missing) error = %v, want port.ErrNotFound", err)
	}
	if err := repository.SoftDelete(ctx, "missing"); !errorspkg.Is(err, port.ErrNotFound) {
		t.Fatalf("SoftDelete(missing) error = %v, want port.ErrNotFound", err)
	}
	if _, err := repository.GetFolderByID(ctx, "missing"); !errorspkg.Is(err, port.ErrNotFound) {
		t.Fatalf("GetFolderByID(missing) error = %v, want port.ErrNotFound", err)
	}
	if _, err := repository.UpdateFolder(ctx, "missing", port.FolderUpdate{}); !errorspkg.Is(err, port.ErrNotFound) {
		t.Fatalf("UpdateFolder(missing) error = %v, want port.ErrNotFound", err)
	}
	if err := repository.SoftDeleteFolder(ctx, "missing"); !errorspkg.Is(err, port.ErrNotFound) {
		t.Fatalf("SoftDeleteFolder(missing) error = %v, want port.ErrNotFound", err)
	}
}

// TestRepositoryDuplicateConstraints verifies duplicate key and slug behavior.
func TestRepositoryDuplicateConstraints(t *testing.T) {
	repository := newRepositoryForTest(t)
	ctx := context.Background()

	if err := repository.Create(ctx, &domain.Asset{ID: "a-1", Key: "assets/a-1.png", Name: "Asset", OriginalName: "one.png", MimeType: "image/png", Size: 120}); err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}
	if err := repository.Create(ctx, &domain.Asset{ID: "a-2", Key: "assets/a-1.png", Name: "Asset", OriginalName: "two.png", MimeType: "image/png", Size: 120}); err == nil {
		t.Fatalf("expected duplicate key error")
	}

	folder := &domain.Folder{ID: "f-1", Name: "Hero"}
	folder.Normalize()
	if err := repository.CreateFolder(ctx, folder); err != nil {
		t.Fatalf("CreateFolder(first) error = %v", err)
	}
	second := &domain.Folder{ID: "f-2", Name: "Hero"}
	second.Normalize()
	if err := repository.CreateFolder(ctx, second); !errorspkg.Is(err, port.ErrFolderAlreadyExists) {
		t.Fatalf("CreateFolder(duplicate root slug) error = %v, want port.ErrFolderAlreadyExists", err)
	}

	parentOne := &domain.Folder{ID: "f-parent-1", Name: "Hi"}
	parentOne.Normalize()
	if err := repository.CreateFolder(ctx, parentOne); err != nil {
		t.Fatalf("CreateFolder(parent one) error = %v", err)
	}
	parentTwo := &domain.Folder{ID: "f-parent-2", Name: "Bye"}
	parentTwo.Normalize()
	if err := repository.CreateFolder(ctx, parentTwo); err != nil {
		t.Fatalf("CreateFolder(parent two) error = %v", err)
	}

	childOne := &domain.Folder{ID: "f-child-1", Name: "Hello", ParentFolderID: parentOne.ID}
	childOne.Normalize()
	if err := repository.CreateFolder(ctx, childOne); err != nil {
		t.Fatalf("CreateFolder(child under parent one) error = %v", err)
	}
	childTwo := &domain.Folder{ID: "f-child-2", Name: "Hello", ParentFolderID: parentTwo.ID}
	childTwo.Normalize()
	if err := repository.CreateFolder(ctx, childTwo); err != nil {
		t.Fatalf("CreateFolder(child under parent two) error = %v", err)
	}
}

// TestRepositoryCreateFolderMissingParent verifies folder-parent existence validation.
func TestRepositoryCreateFolderMissingParent(t *testing.T) {
	repository := newRepositoryForTest(t)
	ctx := context.Background()

	folder := &domain.Folder{
		ID:             "f-1",
		Name:           "Orphan",
		ParentFolderID: "missing",
	}
	folder.Normalize()

	if err := repository.CreateFolder(ctx, folder); !errorspkg.Is(err, port.ErrFolderNotFound) {
		t.Fatalf("CreateFolder(missing parent) error = %v, want port.ErrFolderNotFound", err)
	}
}

// TestRepositoryFolderAssignmentValidation verifies folder-assignment validation.
func TestRepositoryFolderAssignmentValidation(t *testing.T) {
	repository := newRepositoryForTest(t)
	ctx := context.Background()
	asset := &domain.Asset{
		ID:           "a-1",
		Key:          "assets/a-1.png",
		Name:         "Asset",
		OriginalName: "one.png",
		MimeType:     "image/png",
		Size:         120,
	}
	if err := repository.Create(ctx, asset); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	folderID := "missing"
	if _, err := repository.Update(ctx, "a-1", port.AssetUpdate{FolderID: &folderID}); !errorspkg.Is(err, port.ErrFolderNotFound) {
		t.Fatalf("Update(folder missing) error = %v, want port.ErrFolderNotFound", err)
	}
}

// TestNormalizePagination verifies pagination helper defaults and limits.
func TestNormalizePagination(t *testing.T) {
	page, limit := normalizePagination(0, 0)
	if page != 1 || limit != 10 {
		t.Fatalf("normalizePagination(0,0) = (%d,%d), want (1,10)", page, limit)
	}
	_, capped := normalizePagination(1, 999)
	if capped != 100 {
		t.Fatalf("normalizePagination limit = %d, want %d", capped, 100)
	}
}

// TestEnsureSchemaNoop verifies repository EnsureSchema does not mutate schema at runtime.
func TestEnsureSchemaNoop(t *testing.T) {
	repository := newRepositoryForTest(t)
	if err := repository.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema() error = %v", err)
	}
}

// newRepositoryForTest creates a repository bound to in-memory sqlite.
func newRepositoryForTest(t *testing.T) *Repository {
	t.Helper()

	db, err := coredb.Open(coredb.Config{Driver: "sqlite", DSN: "file::memory:?cache=shared"}, nil)
	if err != nil {
		t.Fatalf("coredb.Open() error = %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	repository, err := NewRepository(db)
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	if err := coredbmigration.Apply(context.Background(), db, coredbmigration.Config{Enabled: true, Driver: "sqlite", Table: "schema_migrations"}, nil); err != nil {
		t.Fatalf("coredbmigration.Apply() error = %v", err)
	}
	if err := repository.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema() error = %v", err)
	}

	return repository
}
