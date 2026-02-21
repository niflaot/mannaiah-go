package application

import (
	"context"
	errorspkg "errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"mannaiah/module/assets/domain"
	"mannaiah/module/assets/port"
)

// repositoryMock defines repository behavior for service tests.
type repositoryMock struct {
	// createFn defines create behavior.
	createFn func(ctx context.Context, asset *domain.Asset) error
	// getByIDFn defines get behavior.
	getByIDFn func(ctx context.Context, id string) (*domain.Asset, error)
	// listFn defines list behavior.
	listFn func(ctx context.Context, query port.ListQuery) (*port.PageResult, error)
	// updateFn defines update behavior.
	updateFn func(ctx context.Context, id string, update port.AssetUpdate) (*domain.Asset, error)
	// softDeleteFn defines soft-delete behavior.
	softDeleteFn func(ctx context.Context, id string) error
	// createFolderFn defines folder-create behavior.
	createFolderFn func(ctx context.Context, folder *domain.Folder) error
	// getFolderByIDFn defines folder-get behavior.
	getFolderByIDFn func(ctx context.Context, id string) (*domain.Folder, error)
	// listFoldersFn defines folder-list behavior.
	listFoldersFn func(ctx context.Context, query port.ListQuery) (*port.FolderPageResult, error)
	// listAllFoldersFn defines full-folder listing behavior for tree construction.
	listAllFoldersFn func(ctx context.Context) ([]domain.Folder, error)
	// updateFolderFn defines folder-update behavior.
	updateFolderFn func(ctx context.Context, id string, update port.FolderUpdate) (*domain.Folder, error)
	// softDeleteFolderFn defines folder-delete behavior.
	softDeleteFolderFn func(ctx context.Context, id string) error
	// existsFolderFn defines folder-exists behavior.
	existsFolderFn func(ctx context.Context, id string) (bool, error)
}

// EnsureSchema ignores schema behavior for service tests.
func (m repositoryMock) EnsureSchema(ctx context.Context) error { return nil }

// Create executes configured create behavior.
func (m repositoryMock) Create(ctx context.Context, asset *domain.Asset) error {
	return m.createFn(ctx, asset)
}

// GetByID executes configured get behavior.
func (m repositoryMock) GetByID(ctx context.Context, id string) (*domain.Asset, error) {
	return m.getByIDFn(ctx, id)
}

// List executes configured list behavior.
func (m repositoryMock) List(ctx context.Context, query port.ListQuery) (*port.PageResult, error) {
	return m.listFn(ctx, query)
}

// Update executes configured update behavior.
func (m repositoryMock) Update(ctx context.Context, id string, update port.AssetUpdate) (*domain.Asset, error) {
	return m.updateFn(ctx, id, update)
}

// SoftDelete executes configured soft-delete behavior.
func (m repositoryMock) SoftDelete(ctx context.Context, id string) error {
	return m.softDeleteFn(ctx, id)
}

// CreateFolder executes configured folder-create behavior.
func (m repositoryMock) CreateFolder(ctx context.Context, folder *domain.Folder) error {
	return m.createFolderFn(ctx, folder)
}

// GetFolderByID executes configured folder-get behavior.
func (m repositoryMock) GetFolderByID(ctx context.Context, id string) (*domain.Folder, error) {
	return m.getFolderByIDFn(ctx, id)
}

// ListFolders executes configured folder-list behavior.
func (m repositoryMock) ListFolders(ctx context.Context, query port.ListQuery) (*port.FolderPageResult, error) {
	return m.listFoldersFn(ctx, query)
}

// ListAllFolders executes configured full-folder listing behavior.
func (m repositoryMock) ListAllFolders(ctx context.Context) ([]domain.Folder, error) {
	return m.listAllFoldersFn(ctx)
}

// UpdateFolder executes configured folder-update behavior.
func (m repositoryMock) UpdateFolder(ctx context.Context, id string, update port.FolderUpdate) (*domain.Folder, error) {
	return m.updateFolderFn(ctx, id, update)
}

// SoftDeleteFolder executes configured folder-delete behavior.
func (m repositoryMock) SoftDeleteFolder(ctx context.Context, id string) error {
	return m.softDeleteFolderFn(ctx, id)
}

