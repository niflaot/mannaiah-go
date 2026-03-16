package category_test

import (
	"bytes"
	"context"
	errorspkg "errors"
	stdhttp "net/http"
	"testing"

	corehttp "mannaiah/module/core/http"
	categoryapplication "mannaiah/module/products/application/category"
	categorydomain "mannaiah/module/products/domain/category"
	categoryport "mannaiah/module/products/port/category"

	categoryhttp "mannaiah/module/products/adapter/http/category"
)

// categoryServiceMock defines category service behavior for HTTP handler tests.
type categoryServiceMock struct {
	// createFn defines create behavior.
	createFn func(ctx context.Context, command categoryapplication.CreateCommand) (*categorydomain.Category, error)
	// getFn defines get behavior.
	getFn func(ctx context.Context, id string) (*categorydomain.Category, error)
	// getBySlugFn defines get-by-slug behavior.
	getBySlugFn func(ctx context.Context, slug string) (*categorydomain.Category, error)
	// treeFn defines tree listing behavior.
	treeFn func(ctx context.Context) ([]*categorydomain.Category, error)
	// childrenFn defines children listing behavior.
	childrenFn func(ctx context.Context, parentID string) ([]*categorydomain.Category, error)
	// updateFn defines update behavior.
	updateFn func(ctx context.Context, id string, command categoryapplication.UpdateCommand) (*categorydomain.Category, error)
	// deleteFn defines delete behavior.
	deleteFn func(ctx context.Context, id string) error
	// listProductsFn defines product listing behavior.
	listProductsFn func(ctx context.Context, q categoryapplication.ListProductsQuery) (*categoryport.ListProductsResult, error)
}

// Create executes configured create behavior.
func (m *categoryServiceMock) Create(ctx context.Context, command categoryapplication.CreateCommand) (*categorydomain.Category, error) {
	if m.createFn != nil {
		return m.createFn(ctx, command)
	}

	return &categorydomain.Category{ID: "cat-1", Slug: command.Slug, Name: command.Name}, nil
}

// Get executes configured get behavior.
func (m *categoryServiceMock) Get(ctx context.Context, id string) (*categorydomain.Category, error) {
	if m.getFn != nil {
		return m.getFn(ctx, id)
	}

	return &categorydomain.Category{ID: id, Slug: "test", Name: "Test"}, nil
}

// GetBySlug executes configured get-by-slug behavior.
func (m *categoryServiceMock) GetBySlug(ctx context.Context, slug string) (*categorydomain.Category, error) {
	if m.getBySlugFn != nil {
		return m.getBySlugFn(ctx, slug)
	}

	return &categorydomain.Category{ID: "cat-1", Slug: slug, Name: "Test"}, nil
}

// Tree executes configured tree listing behavior.
func (m *categoryServiceMock) Tree(ctx context.Context) ([]*categorydomain.Category, error) {
	if m.treeFn != nil {
		return m.treeFn(ctx)
	}

	return nil, nil
}

// Children executes configured children listing behavior.
func (m *categoryServiceMock) Children(ctx context.Context, parentID string) ([]*categorydomain.Category, error) {
	if m.childrenFn != nil {
		return m.childrenFn(ctx, parentID)
	}

	return nil, nil
}

// Update executes configured update behavior.
func (m *categoryServiceMock) Update(ctx context.Context, id string, command categoryapplication.UpdateCommand) (*categorydomain.Category, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, command)
	}

	return &categorydomain.Category{ID: id, Slug: "updated", Name: "Updated"}, nil
}

// Delete executes configured delete behavior.
func (m *categoryServiceMock) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}

	return nil
}

// ListProducts executes configured product listing behavior.
func (m *categoryServiceMock) ListProducts(ctx context.Context, q categoryapplication.ListProductsQuery) (*categoryport.ListProductsResult, error) {
	if m.listProductsFn != nil {
		return m.listProductsFn(ctx, q)
	}

	return &categoryport.ListProductsResult{}, nil
}

