package http

import (
	"bytes"
	"context"
	errorspkg "errors"
	"mime/multipart"
	"net/http"
	"testing"

	assetsapplication "mannaiah/module/assets/application"
	"mannaiah/module/assets/domain"
	"mannaiah/module/assets/port"
	corehttp "mannaiah/module/core/http"
)

// serviceMock defines service behavior for asset handler tests.
type serviceMock struct {
	// createFn defines create behavior.
	createFn func(ctx context.Context, command assetsapplication.CreateCommand) (*domain.Asset, error)
	// getFn defines get behavior.
	getFn func(ctx context.Context, id string) (*domain.Asset, error)
	// listFn defines list behavior.
	listFn func(ctx context.Context, query assetsapplication.ListQuery) (*port.PageResult, error)
	// updateFn defines update behavior.
	updateFn func(ctx context.Context, id string, command assetsapplication.UpdateCommand) (*domain.Asset, error)
	// updateNameFn defines update-name behavior.
	updateNameFn func(ctx context.Context, id string, name string) (*domain.Asset, error)
	// deleteFn defines delete behavior.
	deleteFn func(ctx context.Context, id string) error
	// existsFn defines exists behavior.
	existsFn func(ctx context.Context, id string) (bool, error)
	// createFolderFn defines folder-create behavior.
	createFolderFn func(ctx context.Context, command assetsapplication.CreateFolderCommand) (*domain.Folder, error)
	// getFolderFn defines folder-get behavior.
	getFolderFn func(ctx context.Context, id string) (*domain.Folder, error)
	// listFoldersFn defines folder-list behavior.
	listFoldersFn func(ctx context.Context, query assetsapplication.ListQuery) (*port.FolderPageResult, error)
	// updateFolderFn defines folder-update behavior.
	updateFolderFn func(ctx context.Context, id string, command assetsapplication.UpdateFolderCommand) (*domain.Folder, error)
	// deleteFolderFn defines folder-delete behavior.
	deleteFolderFn func(ctx context.Context, id string) error
}

// Create executes configured create behavior.
func (m serviceMock) Create(ctx context.Context, command assetsapplication.CreateCommand) (*domain.Asset, error) {
	return m.createFn(ctx, command)
}

// Get executes configured get behavior.
func (m serviceMock) Get(ctx context.Context, id string) (*domain.Asset, error) {
	return m.getFn(ctx, id)
}

// List executes configured list behavior.
func (m serviceMock) List(ctx context.Context, query assetsapplication.ListQuery) (*port.PageResult, error) {
	return m.listFn(ctx, query)
}

// Update executes configured update behavior.
func (m serviceMock) Update(ctx context.Context, id string, command assetsapplication.UpdateCommand) (*domain.Asset, error) {
	return m.updateFn(ctx, id, command)
}

// UpdateName executes configured update-name behavior.
func (m serviceMock) UpdateName(ctx context.Context, id string, name string) (*domain.Asset, error) {
	return m.updateNameFn(ctx, id, name)
}

// Delete executes configured delete behavior.
func (m serviceMock) Delete(ctx context.Context, id string) error {
	return m.deleteFn(ctx, id)
}

// Exists executes configured exists behavior.
func (m serviceMock) Exists(ctx context.Context, id string) (bool, error) {
	return m.existsFn(ctx, id)
}

// CreateFolder executes configured folder-create behavior.
func (m serviceMock) CreateFolder(ctx context.Context, command assetsapplication.CreateFolderCommand) (*domain.Folder, error) {
	return m.createFolderFn(ctx, command)
}

// GetFolder executes configured folder-get behavior.
func (m serviceMock) GetFolder(ctx context.Context, id string) (*domain.Folder, error) {
	return m.getFolderFn(ctx, id)
}

// ListFolders executes configured folder-list behavior.
func (m serviceMock) ListFolders(ctx context.Context, query assetsapplication.ListQuery) (*port.FolderPageResult, error) {
	return m.listFoldersFn(ctx, query)
}

// UpdateFolder executes configured folder-update behavior.
func (m serviceMock) UpdateFolder(ctx context.Context, id string, command assetsapplication.UpdateFolderCommand) (*domain.Folder, error) {
	return m.updateFolderFn(ctx, id, command)
}

