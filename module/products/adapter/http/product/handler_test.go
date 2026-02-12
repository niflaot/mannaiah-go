package product

import (
	"bytes"
	"context"
	errorspkg "errors"
	stdhttp "net/http"
	"testing"

	corehttp "mannaiah/module/core/http"
	productapplication "mannaiah/module/products/application/product"
	productdomain "mannaiah/module/products/domain/product"
	productport "mannaiah/module/products/port/product"
)

// serviceMock defines product service behavior for HTTP handler tests.
type serviceMock struct {
	// createFn defines create behavior.
	createFn func(ctx context.Context, command productapplication.CreateCommand) (*productdomain.Product, error)
	// getFn defines get behavior.
	getFn func(ctx context.Context, id string) (*productdomain.Product, error)
	// listFn defines list behavior.
	listFn func(ctx context.Context) ([]productdomain.Product, error)
	// updateFn defines update behavior.
	updateFn func(ctx context.Context, id string, command productapplication.UpdateCommand) (*productdomain.Product, error)
	// deleteFn defines delete behavior.
	deleteFn func(ctx context.Context, id string) error
}

// Create executes configured create behavior.
func (m serviceMock) Create(ctx context.Context, command productapplication.CreateCommand) (*productdomain.Product, error) {
	return m.createFn(ctx, command)
}

// Get executes configured get behavior.
func (m serviceMock) Get(ctx context.Context, id string) (*productdomain.Product, error) {
	return m.getFn(ctx, id)
}

// List executes configured list behavior.
func (m serviceMock) List(ctx context.Context) ([]productdomain.Product, error) {
	return m.listFn(ctx)
}

// Update executes configured update behavior.
func (m serviceMock) Update(ctx context.Context, id string, command productapplication.UpdateCommand) (*productdomain.Product, error) {
	return m.updateFn(ctx, id, command)
}

// Delete executes configured delete behavior.
func (m serviceMock) Delete(ctx context.Context, id string) error {
	return m.deleteFn(ctx, id)
}

// authorizerMock defines auth behavior for handler tests.
type authorizerMock struct {
	// requireFn defines auth and permission behavior.
	requireFn func(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
	// isUnauthorizedFn defines unauthorized classification behavior.
	isUnauthorizedFn func(err error) bool
	// isForbiddenFn defines forbidden classification behavior.
	isForbiddenFn func(err error) bool
}

// Require executes configured auth and permission behavior.
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
		t.Fatalf("NewHandler() error = %v, want ErrNilService", err)
	}
}

// TestProductEndpoints verifies CRUD endpoint behavior.
func TestProductEndpoints(t *testing.T) {
	handler := newHandlerForTest(t, serviceMock{
		createFn: func(ctx context.Context, command productapplication.CreateCommand) (*productdomain.Product, error) {
			return &productdomain.Product{ID: "p-1", SKU: command.SKU}, nil
		},
		getFn: func(ctx context.Context, id string) (*productdomain.Product, error) {
			return &productdomain.Product{ID: id, SKU: "SKU-1"}, nil
		},
		listFn: func(ctx context.Context) ([]productdomain.Product, error) {
			return []productdomain.Product{{ID: "p-1", SKU: "SKU-1"}}, nil
		},
		updateFn: func(ctx context.Context, id string, command productapplication.UpdateCommand) (*productdomain.Product, error) {
			return &productdomain.Product{ID: id, SKU: "SKU-2"}, nil
		},
		deleteFn: func(ctx context.Context, id string) error { return nil },
	})
	server := newHTTPServerForHandler(t, handler)

	createReq, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/products", bytes.NewBufferString(`{"sku":"SKU-1"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createResp := runRequest(t, server, createReq)
	if createResp.StatusCode != stdhttp.StatusCreated {
		t.Fatalf("POST status = %d, want %d", createResp.StatusCode, stdhttp.StatusCreated)
	}

	listReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/products", nil)
	listResp := runRequest(t, server, listReq)
	if listResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("GET /products status = %d, want %d", listResp.StatusCode, stdhttp.StatusOK)
	}

	getReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/products/p-1", nil)
	getResp := runRequest(t, server, getReq)
	if getResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("GET /products/:id status = %d, want %d", getResp.StatusCode, stdhttp.StatusOK)
	}

	updateReq, _ := stdhttp.NewRequest(stdhttp.MethodPatch, "/products/p-1", bytes.NewBufferString(`{"sku":"SKU-2"}`))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp := runRequest(t, server, updateReq)
	if updateResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("PATCH status = %d, want %d", updateResp.StatusCode, stdhttp.StatusOK)
	}

	deleteReq, _ := stdhttp.NewRequest(stdhttp.MethodDelete, "/products/p-1", nil)
	deleteResp := runRequest(t, server, deleteReq)
	if deleteResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("DELETE status = %d, want %d", deleteResp.StatusCode, stdhttp.StatusOK)
	}
}

