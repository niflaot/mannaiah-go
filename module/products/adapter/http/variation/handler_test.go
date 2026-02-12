package variation

import (
	"bytes"
	"context"
	errorspkg "errors"
	stdhttp "net/http"
	"testing"

	corehttp "mannaiah/module/core/http"
	variationapplication "mannaiah/module/products/application/variation"
	variationdomain "mannaiah/module/products/domain/variation"
	variationport "mannaiah/module/products/port/variation"
)

// serviceMock defines variation service behavior for HTTP handler tests.
type serviceMock struct {
	// createFn defines create behavior.
	createFn func(ctx context.Context, command variationapplication.CreateCommand) (*variationdomain.Variation, error)
	// getFn defines get behavior.
	getFn func(ctx context.Context, id string) (*variationdomain.Variation, error)
	// listFn defines list behavior.
	listFn func(ctx context.Context) ([]variationdomain.Variation, error)
	// updateFn defines update behavior.
	updateFn func(ctx context.Context, id string, command variationapplication.UpdateCommand) (*variationdomain.Variation, error)
	// deleteFn defines delete behavior.
	deleteFn func(ctx context.Context, id string) error
}

// Create executes configured create behavior.
func (m serviceMock) Create(ctx context.Context, command variationapplication.CreateCommand) (*variationdomain.Variation, error) {
	return m.createFn(ctx, command)
}

// Get executes configured get behavior.
func (m serviceMock) Get(ctx context.Context, id string) (*variationdomain.Variation, error) {
	return m.getFn(ctx, id)
}

// List executes configured list behavior.
func (m serviceMock) List(ctx context.Context) ([]variationdomain.Variation, error) {
	return m.listFn(ctx)
}

// Update executes configured update behavior.
func (m serviceMock) Update(ctx context.Context, id string, command variationapplication.UpdateCommand) (*variationdomain.Variation, error) {
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

// TestVariationEndpoints verifies CRUD endpoint behavior.
func TestVariationEndpoints(t *testing.T) {
	handler := newHandlerForTest(t, serviceMock{
		createFn: func(ctx context.Context, command variationapplication.CreateCommand) (*variationdomain.Variation, error) {
			return &variationdomain.Variation{ID: "v-1", Name: command.Name, Definition: command.Definition, Value: command.Value}, nil
		},
		getFn: func(ctx context.Context, id string) (*variationdomain.Variation, error) {
			return &variationdomain.Variation{ID: id, Name: "Red", Definition: variationdomain.DefinitionColor, Value: "#FF0000"}, nil
		},
		listFn: func(ctx context.Context) ([]variationdomain.Variation, error) {
			return []variationdomain.Variation{{ID: "v-1", Name: "Red", Definition: variationdomain.DefinitionColor, Value: "#FF0000"}}, nil
		},
		updateFn: func(ctx context.Context, id string, command variationapplication.UpdateCommand) (*variationdomain.Variation, error) {
			return &variationdomain.Variation{ID: id, Name: "Dark Red", Definition: variationdomain.DefinitionColor, Value: "#8B0000"}, nil
		},
		deleteFn: func(ctx context.Context, id string) error { return nil },
	})
	server := newHTTPServerForHandler(t, handler)

	createReq, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/variations", bytes.NewBufferString(`{"name":"Red","definition":"COLOR","value":"#FF0000"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createResp := runRequest(t, server, createReq)
	if createResp.StatusCode != stdhttp.StatusCreated {
		t.Fatalf("POST status = %d, want %d", createResp.StatusCode, stdhttp.StatusCreated)
	}

	listReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/variations", nil)
	listResp := runRequest(t, server, listReq)
	if listResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("GET /variations status = %d, want %d", listResp.StatusCode, stdhttp.StatusOK)
	}

	getReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/variations/v-1", nil)
	getResp := runRequest(t, server, getReq)
	if getResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("GET /variations/:id status = %d, want %d", getResp.StatusCode, stdhttp.StatusOK)
	}

	updateReq, _ := stdhttp.NewRequest(stdhttp.MethodPatch, "/variations/v-1", bytes.NewBufferString(`{"name":"Dark Red","definition":"SIZE","value":"#8B0000"}`))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp := runRequest(t, server, updateReq)
	if updateResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("PATCH status = %d, want %d", updateResp.StatusCode, stdhttp.StatusOK)
	}

	deleteReq, _ := stdhttp.NewRequest(stdhttp.MethodDelete, "/variations/v-1", nil)
	deleteResp := runRequest(t, server, deleteReq)
	if deleteResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("DELETE status = %d, want %d", deleteResp.StatusCode, stdhttp.StatusOK)
	}
}

// TestHandlerInvalidPayload verifies invalid payload behavior.
func TestHandlerInvalidPayload(t *testing.T) {
	handler := newHandlerForTest(t, serviceMock{
		createFn: func(ctx context.Context, command variationapplication.CreateCommand) (*variationdomain.Variation, error) {
			return &variationdomain.Variation{}, nil
		},
		getFn: func(ctx context.Context, id string) (*variationdomain.Variation, error) {
			return &variationdomain.Variation{}, nil
		},
		listFn: func(ctx context.Context) ([]variationdomain.Variation, error) {
			return []variationdomain.Variation{}, nil
		},
		updateFn: func(ctx context.Context, id string, command variationapplication.UpdateCommand) (*variationdomain.Variation, error) {
			return &variationdomain.Variation{}, nil
		},
		deleteFn: func(ctx context.Context, id string) error { return nil },
	})
	server := newHTTPServerForHandler(t, handler)

	req, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/variations", bytes.NewBufferString("{"))
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
		variationport.ErrNotFound,
		variationapplication.ErrInvalidID,
		variationdomain.ErrNameRequired,
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
		createFn: func(ctx context.Context, command variationapplication.CreateCommand) (*variationdomain.Variation, error) {
			return &variationdomain.Variation{}, nil
		},
		getFn: func(ctx context.Context, id string) (*variationdomain.Variation, error) {
			return &variationdomain.Variation{}, nil
		},
		listFn: func(ctx context.Context) ([]variationdomain.Variation, error) {
			return []variationdomain.Variation{}, nil
		},
		updateFn: func(ctx context.Context, id string, command variationapplication.UpdateCommand) (*variationdomain.Variation, error) {
			return &variationdomain.Variation{}, nil
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

	unauthorizedReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/variations", nil)
	unauthorizedReq.Header.Set("Authorization", "Bearer unauthorized")
	unauthorizedResp := runRequest(t, server, unauthorizedReq)
	if unauthorizedResp.StatusCode != stdhttp.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", unauthorizedResp.StatusCode, stdhttp.StatusUnauthorized)
	}

	forbiddenReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/variations", nil)
	forbiddenReq.Header.Set("Authorization", "Bearer forbidden")
	forbiddenResp := runRequest(t, server, forbiddenReq)
	if forbiddenResp.StatusCode != stdhttp.StatusForbidden {
		t.Fatalf("status = %d, want %d", forbiddenResp.StatusCode, stdhttp.StatusForbidden)
	}
}

// newHandlerForTest creates handlers for tests.
func newHandlerForTest(t *testing.T, service variationapplication.Service, authorizers ...Authorizer) *Handler {
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

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8141}, nil)
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