// categoryAuthorizerMock defines auth behavior for handler tests.
type categoryAuthorizerMock struct {
	// requireFn defines auth and permission behavior.
	requireFn func(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
	// isUnauthorizedFn defines unauthorized classification behavior.
	isUnauthorizedFn func(err error) bool
	// isForbiddenFn defines forbidden classification behavior.
	isForbiddenFn func(err error) bool
}

// Require executes configured auth behavior.
func (m *categoryAuthorizerMock) Require(ctx context.Context, header string, permissions ...string) error {
	if m.requireFn != nil {
		return m.requireFn(ctx, header, permissions...)
	}

	return nil
}

// IsUnauthorized executes configured unauthorized classification behavior.
func (m *categoryAuthorizerMock) IsUnauthorized(err error) bool {
	if m.isUnauthorizedFn != nil {
		return m.isUnauthorizedFn(err)
	}

	return false
}

// IsForbidden executes configured forbidden classification behavior.
func (m *categoryAuthorizerMock) IsForbidden(err error) bool {
	if m.isForbiddenFn != nil {
		return m.isForbiddenFn(err)
	}

	return false
}

// TestNewHandler_NilService verifies ErrNilService is returned.
func TestNewHandler_NilService(t *testing.T) {
	_, err := categoryhttp.NewHandler(nil)
	if !errorspkg.Is(err, categoryhttp.ErrNilService) {
		t.Fatalf("NewHandler(nil) error = %v, want ErrNilService", err)
	}
}

// TestCategoryEndpoints verifies CRUD endpoint status codes.
func TestCategoryEndpoints(t *testing.T) {
	svc := &categoryServiceMock{}
	handler, err := categoryhttp.NewHandler(svc)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := newServerForHandler(t, handler, 8161)

	createReq, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/categories", bytes.NewBufferString(`{"slug":"electronics","name":"Electronics"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := server.App().Test(createReq)
	if err != nil {
		t.Fatalf("Test(create) error = %v", err)
	}
	if createResp.StatusCode != stdhttp.StatusCreated {
		t.Fatalf("POST /categories status = %d, want %d", createResp.StatusCode, stdhttp.StatusCreated)
	}

	treeReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/categories", nil)
	treeResp, err := server.App().Test(treeReq)
	if err != nil {
		t.Fatalf("Test(tree) error = %v", err)
	}
	if treeResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("GET /categories status = %d, want %d", treeResp.StatusCode, stdhttp.StatusOK)
	}

	getReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/categories/cat-1", nil)
	getResp, err := server.App().Test(getReq)
	if err != nil {
		t.Fatalf("Test(get) error = %v", err)
	}
	if getResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("GET /categories/:id status = %d, want %d", getResp.StatusCode, stdhttp.StatusOK)
	}

	childrenReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/categories/cat-1/children", nil)
	childrenResp, err := server.App().Test(childrenReq)
	if err != nil {
		t.Fatalf("Test(children) error = %v", err)
	}
	if childrenResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("GET /categories/:id/children status = %d, want %d", childrenResp.StatusCode, stdhttp.StatusOK)
	}

	productsReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/categories/cat-1/products", nil)
	productsResp, err := server.App().Test(productsReq)
	if err != nil {
		t.Fatalf("Test(products) error = %v", err)
	}
	if productsResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("GET /categories/:id/products status = %d, want %d", productsResp.StatusCode, stdhttp.StatusOK)
	}

	updateReq, _ := stdhttp.NewRequest(stdhttp.MethodPatch, "/categories/cat-1", bytes.NewBufferString(`{"name":"Updated"}`))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp, err := server.App().Test(updateReq)
	if err != nil {
		t.Fatalf("Test(update) error = %v", err)
	}
	if updateResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("PATCH /categories/:id status = %d, want %d", updateResp.StatusCode, stdhttp.StatusOK)
	}

	deleteReq, _ := stdhttp.NewRequest(stdhttp.MethodDelete, "/categories/cat-1", nil)
	deleteResp, err := server.App().Test(deleteReq)
	if err != nil {
		t.Fatalf("Test(delete) error = %v", err)
	}
	if deleteResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("DELETE /categories/:id status = %d, want %d", deleteResp.StatusCode, stdhttp.StatusOK)
	}
}

// TestCategoryHandler_NotFound verifies 404 error mapping.
func TestCategoryHandler_NotFound(t *testing.T) {
	svc := &categoryServiceMock{
		getFn: func(_ context.Context, _ string) (*categorydomain.Category, error) {
			return nil, categoryapplication.ErrNotFound
		},
	}
	handler, _ := categoryhttp.NewHandler(svc)
	server := newServerForHandler(t, handler, 8162)

	req, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/categories/missing", nil)
	resp, err := server.App().Test(req)
	if err != nil {
		t.Fatalf("Test() error = %v", err)
	}
	if resp.StatusCode != stdhttp.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, stdhttp.StatusNotFound)
	}
}