// DeleteFolder executes configured folder-delete behavior.
func (m serviceMock) DeleteFolder(ctx context.Context, id string) error {
	return m.deleteFolderFn(ctx, id)
}

// authorizerMock defines auth behavior for handler tests.
type authorizerMock struct {
	// requireFn defines auth behavior.
	requireFn func(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
	// isUnauthorizedFn defines unauthorized classification behavior.
	isUnauthorizedFn func(err error) bool
	// isForbiddenFn defines forbidden classification behavior.
	isForbiddenFn func(err error) bool
}

// Require executes configured auth behavior.
func (m authorizerMock) Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error {
	return m.requireFn(ctx, authorizationHeader, requiredPermissions...)
}

// IsUnauthorized executes configured unauthorized classification behavior.
func (m authorizerMock) IsUnauthorized(err error) bool {
	return m.isUnauthorizedFn(err)
}

// IsForbidden executes configured forbidden classification behavior.
func (m authorizerMock) IsForbidden(err error) bool {
	return m.isForbiddenFn(err)
}

// TestNewHandlerRejectsNilService verifies constructor validation behavior.
func TestNewHandlerRejectsNilService(t *testing.T) {
	if _, err := NewHandler(nil); !errorspkg.Is(err, ErrNilService) {
		t.Fatalf("NewHandler(nil) error = %v, want ErrNilService", err)
	}
}

// TestAssetEndpoints verifies endpoint behavior for asset and folder CRUD paths.
func TestAssetEndpoints(t *testing.T) {
	handler := newHandlerForTest(t, newServiceMock())
	server := newHTTPServerForHandler(t, handler)

	createRequest, createContentType := newUploadRequestBody(t, "file", "image.png", []byte("payload"), map[string]string{
		"name":     "Image",
		"tags":     `[{"name":"hero","color":"#ff0000"}]`,
		"metadata": `{"alt":"hero image"}`,
	})
	createReq, _ := http.NewRequest(http.MethodPost, "/assets", createRequest)
	createReq.Header.Set("Content-Type", createContentType)
	createResp := runRequest(t, server, createReq)
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("POST /assets status = %d, want %d", createResp.StatusCode, http.StatusCreated)
	}

	listReq, _ := http.NewRequest(http.MethodGet, "/assets?page=1&limit=10", nil)
	listResp := runRequest(t, server, listReq)
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /assets status = %d, want %d", listResp.StatusCode, http.StatusOK)
	}

	getReq, _ := http.NewRequest(http.MethodGet, "/assets/a-1", nil)
	getResp := runRequest(t, server, getReq)
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /assets/:id status = %d, want %d", getResp.StatusCode, http.StatusOK)
	}

	updateReq, _ := http.NewRequest(http.MethodPatch, "/assets/a-1", bytes.NewBufferString(`{"name":"updated","folderId":"f-1","tags":[{"name":"hero","color":"#ff0000"}],"metadata":{"alt":"updated"}}`))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp := runRequest(t, server, updateReq)
	if updateResp.StatusCode != http.StatusOK {
		t.Fatalf("PATCH /assets/:id status = %d, want %d", updateResp.StatusCode, http.StatusOK)
	}

	deleteReq, _ := http.NewRequest(http.MethodDelete, "/assets/a-1", nil)
	deleteResp := runRequest(t, server, deleteReq)
	if deleteResp.StatusCode != http.StatusOK {
		t.Fatalf("DELETE /assets/:id status = %d, want %d", deleteResp.StatusCode, http.StatusOK)
	}

	createFolderReq, _ := http.NewRequest(http.MethodPost, "/assets/folders", bytes.NewBufferString(`{"name":"Hero","parentFolderId":"root","tags":[{"name":"hero","color":"#ff0000"}]}`))
	createFolderReq.Header.Set("Content-Type", "application/json")
	createFolderResp := runRequest(t, server, createFolderReq)
	if createFolderResp.StatusCode != http.StatusCreated {
		t.Fatalf("POST /assets/folders status = %d, want %d", createFolderResp.StatusCode, http.StatusCreated)
	}

	listFoldersReq, _ := http.NewRequest(http.MethodGet, "/assets/folders?page=1&limit=10&parentFolderId=root", nil)
	listFoldersResp := runRequest(t, server, listFoldersReq)
	if listFoldersResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /assets/folders status = %d, want %d", listFoldersResp.StatusCode, http.StatusOK)
	}

	getFolderReq, _ := http.NewRequest(http.MethodGet, "/assets/folders/f-1", nil)
	getFolderResp := runRequest(t, server, getFolderReq)
	if getFolderResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /assets/folders/:id status = %d, want %d", getFolderResp.StatusCode, http.StatusOK)
	}

	updateFolderReq, _ := http.NewRequest(http.MethodPatch, "/assets/folders/f-1", bytes.NewBufferString(`{"name":"Catalog","parentFolderId":"root"}`))
	updateFolderReq.Header.Set("Content-Type", "application/json")
	updateFolderResp := runRequest(t, server, updateFolderReq)
	if updateFolderResp.StatusCode != http.StatusOK {
		t.Fatalf("PATCH /assets/folders/:id status = %d, want %d", updateFolderResp.StatusCode, http.StatusOK)
	}

	deleteFolderReq, _ := http.NewRequest(http.MethodDelete, "/assets/folders/f-1", nil)
	deleteFolderResp := runRequest(t, server, deleteFolderReq)
	if deleteFolderResp.StatusCode != http.StatusOK {
		t.Fatalf("DELETE /assets/folders/:id status = %d, want %d", deleteFolderResp.StatusCode, http.StatusOK)
	}
}