// ExistsFolder executes configured folder-exists behavior.
func (m repositoryMock) ExistsFolder(ctx context.Context, id string) (bool, error) {
	return m.existsFolderFn(ctx, id)
}

// storageMock defines storage behavior for service tests.
type storageMock struct {
	// uploadFn defines upload behavior.
	uploadFn func(ctx context.Context, request port.UploadRequest) error
	// deleteFn defines delete behavior.
	deleteFn func(ctx context.Context, key string) error
	// existsFn defines exists behavior.
	existsFn func(ctx context.Context, key string) (bool, error)
	// availabilityErr defines availability behavior.
	availabilityErr error
}

// Upload executes configured upload behavior.
func (m storageMock) Upload(ctx context.Context, request port.UploadRequest) error {
	return m.uploadFn(ctx, request)
}

// Delete executes configured delete behavior.
func (m storageMock) Delete(ctx context.Context, key string) error {
	return m.deleteFn(ctx, key)
}

// Exists executes configured exists behavior.
func (m storageMock) Exists(ctx context.Context, key string) (bool, error) {
	return m.existsFn(ctx, key)
}

// AvailabilityError returns configured availability behavior.
func (m storageMock) AvailabilityError() error {
	return m.availabilityErr
}

// publisherMock defines event publication behavior for service tests.
type publisherMock struct {
	// publishFn defines publish behavior.
	publishFn func(ctx context.Context, event port.IntegrationEvent) error
}

// Publish executes configured publish behavior.
func (m publisherMock) Publish(ctx context.Context, event port.IntegrationEvent) error {
	return m.publishFn(ctx, event)
}

// TestNewService validates service constructor behavior.
func TestNewService(t *testing.T) {
	repository := newRepositoryMock()
	storage := newStorageMock()

	if _, err := NewService(nil, storage); !errorspkg.Is(err, ErrNilRepository) {
		t.Fatalf("NewService(nil, storage) error = %v, want ErrNilRepository", err)
	}
	if _, err := NewService(repository, nil); !errorspkg.Is(err, ErrNilStorage) {
		t.Fatalf("NewService(repository, nil) error = %v, want ErrNilStorage", err)
	}

	service, err := NewService(repository, storage)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if service == nil {
		t.Fatalf("expected non-nil service")
	}
}

