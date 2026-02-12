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
	// updateNameFn defines update behavior.
	updateNameFn func(ctx context.Context, id string, name string) (*domain.Asset, error)
	// deleteFn defines delete behavior.
	deleteFn func(ctx context.Context, id string) error
	// existsFn defines exists behavior.
	existsFn func(ctx context.Context, id string) (bool, error)
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

// UpdateName executes configured update behavior.
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

// TestAssetEndpoints verifies endpoint behavior for CRUD paths.
func TestAssetEndpoints(t *testing.T) {
	handler := newHandlerForTest(t, serviceMock{
		createFn: func(ctx context.Context, command assetsapplication.CreateCommand) (*domain.Asset, error) {
			return &domain.Asset{ID: "a-1", Key: "assets/a-1.png", Name: "Asset", OriginalName: command.OriginalName, MimeType: command.MimeType, Size: command.Size}, nil
		},
		getFn: func(ctx context.Context, id string) (*domain.Asset, error) {
			return &domain.Asset{ID: id, Key: "assets/a-1.png", Name: "Asset", OriginalName: "one.png", MimeType: "image/png", Size: 10}, nil
		},
		listFn: func(ctx context.Context, query assetsapplication.ListQuery) (*port.PageResult, error) {
			return &port.PageResult{Data: []domain.Asset{{ID: "a-1"}}, Total: 1, Page: query.Page, Limit: query.Limit}, nil
		},
		updateNameFn: func(ctx context.Context, id string, name string) (*domain.Asset, error) {
			return &domain.Asset{ID: id, Name: name}, nil
		},
		deleteFn: func(ctx context.Context, id string) error { return nil },
		existsFn: func(ctx context.Context, id string) (bool, error) { return true, nil },
	})
	server := newHTTPServerForHandler(t, handler)

	createRequest, createContentType := newUploadRequestBody(t, "file", "image.png", []byte("payload"), "name", "Image")
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

	updateReq, _ := http.NewRequest(http.MethodPatch, "/assets/a-1", bytes.NewBufferString(`{"name":"updated"}`))
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
}

// TestHandlerErrorMapping verifies mapped error behavior.
func TestHandlerErrorMapping(t *testing.T) {
	handler := &Handler{}
	cases := []error{
		assetsapplication.ErrStorageUnavailable,
		assetsapplication.ErrInvalidID,
		assetsapplication.ErrInvalidName,
		assetsapplication.ErrFileRequired,
		assetsapplication.ErrFileTooLarge,
		domain.ErrKeyRequired,
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

	handler := newHandlerForTest(t, serviceMock{
		createFn: func(ctx context.Context, command assetsapplication.CreateCommand) (*domain.Asset, error) { return &domain.Asset{}, nil },
		getFn:    func(ctx context.Context, id string) (*domain.Asset, error) { return &domain.Asset{}, nil },
		listFn:   func(ctx context.Context, query assetsapplication.ListQuery) (*port.PageResult, error) { return &port.PageResult{}, nil },
		updateNameFn: func(ctx context.Context, id string, name string) (*domain.Asset, error) {
			return &domain.Asset{}, nil
		},
		deleteFn: func(ctx context.Context, id string) error { return nil },
		existsFn: func(ctx context.Context, id string) (bool, error) { return true, nil },
	}, authorizerMock{
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
			return ctx.SendStatus(200)
		})
	})

	request, _ := http.NewRequest(http.MethodGet, "/probe?page=x", nil)
	response := runRequest(t, server, request)
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusBadRequest)
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
func newUploadRequestBody(t *testing.T, fileField string, fileName string, content []byte, extraKey string, extraValue string) (*bytes.Buffer, string) {
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
	if extraKey != "" {
		if writeErr := writer.WriteField(extraKey, extraValue); writeErr != nil {
			t.Fatalf("WriteField() error = %v", writeErr)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}

	return body, writer.FormDataContentType()
}