// TestHandlerErrorMapping verifies mapped error behavior.
func TestHandlerErrorMapping(t *testing.T) {
	handler := &Handler{}
	cases := []error{
		assetsapplication.ErrStorageUnavailable,
		assetsapplication.ErrInvalidID,
		assetsapplication.ErrInvalidName,
		assetsapplication.ErrInvalidFolderID,
		assetsapplication.ErrInvalidFolderName,
		assetsapplication.ErrInvalidFolderParent,
		assetsapplication.ErrFileRequired,
		assetsapplication.ErrFileTooLarge,
		domain.ErrKeyRequired,
		domain.ErrFolderNameRequired,
		domain.ErrFolderParentSelfReference,
		domain.ErrFolderParentCycle,
		domain.ErrInvalidMetadata,
		port.ErrFolderNotFound,
		port.ErrNotFound,
		errorspkg.New("boom"),
	}

	for _, value := range cases {
		if mapped := handler.mapError(value); mapped == nil {
			t.Fatalf("expected mapped error for %v", value)
		}
	}
}

// TestHandlerAuthEnforcement verifies route-level auth behavior.
func TestHandlerAuthEnforcement(t *testing.T) {
	unauthorizedErr := errorspkg.New("unauthorized")
	forbiddenErr := errorspkg.New("forbidden")

	handler := newHandlerForTest(t, newServiceMock(), authorizerMock{
		requireFn: func(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error {
			if authorizationHeader == "Bearer unauthorized" {
				return unauthorizedErr
			}
			if authorizationHeader == "Bearer forbidden" {
				return forbiddenErr
			}
			return nil
		},
		isUnauthorizedFn: func(err error) bool { return errorspkg.Is(err, unauthorizedErr) },
		isForbiddenFn:    func(err error) bool { return errorspkg.Is(err, forbiddenErr) },
	})

	server := newHTTPServerForHandler(t, handler)

	unauthorizedReq, _ := http.NewRequest(http.MethodGet, "/assets", nil)
	unauthorizedReq.Header.Set("Authorization", "Bearer unauthorized")
	if response := runRequest(t, server, unauthorizedReq); response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d, want %d", response.StatusCode, http.StatusUnauthorized)
	}

	forbiddenReq, _ := http.NewRequest(http.MethodGet, "/assets", nil)
	forbiddenReq.Header.Set("Authorization", "Bearer forbidden")
	if response := runRequest(t, server, forbiddenReq); response.StatusCode != http.StatusForbidden {
		t.Fatalf("forbidden status = %d, want %d", response.StatusCode, http.StatusForbidden)
	}
}

