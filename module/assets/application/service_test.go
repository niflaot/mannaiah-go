package application

import (
	"context"
	errorspkg "errors"
	"strings"
	"testing"

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
	// updateNameFn defines update behavior.
	updateNameFn func(ctx context.Context, id string, name string) (*domain.Asset, error)
	// softDeleteFn defines soft-delete behavior.
	softDeleteFn func(ctx context.Context, id string) error
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

// UpdateName executes configured update behavior.
func (m repositoryMock) UpdateName(ctx context.Context, id string, name string) (*domain.Asset, error) {
	return m.updateNameFn(ctx, id, name)
}

// SoftDelete executes configured soft-delete behavior.
func (m repositoryMock) SoftDelete(ctx context.Context, id string) error {
	return m.softDeleteFn(ctx, id)
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
	storage := storageMock{
		uploadFn: func(ctx context.Context, request port.UploadRequest) error { return nil },
		deleteFn: func(ctx context.Context, key string) error { return nil },
		existsFn: func(ctx context.Context, key string) (bool, error) { return true, nil },
	}
	repository := repositoryMock{
		createFn: func(ctx context.Context, asset *domain.Asset) error { return nil },
		getByIDFn: func(ctx context.Context, id string) (*domain.Asset, error) {
			return &domain.Asset{ID: id}, nil
		},
		listFn: func(ctx context.Context, query port.ListQuery) (*port.PageResult, error) {
			return &port.PageResult{}, nil
		},
		updateNameFn: func(ctx context.Context, id string, name string) (*domain.Asset, error) {
			return &domain.Asset{ID: id, Name: name}, nil
		},
		softDeleteFn: func(ctx context.Context, id string) error { return nil },
	}

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

// TestCreate validates create command behavior and invariants.
func TestCreate(t *testing.T) {
	repository := repositoryMock{
		createFn: func(ctx context.Context, asset *domain.Asset) error { return nil },
		getByIDFn: func(ctx context.Context, id string) (*domain.Asset, error) {
			return &domain.Asset{ID: id}, nil
		},
		listFn: func(ctx context.Context, query port.ListQuery) (*port.PageResult, error) { return &port.PageResult{}, nil },
		updateNameFn: func(ctx context.Context, id string, name string) (*domain.Asset, error) {
			return &domain.Asset{ID: id, Name: name}, nil
		},
		softDeleteFn: func(ctx context.Context, id string) error { return nil },
	}

	service, err := NewService(repository, storageMock{
		uploadFn: func(ctx context.Context, request port.UploadRequest) error { return nil },
		deleteFn: func(ctx context.Context, key string) error { return nil },
		existsFn: func(ctx context.Context, key string) (bool, error) { return true, nil },
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if _, createErr := service.Create(context.Background(), CreateCommand{}); !errorspkg.Is(createErr, ErrFileRequired) {
		t.Fatalf("Create(empty) error = %v, want ErrFileRequired", createErr)
	}

	if _, createErr := service.Create(context.Background(), CreateCommand{Body: []byte("a"), Size: 11 * 1024 * 1024}); !errorspkg.Is(createErr, ErrFileTooLarge) {
		t.Fatalf("Create(large) error = %v, want ErrFileTooLarge", createErr)
	}

	if _, createErr := service.Create(context.Background(), CreateCommand{Body: []byte("a"), OriginalName: "file.png"}); !errorspkg.Is(createErr, domain.ErrMimeTypeRequired) {
		t.Fatalf("Create(missing mime) error = %v, want domain.ErrMimeTypeRequired", createErr)
	}
}

// TestCreateWithStorageUnavailable verifies availability failure behavior.
func TestCreateWithStorageUnavailable(t *testing.T) {
	service, err := NewService(repositoryMock{
		createFn: func(ctx context.Context, asset *domain.Asset) error { return nil },
		getByIDFn: func(ctx context.Context, id string) (*domain.Asset, error) {
			return &domain.Asset{ID: id}, nil
		},
		listFn:       func(ctx context.Context, query port.ListQuery) (*port.PageResult, error) { return &port.PageResult{}, nil },
		updateNameFn: func(ctx context.Context, id string, name string) (*domain.Asset, error) { return &domain.Asset{ID: id}, nil },
		softDeleteFn: func(ctx context.Context, id string) error { return nil },
	}, storageMock{
		uploadFn:       func(ctx context.Context, request port.UploadRequest) error { return nil },
		deleteFn:       func(ctx context.Context, key string) error { return nil },
		existsFn:       func(ctx context.Context, key string) (bool, error) { return true, nil },
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

// TestCreateSuccess verifies create flow with storage upload and event publication.
func TestCreateSuccess(t *testing.T) {
	var uploadedKey string
	var publishedTopic string

	service, err := NewService(repositoryMock{
		createFn: func(ctx context.Context, asset *domain.Asset) error {
			asset.CreatedAt = asset.CreatedAt.UTC()
			asset.UpdatedAt = asset.CreatedAt
			return nil
		},
		getByIDFn: func(ctx context.Context, id string) (*domain.Asset, error) {
			return &domain.Asset{ID: id}, nil
		},
		listFn:       func(ctx context.Context, query port.ListQuery) (*port.PageResult, error) { return &port.PageResult{}, nil },
		updateNameFn: func(ctx context.Context, id string, name string) (*domain.Asset, error) { return &domain.Asset{ID: id}, nil },
		softDeleteFn: func(ctx context.Context, id string) error { return nil },
	}, storageMock{
		uploadFn: func(ctx context.Context, request port.UploadRequest) error {
			uploadedKey = request.Key
			if request.ContentType != "image/png" {
				t.Fatalf("request.ContentType = %q, want %q", request.ContentType, "image/png")
			}
			return nil
		},
		deleteFn: func(ctx context.Context, key string) error { return nil },
		existsFn: func(ctx context.Context, key string) (bool, error) { return true, nil },
	}, publisherMock{publishFn: func(ctx context.Context, event port.IntegrationEvent) error {
		publishedTopic = event.Topic
		return nil
	}})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	entity, createErr := service.Create(context.Background(), CreateCommand{
		Name:         " Product Image ",
		OriginalName: " image one.png ",
		MimeType:     "image/png",
		Body:         []byte("payload"),
	})
	if createErr != nil {
		t.Fatalf("Create() error = %v", createErr)
	}
	if entity == nil {
		t.Fatalf("expected created entity")
	}
	if strings.TrimSpace(entity.ID) == "" {
		t.Fatalf("expected generated id")
	}
	if !strings.HasPrefix(uploadedKey, "assets/") {
		t.Fatalf("uploadedKey = %q, want prefix assets/", uploadedKey)
	}
	if publishedTopic != TopicAssetCreated {
		t.Fatalf("publishedTopic = %q, want %q", publishedTopic, TopicAssetCreated)
	}
}

// TestCreateRollback verifies rollback behavior when metadata persistence fails.
func TestCreateRollback(t *testing.T) {
	repositoryErr := errorspkg.New("db failed")
	rollbackCalled := false

	service, err := NewService(repositoryMock{
		createFn: func(ctx context.Context, asset *domain.Asset) error {
			return repositoryErr
		},
		getByIDFn:     func(ctx context.Context, id string) (*domain.Asset, error) { return nil, nil },
		listFn:        func(ctx context.Context, query port.ListQuery) (*port.PageResult, error) { return &port.PageResult{}, nil },
		updateNameFn:  func(ctx context.Context, id string, name string) (*domain.Asset, error) { return &domain.Asset{}, nil },
		softDeleteFn:  func(ctx context.Context, id string) error { return nil },
	}, storageMock{
		uploadFn: func(ctx context.Context, request port.UploadRequest) error { return nil },
		deleteFn: func(ctx context.Context, key string) error {
			rollbackCalled = true
			return nil
		},
		existsFn: func(ctx context.Context, key string) (bool, error) { return true, nil },
	})
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
}

// TestCreatePublishFailure verifies publication failure handling.
func TestCreatePublishFailure(t *testing.T) {
	publishErr := errorspkg.New("publish failed")

	service, err := NewService(repositoryMock{
		createFn:      func(ctx context.Context, asset *domain.Asset) error { return nil },
		getByIDFn:     func(ctx context.Context, id string) (*domain.Asset, error) { return nil, nil },
		listFn:        func(ctx context.Context, query port.ListQuery) (*port.PageResult, error) { return &port.PageResult{}, nil },
		updateNameFn:  func(ctx context.Context, id string, name string) (*domain.Asset, error) { return &domain.Asset{}, nil },
		softDeleteFn:  func(ctx context.Context, id string) error { return nil },
	}, storageMock{
		uploadFn: func(ctx context.Context, request port.UploadRequest) error { return nil },
		deleteFn: func(ctx context.Context, key string) error { return nil },
		existsFn: func(ctx context.Context, key string) (bool, error) { return true, nil },
	}, publisherMock{publishFn: func(ctx context.Context, event port.IntegrationEvent) error {
		return publishErr
	}})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, createErr := service.Create(context.Background(), CreateCommand{OriginalName: "a.png", MimeType: "image/png", Body: []byte("p")})
	if !errorspkg.Is(createErr, publishErr) {
		t.Fatalf("Create() error = %v, want publishErr", createErr)
	}
}

// TestGetListUpdateDeleteExists verifies non-create service operations.
func TestGetListUpdateDeleteExists(t *testing.T) {
	entity := &domain.Asset{ID: "a-1", Key: "assets/a-1.png", Name: "name", OriginalName: "a.png", MimeType: "image/png", Size: 10}

	service, err := NewService(repositoryMock{
		createFn: func(ctx context.Context, asset *domain.Asset) error { return nil },
		getByIDFn: func(ctx context.Context, id string) (*domain.Asset, error) {
			if id == "missing" {
				return nil, port.ErrNotFound
			}
			return entity, nil
		},
		listFn: func(ctx context.Context, query port.ListQuery) (*port.PageResult, error) {
			return &port.PageResult{Data: []domain.Asset{*entity}, Total: 1, Page: query.Page, Limit: query.Limit}, nil
		},
		updateNameFn: func(ctx context.Context, id string, name string) (*domain.Asset, error) {
			updated := *entity
			updated.Name = name
			return &updated, nil
		},
		softDeleteFn: func(ctx context.Context, id string) error { return nil },
	}, storageMock{
		uploadFn: func(ctx context.Context, request port.UploadRequest) error { return nil },
		deleteFn: func(ctx context.Context, key string) error { return nil },
		existsFn: func(ctx context.Context, key string) (bool, error) { return true, nil },
	}, publisherMock{publishFn: func(ctx context.Context, event port.IntegrationEvent) error { return nil }})
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

	if _, updateErr := service.UpdateName(context.Background(), "", "name"); !errorspkg.Is(updateErr, ErrInvalidID) {
		t.Fatalf("UpdateName(empty id) error = %v, want ErrInvalidID", updateErr)
	}
	if _, updateErr := service.UpdateName(context.Background(), "a-1", " "); !errorspkg.Is(updateErr, ErrInvalidName) {
		t.Fatalf("UpdateName(empty name) error = %v, want ErrInvalidName", updateErr)
	}
	if _, updateErr := service.UpdateName(context.Background(), "a-1", "new"); updateErr != nil {
		t.Fatalf("UpdateName() error = %v", updateErr)
	}

	if deleteErr := service.Delete(context.Background(), ""); !errorspkg.Is(deleteErr, ErrInvalidID) {
		t.Fatalf("Delete(empty) error = %v, want ErrInvalidID", deleteErr)
	}
	if deleteErr := service.Delete(context.Background(), "a-1"); deleteErr != nil {
		t.Fatalf("Delete() error = %v", deleteErr)
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

// TestBuildStorageKey verifies storage key normalization behavior.
func TestBuildStorageKey(t *testing.T) {
	key := buildStorageKey(" id-1 ", " ../my image.png ")
	if !strings.HasPrefix(key, "assets/id-1-") {
		t.Fatalf("key = %q, want prefix %q", key, "assets/id-1-")
	}
	if !strings.HasSuffix(key, "my-image.png") {
		t.Fatalf("key = %q, want suffix %q", key, "my-image.png")
	}

	fallback := buildStorageKey("id-2", "")
	if !strings.HasSuffix(fallback, "-file") {
		t.Fatalf("fallback = %q, want suffix %q", fallback, "-file")
	}
}