// TestCreateValidationAndSuccess verifies create behavior for assets.
func TestCreateValidationAndSuccess(t *testing.T) {
	var uploadedKey string
	var publishedTopic string
	service, err := NewService(
		newRepositoryMockWith(repositoryMock{
			createFn: func(ctx context.Context, asset *domain.Asset) error { return nil },
			existsFolderFn: func(ctx context.Context, id string) (bool, error) {
				return id == "f-1", nil
			},
		}),
		newStorageMock(),
		publisherMock{publishFn: func(ctx context.Context, event port.IntegrationEvent) error {
			publishedTopic = event.Topic
			return nil
		}},
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	service.storage = storageMock{
		uploadFn: func(ctx context.Context, request port.UploadRequest) error {
			uploadedKey = request.Key
			return nil
		},
		deleteFn: func(ctx context.Context, key string) error { return nil },
		existsFn: func(ctx context.Context, key string) (bool, error) { return true, nil },
	}

	if _, createErr := service.Create(context.Background(), CreateCommand{}); !errorspkg.Is(createErr, ErrFileRequired) {
		t.Fatalf("Create(empty) error = %v, want ErrFileRequired", createErr)
	}

	if _, createErr := service.Create(context.Background(), CreateCommand{Body: []byte("a"), Size: 11 * 1024 * 1024}); !errorspkg.Is(createErr, ErrFileTooLarge) {
		t.Fatalf("Create(large) error = %v, want ErrFileTooLarge", createErr)
	}

	if _, createErr := service.Create(context.Background(), CreateCommand{
		OriginalName: "a.png",
		MimeType:     "image/png",
		Body:         []byte("p"),
		FolderID:     "missing",
	}); !errorspkg.Is(createErr, port.ErrFolderNotFound) {
		t.Fatalf("Create(folder missing) error = %v, want port.ErrFolderNotFound", createErr)
	}

	entity, createErr := service.Create(context.Background(), CreateCommand{
		Name:         " Hero ",
		OriginalName: "image.png",
		FolderID:     "f-1",
		MimeType:     "image/png",
		Body:         []byte("payload"),
		Tags:         []domain.Tag{{Name: "hero", Color: "#ff0000"}},
		Metadata:     map[string]string{"alt": "hero"},
	})
	if createErr != nil {
		t.Fatalf("Create() error = %v", createErr)
	}
	if entity == nil {
		t.Fatalf("expected created entity")
	}
	if !strings.HasPrefix(uploadedKey, "assets/") {
		t.Fatalf("uploadedKey = %q, want assets/ prefix", uploadedKey)
	}
	if publishedTopic != TopicAssetCreated {
		t.Fatalf("publishedTopic = %q, want %q", publishedTopic, TopicAssetCreated)
	}
}

// TestCreateRollbackAndPublishFailures verifies create rollback and publish failures.
func TestCreateRollbackAndPublishFailures(t *testing.T) {
	repositoryErr := errorspkg.New("db failed")
	rollbackCalled := false

	service, err := NewService(
		newRepositoryMockWith(repositoryMock{
			createFn: func(ctx context.Context, asset *domain.Asset) error { return repositoryErr },
		}),
		storageMock{
			uploadFn: func(ctx context.Context, request port.UploadRequest) error { return nil },
			deleteFn: func(ctx context.Context, key string) error {
				rollbackCalled = true
				return nil
			},
			existsFn: func(ctx context.Context, key string) (bool, error) { return true, nil },
		},
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, createErr := service.Create(context.Background(), CreateCommand{OriginalName: "a.png", MimeType: "image/png", Body: []byte("p")})
	if !errorspkg.Is(createErr, repositoryErr) {
		t.Fatalf("Create() error = %v, want repositoryErr", createErr)
	}
	if !rollbackCalled {
		t.Fatalf("expected rollback delete call")
	}

	publishErr := errorspkg.New("publish failed")
	service, err = NewService(
		newRepositoryMock(),
		newStorageMock(),
		publisherMock{publishFn: func(ctx context.Context, event port.IntegrationEvent) error { return publishErr }},
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	_, createErr = service.Create(context.Background(), CreateCommand{OriginalName: "a.png", MimeType: "image/png", Body: []byte("p")})
	if !errorspkg.Is(createErr, publishErr) {
		t.Fatalf("Create() error = %v, want publishErr", createErr)
	}
}

// TestGetListUpdateDeleteExists verifies non-create asset operations.
func TestGetListUpdateDeleteExists(t *testing.T) {
	entity := &domain.Asset{ID: "a-1", Key: "assets/a-1.png", Name: "name", OriginalName: "a.png", MimeType: "image/png", Size: 10}
	service, err := NewService(
		newRepositoryMockWith(repositoryMock{
			getByIDFn: func(ctx context.Context, id string) (*domain.Asset, error) {
				if id == "missing" {
					return nil, port.ErrNotFound
				}
				return entity, nil
			},
			listFn: func(ctx context.Context, query port.ListQuery) (*port.PageResult, error) {
				return &port.PageResult{Data: []domain.Asset{*entity}, Total: 1, Page: query.Page, Limit: query.Limit}, nil
			},
			updateFn: func(ctx context.Context, id string, update port.AssetUpdate) (*domain.Asset, error) {
				updated := *entity
				if update.Name != nil {
					updated.Name = *update.Name
				}
				if update.FolderID != nil {
					updated.FolderID = *update.FolderID
				}
				return &updated, nil
			},
			softDeleteFn: func(ctx context.Context, id string) error { return nil },
			existsFolderFn: func(ctx context.Context, id string) (bool, error) {
				return id != "missing-folder", nil
			},
		}),
		newStorageMock(),
		publisherMock{publishFn: func(ctx context.Context, event port.IntegrationEvent) error { return nil }},
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if _, getErr := service.Get(context.Background(), ""); !errorspkg.Is(getErr, ErrInvalidID) {
		t.Fatalf("Get(empty) error = %v, want ErrInvalidID", getErr)
	}
	if _, getErr := service.Get(context.Background(), "a-1"); getErr != nil {
		t.Fatalf("Get() error = %v", getErr)
	}

	listed, listErr := service.List(context.Background(), ListQuery{Page: 1, Limit: 10, Filters: "img"})
	if listErr != nil {
		t.Fatalf("List() error = %v", listErr)
	}
	if listed.Total != 1 {
		t.Fatalf("listed.Total = %d, want %d", listed.Total, 1)
	}

	if _, updateErr := service.Update(context.Background(), "a-1", UpdateCommand{Name: ptr(" ")}); !errorspkg.Is(updateErr, ErrInvalidName) {
		t.Fatalf("Update(empty name) error = %v, want ErrInvalidName", updateErr)
	}
	if _, updateErr := service.Update(context.Background(), "a-1", UpdateCommand{FolderID: ptr("missing-folder")}); !errorspkg.Is(updateErr, port.ErrFolderNotFound) {
		t.Fatalf("Update(folder missing) error = %v, want port.ErrFolderNotFound", updateErr)
	}
	updated, updateErr := service.Update(context.Background(), "a-1", UpdateCommand{Name: ptr("updated"), FolderID: ptr("f-1")})
	if updateErr != nil {
		t.Fatalf("Update() error = %v", updateErr)
	}
	if updated.Name != "updated" {
		t.Fatalf("updated.Name = %q, want %q", updated.Name, "updated")
	}

	if _, updateErr := service.UpdateName(context.Background(), "a-1", "updated-2"); updateErr != nil {
		t.Fatalf("UpdateName() error = %v", updateErr)
	}

	if deleteErr := service.Delete(context.Background(), "a-1"); deleteErr != nil {
		t.Fatalf("Delete() error = %v", deleteErr)
	}
	if deleteErr := service.Delete(context.Background(), ""); !errorspkg.Is(deleteErr, ErrInvalidID) {
		t.Fatalf("Delete(empty) error = %v, want ErrInvalidID", deleteErr)
	}

	exists, existsErr := service.Exists(context.Background(), "a-1")
	if existsErr != nil {
		t.Fatalf("Exists() error = %v", existsErr)
	}
	if !exists {
		t.Fatalf("Exists() = false, want true")
	}
	missing, missingErr := service.Exists(context.Background(), "missing")
	if missingErr != nil {
		t.Fatalf("Exists(missing) error = %v", missingErr)
	}
	if missing {
		t.Fatalf("Exists(missing) = true, want false")
	}
}

// TestFolderOperations verifies folder create/read/list/update/delete behavior.
func TestFolderOperations(t *testing.T) {
	folder := &domain.Folder{ID: "f-1", Name: "Hero", Slug: "hero"}
	child := &domain.Folder{ID: "f-2", Name: "Child", Slug: "child", ParentFolderID: "f-1"}
	service, err := NewService(
		newRepositoryMockWith(repositoryMock{
			createFolderFn: func(ctx context.Context, value *domain.Folder) error {
				folder = value
				return nil
			},
			getFolderByIDFn: func(ctx context.Context, id string) (*domain.Folder, error) {
				if id == "f-2" {
					return child, nil
				}
				if id == "missing" {
					return nil, port.ErrNotFound
				}
				return folder, nil
			},
			listFoldersFn: func(ctx context.Context, query port.ListQuery) (*port.FolderPageResult, error) {
				if query.ParentFolderID == "f-1" {
					return &port.FolderPageResult{Data: []domain.Folder{*child}, Total: 1, Page: query.Page, Limit: query.Limit}, nil
				}
				return &port.FolderPageResult{Data: []domain.Folder{*folder}, Total: 1, Page: query.Page, Limit: query.Limit}, nil
			},
			listAllFoldersFn: func(ctx context.Context) ([]domain.Folder, error) {
				return []domain.Folder{
					{ID: "f-1", Name: "Hero", Slug: "hero"},
					{ID: "f-2", Name: "Child", Slug: "child", ParentFolderID: "f-1"},
				}, nil
			},
			updateFolderFn: func(ctx context.Context, id string, update port.FolderUpdate) (*domain.Folder, error) {
				if update.Name != nil {
					folder.Name = *update.Name
				}
				if update.ParentFolderID != nil {
					folder.ParentFolderID = *update.ParentFolderID
				}
				return folder, nil
			},
			softDeleteFolderFn: func(ctx context.Context, id string) error { return nil },
		}),
		newStorageMock(),
		publisherMock{publishFn: func(ctx context.Context, event port.IntegrationEvent) error { return nil }},
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if _, createErr := service.CreateFolder(context.Background(), CreateFolderCommand{}); !errorspkg.Is(createErr, ErrInvalidFolderName) {
		t.Fatalf("CreateFolder(empty) error = %v, want ErrInvalidFolderName", createErr)
	}
	if _, createErr := service.CreateFolder(context.Background(), CreateFolderCommand{Name: "Hero", ParentFolderID: "missing"}); !errorspkg.Is(createErr, port.ErrFolderNotFound) {
		t.Fatalf("CreateFolder(missing parent) error = %v, want port.ErrFolderNotFound", createErr)
	}

	created, createErr := service.CreateFolder(context.Background(), CreateFolderCommand{Name: " Hero ", ParentFolderID: "f-1"})
	if createErr != nil {
		t.Fatalf("CreateFolder() error = %v", createErr)
	}
	if created.Name != "Hero" {
		t.Fatalf("created.Name = %q, want %q", created.Name, "Hero")
	}
	if created.ParentFolderID != "f-1" {
		t.Fatalf("created.ParentFolderID = %q, want %q", created.ParentFolderID, "f-1")
	}

	if _, getErr := service.GetFolder(context.Background(), ""); !errorspkg.Is(getErr, ErrInvalidFolderID) {
		t.Fatalf("GetFolder(empty) error = %v, want ErrInvalidFolderID", getErr)
	}
	if _, getErr := service.GetFolder(context.Background(), "f-1"); getErr != nil {
		t.Fatalf("GetFolder() error = %v", getErr)
	}

	listed, listErr := service.ListFolders(context.Background(), ListQuery{Page: 1, Limit: 10, ParentFolderID: "f-1"})
	if listErr != nil {
		t.Fatalf("ListFolders() error = %v", listErr)
	}
	if listed.Total != 1 {
		t.Fatalf("listed.Total = %d, want %d", listed.Total, 1)
	}

	tree, treeErr := service.GetFolderTree(context.Background())
	if treeErr != nil {
		t.Fatalf("GetFolderTree() error = %v", treeErr)
	}
	if len(tree) != 1 {
		t.Fatalf("len(tree) = %d, want %d", len(tree), 1)
	}
	if len(tree[0].Children) != 1 || tree[0].Children[0].ID != child.ID {
		t.Fatalf("tree[0].Children = %#v, want child id %q", tree[0].Children, child.ID)
	}

	if _, updateErr := service.UpdateFolder(context.Background(), "f-1", UpdateFolderCommand{Name: ptr(" ")}); !errorspkg.Is(updateErr, ErrInvalidFolderName) {
		t.Fatalf("UpdateFolder(empty name) error = %v, want ErrInvalidFolderName", updateErr)
	}
	if _, updateErr := service.UpdateFolder(context.Background(), "f-1", UpdateFolderCommand{ParentFolderID: ptr("f-1")}); !errorspkg.Is(updateErr, ErrInvalidFolderParent) {
		t.Fatalf("UpdateFolder(self parent) error = %v, want ErrInvalidFolderParent", updateErr)
	}
	if _, updateErr := service.UpdateFolder(context.Background(), "f-1", UpdateFolderCommand{ParentFolderID: ptr("f-2")}); !errorspkg.Is(updateErr, ErrInvalidFolderParent) {
		t.Fatalf("UpdateFolder(cycle) error = %v, want ErrInvalidFolderParent", updateErr)
	}
	if _, updateErr := service.UpdateFolder(context.Background(), "f-1", UpdateFolderCommand{Name: ptr("Catalog")}); updateErr != nil {
		t.Fatalf("UpdateFolder() error = %v", updateErr)
	}

	if deleteErr := service.DeleteFolder(context.Background(), ""); !errorspkg.Is(deleteErr, ErrInvalidFolderID) {
		t.Fatalf("DeleteFolder(empty) error = %v, want ErrInvalidFolderID", deleteErr)
	}
	if deleteErr := service.DeleteFolder(context.Background(), "f-1"); deleteErr != nil {
		t.Fatalf("DeleteFolder() error = %v", deleteErr)
	}
}

// TestGetFolderTreeHandlesCycles verifies tree construction degrades gracefully on cyclic records.
func TestGetFolderTreeHandlesCycles(t *testing.T) {
	service, err := NewService(
		newRepositoryMockWith(repositoryMock{
			listAllFoldersFn: func(ctx context.Context) ([]domain.Folder, error) {
				return []domain.Folder{
					{ID: "f-1", Name: "One", Slug: "one", ParentFolderID: "f-2"},
					{ID: "f-2", Name: "Two", Slug: "two", ParentFolderID: "f-1"},
				}, nil
			},
		}),
		newStorageMock(),
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	tree, treeErr := service.GetFolderTree(context.Background())
	if treeErr != nil {
		t.Fatalf("GetFolderTree() error = %v", treeErr)
	}
	if len(tree) != 1 {
		t.Fatalf("len(tree) = %d, want %d", len(tree), 1)
	}
	if tree[0].ID != "f-1" {
		t.Fatalf("tree[0].ID = %q, want %q", tree[0].ID, "f-1")
	}
	if len(tree[0].Children) != 1 || tree[0].Children[0].ID != "f-2" {
		t.Fatalf("tree children = %#v, want one child f-2", tree[0].Children)
	}
	if len(tree[0].Children[0].Children) != 0 {
		t.Fatalf("expected cycle to be truncated, got %#v", tree[0].Children[0].Children)
	}
}

// TestCreateWithStorageUnavailable verifies availability failure behavior.
func TestCreateWithStorageUnavailable(t *testing.T) {
	service, err := NewService(newRepositoryMock(), storageMock{
		uploadFn:        func(ctx context.Context, request port.UploadRequest) error { return nil },
		deleteFn:        func(ctx context.Context, key string) error { return nil },
		existsFn:        func(ctx context.Context, key string) (bool, error) { return true, nil },
		availabilityErr: errorspkg.New("disabled"),
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, createErr := service.Create(context.Background(), CreateCommand{
		OriginalName: "file.png",
		MimeType:     "image/png",
		Body:         []byte("payload"),
	})
	if !errorspkg.Is(createErr, ErrStorageUnavailable) {
		t.Fatalf("Create() error = %v, want ErrStorageUnavailable", createErr)
	}
}

// TestUpdateLocksByAssetID verifies per-asset write serialization behavior.
func TestUpdateLocksByAssetID(t *testing.T) {
	var active int32
	var maxActive int32

	service, err := NewService(
		newRepositoryMockWith(repositoryMock{
			updateFn: func(ctx context.Context, id string, update port.AssetUpdate) (*domain.Asset, error) {
				current := atomic.AddInt32(&active, 1)
				updateMax(&maxActive, current)
				time.Sleep(10 * time.Millisecond)
				atomic.AddInt32(&active, -1)
				return &domain.Asset{ID: id, Name: "ok"}, nil
			},
		}),
		newStorageMock(),
		publisherMock{publishFn: func(ctx context.Context, event port.IntegrationEvent) error { return nil }},
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	runConcurrent(8, func() {
		_, callErr := service.Update(context.Background(), "a-1", UpdateCommand{Name: ptr("name")})
		if callErr != nil {
			t.Errorf("Update() error = %v", callErr)
		}
	})
	if maxActive != 1 {
		t.Fatalf("maxActive = %d, want %d", maxActive, 1)
	}
}

// TestUpdateFolderLocksByFolderID verifies per-folder write serialization behavior.
func TestUpdateFolderLocksByFolderID(t *testing.T) {
	var active int32
	var maxActive int32

	service, err := NewService(
		newRepositoryMockWith(repositoryMock{
			updateFolderFn: func(ctx context.Context, id string, update port.FolderUpdate) (*domain.Folder, error) {
				current := atomic.AddInt32(&active, 1)
				updateMax(&maxActive, current)
				time.Sleep(10 * time.Millisecond)
				atomic.AddInt32(&active, -1)
				return &domain.Folder{ID: id, Name: "ok", Slug: "ok"}, nil
			},
		}),
		newStorageMock(),
		publisherMock{publishFn: func(ctx context.Context, event port.IntegrationEvent) error { return nil }},
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	runConcurrent(8, func() {
		_, callErr := service.UpdateFolder(context.Background(), "f-1", UpdateFolderCommand{Name: ptr("name")})
		if callErr != nil {
			t.Errorf("UpdateFolder() error = %v", callErr)
		}
	})
	if maxActive != 1 {
		t.Fatalf("maxActive = %d, want %d", maxActive, 1)
	}
}

// newRepositoryMock creates repository mocks with default behaviors.
func newRepositoryMock() repositoryMock {
	return newRepositoryMockWith(repositoryMock{})
}

// newRepositoryMockWith creates repository mocks applying overrides.
func newRepositoryMockWith(overrides repositoryMock) repositoryMock {
	mock := repositoryMock{
		createFn:  func(ctx context.Context, asset *domain.Asset) error { return nil },
		getByIDFn: func(ctx context.Context, id string) (*domain.Asset, error) { return &domain.Asset{ID: id}, nil },
		listFn: func(ctx context.Context, query port.ListQuery) (*port.PageResult, error) {
			return &port.PageResult{}, nil
		},
		updateFn: func(ctx context.Context, id string, update port.AssetUpdate) (*domain.Asset, error) {
			return &domain.Asset{ID: id}, nil
		},
		softDeleteFn:   func(ctx context.Context, id string) error { return nil },
		createFolderFn: func(ctx context.Context, folder *domain.Folder) error { return nil },
		getFolderByIDFn: func(ctx context.Context, id string) (*domain.Folder, error) {
			return &domain.Folder{ID: id, Name: "folder", Slug: "folder"}, nil
		},
		listFoldersFn: func(ctx context.Context, query port.ListQuery) (*port.FolderPageResult, error) {
			return &port.FolderPageResult{}, nil
		},
		listAllFoldersFn: func(ctx context.Context) ([]domain.Folder, error) {
			return []domain.Folder{}, nil
		},
		updateFolderFn: func(ctx context.Context, id string, update port.FolderUpdate) (*domain.Folder, error) {
			return &domain.Folder{ID: id, Name: "folder", Slug: "folder"}, nil
		},
		softDeleteFolderFn: func(ctx context.Context, id string) error {
			return nil
		},
		existsFolderFn: func(ctx context.Context, id string) (bool, error) { return true, nil },
	}

	if overrides.createFn != nil {
		mock.createFn = overrides.createFn
	}
	if overrides.getByIDFn != nil {
		mock.getByIDFn = overrides.getByIDFn
	}
	if overrides.listFn != nil {
		mock.listFn = overrides.listFn
	}
	if overrides.updateFn != nil {
		mock.updateFn = overrides.updateFn
	}
	if overrides.softDeleteFn != nil {
		mock.softDeleteFn = overrides.softDeleteFn
	}
	if overrides.createFolderFn != nil {
		mock.createFolderFn = overrides.createFolderFn
	}
	if overrides.getFolderByIDFn != nil {
		mock.getFolderByIDFn = overrides.getFolderByIDFn
	}
	if overrides.listFoldersFn != nil {
		mock.listFoldersFn = overrides.listFoldersFn
	}
	if overrides.listAllFoldersFn != nil {
		mock.listAllFoldersFn = overrides.listAllFoldersFn
	}
	if overrides.updateFolderFn != nil {
		mock.updateFolderFn = overrides.updateFolderFn
	}
	if overrides.softDeleteFolderFn != nil {
		mock.softDeleteFolderFn = overrides.softDeleteFolderFn
	}
	if overrides.existsFolderFn != nil {
		mock.existsFolderFn = overrides.existsFolderFn
	}

	return mock
}

// newStorageMock creates storage mocks with default behaviors.
func newStorageMock() storageMock {
	return storageMock{
		uploadFn: func(ctx context.Context, request port.UploadRequest) error { return nil },
		deleteFn: func(ctx context.Context, key string) error { return nil },
		existsFn: func(ctx context.Context, key string) (bool, error) { return true, nil },
	}
}

// ptr creates pointers for scalar values.
func ptr(value string) *string {
	return &value
}

// runConcurrent executes functions concurrently count times.
func runConcurrent(count int, fn func()) {
	var group sync.WaitGroup
	group.Add(count)

	for range count {
		go func() {
			defer group.Done()
			fn()
		}()
	}

	group.Wait()
}

// updateMax updates max values using compare-and-swap loops.
func updateMax(target *int32, candidate int32) {
	for {
		current := atomic.LoadInt32(target)
		if candidate <= current {
			return
		}
		if atomic.CompareAndSwapInt32(target, current, candidate) {
			return
		}
	}
}