// TestHandlerInvalidPayload verifies invalid payload behavior.
func TestHandlerInvalidPayload(t *testing.T) {
	handler := newHandlerForTest(t, serviceMock{
		createFn: func(ctx context.Context, command productapplication.CreateCommand) (*productdomain.Product, error) {
			return &productdomain.Product{}, nil
		},
		getFn: func(ctx context.Context, id string) (*productdomain.Product, error) {
			return &productdomain.Product{}, nil
		},
		listFn: func(ctx context.Context) ([]productdomain.Product, error) { return []productdomain.Product{}, nil },
		updateFn: func(ctx context.Context, id string, command productapplication.UpdateCommand) (*productdomain.Product, error) {
			return &productdomain.Product{}, nil
		},
		deleteFn: func(ctx context.Context, id string) error { return nil },
	})
	server := newHTTPServerForHandler(t, handler)

	req, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/products", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	resp := runRequest(t, server, req)
	if resp.StatusCode != stdhttp.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, stdhttp.StatusBadRequest)
	}
}

// TestHandlerMapErrorVariants verifies mapped error behavior.
func TestHandlerMapErrorVariants(t *testing.T) {
	handler := &Handler{}
	cases := []error{
		productport.ErrNotFound,
		productapplication.ErrInvalidID,
		productdomain.ErrSKURequired,
		productport.ErrDuplicateSKU,
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
	unauthorizedError := errorspkg.New("unauthorized")
	forbiddenError := errorspkg.New("forbidden")

	handler := newHandlerForTest(t, serviceMock{
		createFn: func(ctx context.Context, command productapplication.CreateCommand) (*productdomain.Product, error) {
			return &productdomain.Product{}, nil
		},
		getFn: func(ctx context.Context, id string) (*productdomain.Product, error) {
			return &productdomain.Product{}, nil
		},
		listFn: func(ctx context.Context) ([]productdomain.Product, error) { return []productdomain.Product{}, nil },
		updateFn: func(ctx context.Context, id string, command productapplication.UpdateCommand) (*productdomain.Product, error) {
			return &productdomain.Product{}, nil
		},
		deleteFn: func(ctx context.Context, id string) error { return nil },
	}, authorizerMock{
		requireFn: func(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error {
			if authorizationHeader == "Bearer unauthorized" {
				return unauthorizedError
			}
			if authorizationHeader == "Bearer forbidden" {
				return forbiddenError
			}
			return nil
		},
		isUnauthorizedFn: func(err error) bool { return errorspkg.Is(err, unauthorizedError) },
		isForbiddenFn:    func(err error) bool { return errorspkg.Is(err, forbiddenError) },
	})

	server := newHTTPServerForHandler(t, handler)

	unauthorizedReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/products", nil)
	unauthorizedReq.Header.Set("Authorization", "Bearer unauthorized")
	unauthorizedResp := runRequest(t, server, unauthorizedReq)
	if unauthorizedResp.StatusCode != stdhttp.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", unauthorizedResp.StatusCode, stdhttp.StatusUnauthorized)
	}

	forbiddenReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/products", nil)
	forbiddenReq.Header.Set("Authorization", "Bearer forbidden")
	forbiddenResp := runRequest(t, server, forbiddenReq)
	if forbiddenResp.StatusCode != stdhttp.StatusForbidden {
		t.Fatalf("status = %d, want %d", forbiddenResp.StatusCode, stdhttp.StatusForbidden)
	}
}

// newHandlerForTest creates handlers for tests.
func newHandlerForTest(t *testing.T, service productapplication.Service, authorizers ...Authorizer) *Handler {
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

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8140}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(handler.RegisterRoutes)

	return server
}

// runRequest runs HTTP requests against test servers.
func runRequest(t *testing.T, server *corehttp.Server, request *stdhttp.Request) *stdhttp.Response {
	t.Helper()

	response, err := server.App().Test(request)
	if err != nil {
		t.Fatalf("App().Test() error = %v", err)
	}

	return response
}