// TestParseIntQuery verifies integer query parsing behavior.
func TestParseIntQuery(t *testing.T) {
	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8151}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}

	server.RegisterRoutes(func(router corehttp.Router) {
		router.Get("/probe", func(ctx corehttp.Context) error {
			_, parseErr := parseIntQuery(ctx, "page", 1)
			if parseErr != nil {
				return corehttp.NewAppError(400, "bad", parseErr)
			}
			_, jsonErr := parseJSONField[[]domain.Tag]("[bad json")
			if jsonErr != nil {
				return corehttp.NewAppError(400, "bad_json", jsonErr)
			}
			return ctx.SendStatus(200)
		})
	})

	request, _ := http.NewRequest(http.MethodGet, "/probe?page=x", nil)
	response := runRequest(t, server, request)
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusBadRequest)
	}
}

// TestFolderListAndCreateErrorPaths verifies folder list and create failure mappings.
func TestFolderListAndCreateErrorPaths(t *testing.T) {
	handler := newHandlerForTest(t, serviceMock{
		createFn:       newServiceMock().createFn,
		getFn:          newServiceMock().getFn,
		listFn:         newServiceMock().listFn,
		updateFn:       newServiceMock().updateFn,
		updateNameFn:   newServiceMock().updateNameFn,
		deleteFn:       newServiceMock().deleteFn,
		existsFn:       newServiceMock().existsFn,
		getFolderFn:    newServiceMock().getFolderFn,
		updateFolderFn: newServiceMock().updateFolderFn,
		deleteFolderFn: newServiceMock().deleteFolderFn,
		createFolderFn: func(ctx context.Context, command assetsapplication.CreateFolderCommand) (*domain.Folder, error) {
			return nil, domain.ErrFolderNameRequired
		},
		listFoldersFn: func(ctx context.Context, query assetsapplication.ListQuery) (*port.FolderPageResult, error) {
			return &port.FolderPageResult{}, nil
		},
	})
	handler.SetAuthorizer(nil)
	server := newHTTPServerForHandler(t, handler)

	request, _ := http.NewRequest(http.MethodGet, "/assets/folders?page=bad", nil)
	response := runRequest(t, server, request)
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("GET /assets/folders?page=bad status = %d, want %d", response.StatusCode, http.StatusBadRequest)
	}

	createBadBody, _ := http.NewRequest(http.MethodPost, "/assets/folders", bytes.NewBufferString(`{"name":" "}`))
	createBadBody.Header.Set("Content-Type", "application/json")
	createBadBodyResponse := runRequest(t, server, createBadBody)
	if createBadBodyResponse.StatusCode != http.StatusBadRequest {
		t.Fatalf("POST /assets/folders invalid status = %d, want %d", createBadBodyResponse.StatusCode, http.StatusBadRequest)
	}
}

// TestAssetCreateAndUpdateErrorPaths verifies invalid multipart/json request handling.
func TestAssetCreateAndUpdateErrorPaths(t *testing.T) {
	handler := newHandlerForTest(t, newServiceMock())
	server := newHTTPServerForHandler(t, handler)

	noFileRequest, _ := http.NewRequest(http.MethodPost, "/assets", nil)
	noFileResponse := runRequest(t, server, noFileRequest)
	if noFileResponse.StatusCode != http.StatusBadRequest {
		t.Fatalf("POST /assets without file status = %d, want %d", noFileResponse.StatusCode, http.StatusBadRequest)
	}

	invalidTagsBody, invalidTagsContentType := newUploadRequestBody(t, "file", "image.png", []byte("payload"), map[string]string{
		"tags": "[bad json",
	})
	invalidTagsRequest, _ := http.NewRequest(http.MethodPost, "/assets", invalidTagsBody)
	invalidTagsRequest.Header.Set("Content-Type", invalidTagsContentType)
	invalidTagsResponse := runRequest(t, server, invalidTagsRequest)
	if invalidTagsResponse.StatusCode != http.StatusBadRequest {
		t.Fatalf("POST /assets invalid tags status = %d, want %d", invalidTagsResponse.StatusCode, http.StatusBadRequest)
	}

	invalidUpdateRequest, _ := http.NewRequest(http.MethodPatch, "/assets/a-1", bytes.NewBufferString(`{`))
	invalidUpdateRequest.Header.Set("Content-Type", "application/json")
	invalidUpdateResponse := runRequest(t, server, invalidUpdateRequest)
	if invalidUpdateResponse.StatusCode != http.StatusBadRequest {
		t.Fatalf("PATCH /assets/:id invalid payload status = %d, want %d", invalidUpdateResponse.StatusCode, http.StatusBadRequest)
	}

	if values := dereferenceTags(nil); values != nil {
		t.Fatalf("dereferenceTags(nil) = %#v, want nil", values)
	}
	if values := dereferenceMetadata(nil); values != nil {
		t.Fatalf("dereferenceMetadata(nil) = %#v, want nil", values)
	}
}