// TestCategoryHandler_SlugConflict verifies 409 for duplicate slug.
func TestCategoryHandler_SlugConflict(t *testing.T) {
	svc := &categoryServiceMock{
		createFn: func(_ context.Context, _ categoryapplication.CreateCommand) (*categorydomain.Category, error) {
			return nil, categoryapplication.ErrDuplicateSlug
		},
	}
	handler, _ := categoryhttp.NewHandler(svc)
	server := newServerForHandler(t, handler, 8163)

	req, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/categories", bytes.NewBufferString(`{"slug":"dup","name":"Dup"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := server.App().Test(req)
	if err != nil {
		t.Fatalf("Test() error = %v", err)
	}
	if resp.StatusCode != stdhttp.StatusConflict {
		t.Fatalf("status = %d, want %d", resp.StatusCode, stdhttp.StatusConflict)
	}
}

// TestCategoryHandler_Auth verifies auth enforcement on protected endpoints.
func TestCategoryHandler_Auth(t *testing.T) {
	unauthorizedErr := errorspkg.New("unauthorized")
	forbiddenErr := errorspkg.New("forbidden")

	svc := &categoryServiceMock{}
	authorizer := &categoryAuthorizerMock{
		requireFn: func(_ context.Context, header string, _ ...string) error {
			if header == "Bearer unauth" {
				return unauthorizedErr
			}
			if header == "Bearer forbidden" {
				return forbiddenErr
			}

			return nil
		},
		isUnauthorizedFn: func(err error) bool { return errorspkg.Is(err, unauthorizedErr) },
		isForbiddenFn:    func(err error) bool { return errorspkg.Is(err, forbiddenErr) },
	}

	handler, _ := categoryhttp.NewHandler(svc, authorizer)
	server := newServerForHandler(t, handler, 8164)

	unauthReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/categories", nil)
	unauthReq.Header.Set("Authorization", "Bearer unauth")
	unauthResp, err := server.App().Test(unauthReq)
	if err != nil {
		t.Fatalf("Test(unauth) error = %v", err)
	}
	if unauthResp.StatusCode != stdhttp.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", unauthResp.StatusCode, stdhttp.StatusUnauthorized)
	}

	forbiddenReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/categories", nil)
	forbiddenReq.Header.Set("Authorization", "Bearer forbidden")
	forbiddenResp, err := server.App().Test(forbiddenReq)
	if err != nil {
		t.Fatalf("Test(forbidden) error = %v", err)
	}
	if forbiddenResp.StatusCode != stdhttp.StatusForbidden {
		t.Fatalf("status = %d, want %d", forbiddenResp.StatusCode, stdhttp.StatusForbidden)
	}
}

// TestCategoryHandler_InvalidPayload verifies bad request on malformed JSON.
func TestCategoryHandler_InvalidPayload(t *testing.T) {
	svc := &categoryServiceMock{}
	handler, _ := categoryhttp.NewHandler(svc)
	server := newServerForHandler(t, handler, 8165)

	req, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/categories", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := server.App().Test(req)
	if err != nil {
		t.Fatalf("Test() error = %v", err)
	}
	if resp.StatusCode != stdhttp.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, stdhttp.StatusBadRequest)
	}
}

// TestCategoryHandler_MapErrorVariants verifies all error mappings.
func TestCategoryHandler_MapErrorVariants(t *testing.T) {
	svc := &categoryServiceMock{
		getFn: func(_ context.Context, _ string) (*categorydomain.Category, error) {
			return nil, errorspkg.New("boom")
		},
	}
	handler, _ := categoryhttp.NewHandler(svc)
	server := newServerForHandler(t, handler, 8166)

	req, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/categories/any", nil)
	resp, err := server.App().Test(req)
	if err != nil {
		t.Fatalf("Test() error = %v", err)
	}
	if resp.StatusCode != stdhttp.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", resp.StatusCode, stdhttp.StatusInternalServerError)
	}
}

// newServerForHandler creates HTTP servers for category handler tests.
func newServerForHandler(t *testing.T, handler *categoryhttp.Handler, port int) *corehttp.Server {
	t.Helper()

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: port}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(handler.RegisterRoutes)

	return server
}