// newServiceMock creates default service mocks for handler tests.
func newServiceMock() serviceMock {
	return serviceMock{
		createFn: func(ctx context.Context, command assetsapplication.CreateCommand) (*domain.Asset, error) {
			return &domain.Asset{ID: "a-1", Key: "assets/a-1.png", Name: "Asset", OriginalName: command.OriginalName, MimeType: command.MimeType, Size: command.Size}, nil
		},
		getFn: func(ctx context.Context, id string) (*domain.Asset, error) {
			return &domain.Asset{ID: id, Key: "assets/a-1.png", Name: "Asset", OriginalName: "one.png", MimeType: "image/png", Size: 10}, nil
		},
		listFn: func(ctx context.Context, query assetsapplication.ListQuery) (*port.PageResult, error) {
			return &port.PageResult{Data: []domain.Asset{{ID: "a-1"}}, Total: 1, Page: query.Page, Limit: query.Limit}, nil
		},
		updateFn: func(ctx context.Context, id string, command assetsapplication.UpdateCommand) (*domain.Asset, error) {
			name := "updated"
			if command.Name != nil {
				name = *command.Name
			}
			return &domain.Asset{ID: id, Name: name}, nil
		},
		updateNameFn: func(ctx context.Context, id string, name string) (*domain.Asset, error) {
			return &domain.Asset{ID: id, Name: name}, nil
		},
		deleteFn: func(ctx context.Context, id string) error { return nil },
		existsFn: func(ctx context.Context, id string) (bool, error) { return true, nil },
		createFolderFn: func(ctx context.Context, command assetsapplication.CreateFolderCommand) (*domain.Folder, error) {
			return &domain.Folder{ID: "f-1", Name: command.Name, Slug: "hero", ParentFolderID: command.ParentFolderID}, nil
		},
		getFolderFn: func(ctx context.Context, id string) (*domain.Folder, error) {
			return &domain.Folder{ID: id, Name: "Hero", Slug: "hero"}, nil
		},
		listFoldersFn: func(ctx context.Context, query assetsapplication.ListQuery) (*port.FolderPageResult, error) {
			return &port.FolderPageResult{Data: []domain.Folder{{ID: "f-1", Name: "Hero", Slug: "hero"}}, Total: 1, Page: query.Page, Limit: query.Limit}, nil
		},
		updateFolderFn: func(ctx context.Context, id string, command assetsapplication.UpdateFolderCommand) (*domain.Folder, error) {
			parentID := ""
			if command.ParentFolderID != nil {
				parentID = *command.ParentFolderID
			}
			return &domain.Folder{ID: id, Name: "Catalog", Slug: "catalog", ParentFolderID: parentID}, nil
		},
		deleteFolderFn: func(ctx context.Context, id string) error { return nil },
	}
}

// newHandlerForTest creates handlers for tests.
func newHandlerForTest(t *testing.T, service assetsapplication.Service, authorizers ...Authorizer) *Handler {
	t.Helper()

	handler, err := NewHandler(service, authorizers...)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	return handler
}

// newHTTPServerForHandler creates servers for handler tests.
func newHTTPServerForHandler(t *testing.T, handler *Handler) *corehttp.Server {
	t.Helper()

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8150}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(handler.RegisterRoutes)

	return server
}

// runRequest runs HTTP requests against test servers.
func runRequest(t *testing.T, server *corehttp.Server, request *http.Request) *http.Response {
	t.Helper()

	response, err := server.App().Test(request)
	if err != nil {
		t.Fatalf("App().Test() error = %v", err)
	}

	return response
}

// newUploadRequestBody builds a multipart upload request body and content type.
func newUploadRequestBody(t *testing.T, fileField string, fileName string, content []byte, fields map[string]string) (*bytes.Buffer, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile(fileField, fileName)
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("part.Write() error = %v", err)
	}
	for key, value := range fields {
		if writeErr := writer.WriteField(key, value); writeErr != nil {
			t.Fatalf("WriteField() error = %v", writeErr)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}

	return body, writer.FormDataContentType()
}
