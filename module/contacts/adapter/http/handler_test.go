package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	stdhttp "net/http"
	"testing"

	"mannaiah/module/contacts/application"
	"mannaiah/module/contacts/domain"
	"mannaiah/module/contacts/port"
	corehttp "mannaiah/module/core/http"
)

// serviceMock defines application behavior for HTTP handler tests.
type serviceMock struct {
	// createFn defines create behavior.
	createFn func(ctx context.Context, command application.CreateCommand) (*domain.Contact, error)
	// getFn defines get behavior.
	getFn func(ctx context.Context, id string) (*domain.Contact, error)
	// listFn defines list behavior.
	listFn func(ctx context.Context, query port.ListQuery) (*application.ListResult, error)
	// updateFn defines update behavior.
	updateFn func(ctx context.Context, id string, command application.UpdateCommand) (*domain.Contact, error)
	// deleteFn defines delete behavior.
	deleteFn func(ctx context.Context, id string) error
}

// Create executes configured create behavior.
func (m serviceMock) Create(ctx context.Context, command application.CreateCommand) (*domain.Contact, error) {
	return m.createFn(ctx, command)
}

// Get executes configured get behavior.
func (m serviceMock) Get(ctx context.Context, id string) (*domain.Contact, error) {
	return m.getFn(ctx, id)
}

// List executes configured list behavior.
func (m serviceMock) List(ctx context.Context, query port.ListQuery) (*application.ListResult, error) {
	return m.listFn(ctx, query)
}

// Update executes configured update behavior.
func (m serviceMock) Update(ctx context.Context, id string, command application.UpdateCommand) (*domain.Contact, error) {
	return m.updateFn(ctx, id, command)
}

// Delete executes configured delete behavior.
func (m serviceMock) Delete(ctx context.Context, id string) error {
	return m.deleteFn(ctx, id)
}

// authorizerMock defines auth behavior for handler tests.
type authorizerMock struct {
	// requireFn defines auth and permission-check behavior.
	requireFn func(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
	// isUnauthorizedFn defines auth error classification behavior.
	isUnauthorizedFn func(err error) bool
	// isForbiddenFn defines permission error classification behavior.
	isForbiddenFn func(err error) bool
}

// Require executes configured auth and permission-check behavior.
func (m authorizerMock) Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error {
	return m.requireFn(ctx, authorizationHeader, requiredPermissions...)
}

// IsUnauthorized executes configured authentication-error classification behavior.
func (m authorizerMock) IsUnauthorized(err error) bool {
	return m.isUnauthorizedFn(err)
}

// IsForbidden executes configured authorization-error classification behavior.
func (m authorizerMock) IsForbidden(err error) bool {
	return m.isForbiddenFn(err)
}

// TestNewHandlerRejectsNilService verifies constructor validation for nil services.
func TestNewHandlerRejectsNilService(t *testing.T) {
	_, err := NewHandler(nil)
	if !errors.Is(err, ErrNilService) {
		t.Fatalf("NewHandler() error = %v, want ErrNilService", err)
	}
}

// TestHandlerCreateEndpoint verifies POST /contacts behavior.
func TestHandlerCreateEndpoint(t *testing.T) {
	handler := newHandlerForTest(t, serviceMock{
		createFn: func(ctx context.Context, command application.CreateCommand) (*domain.Contact, error) {
			return &domain.Contact{ID: "c-1", Email: command.Email, LegalName: command.LegalName}, nil
		},
		getFn:  func(ctx context.Context, id string) (*domain.Contact, error) { return nil, nil },
		listFn: func(ctx context.Context, query port.ListQuery) (*application.ListResult, error) { return nil, nil },
		updateFn: func(ctx context.Context, id string, command application.UpdateCommand) (*domain.Contact, error) {
			return nil, nil
		},
		deleteFn: func(ctx context.Context, id string) error { return nil },
	})
	server := newHTTPServerForHandler(t, handler)

	body := bytes.NewBufferString(`{"email":"john@example.com","legalName":"Acme"}`)
	req, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/contacts", body)
	req.Header.Set("Content-Type", "application/json")

	resp := runRequest(t, server, req)
	if resp.StatusCode != stdhttp.StatusCreated {
		t.Fatalf("status = %d, want %d", resp.StatusCode, stdhttp.StatusCreated)
	}

	payload := decodeJSONMap(t, resp)
	if payload["id"] != "c-1" {
		t.Fatalf("id = %v, want %q", payload["id"], "c-1")
	}
}

// TestHandlerListEndpoint verifies GET /contacts pagination/filter behavior.
func TestHandlerListEndpoint(t *testing.T) {
	handler := newHandlerForTest(t, serviceMock{
		createFn: func(ctx context.Context, command application.CreateCommand) (*domain.Contact, error) { return nil, nil },
		getFn:    func(ctx context.Context, id string) (*domain.Contact, error) { return nil, nil },
		listFn: func(ctx context.Context, query port.ListQuery) (*application.ListResult, error) {
			if query.Page != 1 || query.Limit != 2 || query.ExcludeIDs[0] != "x" {
				t.Fatalf("unexpected query: %+v", query)
			}
			return &application.ListResult{Data: []domain.Contact{{ID: "c-1", Email: "a@example.com", LegalName: "Acme"}}, Page: 1, Limit: 2, Total: 1, TotalPages: 1}, nil
		},
		updateFn: func(ctx context.Context, id string, command application.UpdateCommand) (*domain.Contact, error) {
			return nil, nil
		},
		deleteFn: func(ctx context.Context, id string) error { return nil },
	})
	server := newHTTPServerForHandler(t, handler)

	req, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/contacts?page=1&limit=2&excludeIds=x", nil)
	resp := runRequest(t, server, req)
	if resp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, stdhttp.StatusOK)
	}

	payload := decodeJSONMap(t, resp)
	if payload["meta"] == nil {
		t.Fatalf("expected meta field")
	}
}

// TestHandlerFindOneUpdateDeleteEndpoints verifies contact by-id CRUD endpoint behavior.
func TestHandlerFindOneUpdateDeleteEndpoints(t *testing.T) {
	handler := newHandlerForTest(t, serviceMock{
		createFn: func(ctx context.Context, command application.CreateCommand) (*domain.Contact, error) { return nil, nil },
		getFn: func(ctx context.Context, id string) (*domain.Contact, error) {
			return &domain.Contact{ID: id, Email: "a@example.com", LegalName: "Acme"}, nil
		},
		listFn: func(ctx context.Context, query port.ListQuery) (*application.ListResult, error) {
			return nil, nil
		},
		updateFn: func(ctx context.Context, id string, command application.UpdateCommand) (*domain.Contact, error) {
			return &domain.Contact{ID: id, Email: "b@example.com", LegalName: "Acme"}, nil
		},
		deleteFn: func(ctx context.Context, id string) error { return nil },
	})
	server := newHTTPServerForHandler(t, handler)

	getReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/contacts/c-1", nil)
	getResp := runRequest(t, server, getReq)
	if getResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("GET status = %d, want %d", getResp.StatusCode, stdhttp.StatusOK)
	}

	updateReq, _ := stdhttp.NewRequest(stdhttp.MethodPatch, "/contacts/c-1", bytes.NewBufferString(`{"email":"b@example.com"}`))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp := runRequest(t, server, updateReq)
	if updateResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("PATCH status = %d, want %d", updateResp.StatusCode, stdhttp.StatusOK)
	}

	deleteReq, _ := stdhttp.NewRequest(stdhttp.MethodDelete, "/contacts/c-1", nil)
	deleteResp := runRequest(t, server, deleteReq)
	if deleteResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("DELETE status = %d, want %d", deleteResp.StatusCode, stdhttp.StatusOK)
	}
}

// TestHandlerMapsErrors verifies mapped error payload behavior.
func TestHandlerMapsErrors(t *testing.T) {
	handler := newHandlerForTest(t, serviceMock{
		createFn: func(ctx context.Context, command application.CreateCommand) (*domain.Contact, error) {
			return nil, port.ErrNotFound
		},
		getFn:  func(ctx context.Context, id string) (*domain.Contact, error) { return nil, nil },
		listFn: func(ctx context.Context, query port.ListQuery) (*application.ListResult, error) { return nil, nil },
		updateFn: func(ctx context.Context, id string, command application.UpdateCommand) (*domain.Contact, error) {
			return nil, nil
		},
		deleteFn: func(ctx context.Context, id string) error { return nil },
	})
	server := newHTTPServerForHandler(t, handler)

	req, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/contacts", bytes.NewBufferString(`{"email":"john@example.com","legalName":"Acme"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := runRequest(t, server, req)

	if resp.StatusCode != stdhttp.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, stdhttp.StatusNotFound)
	}
	payload := decodeJSONMap(t, resp)
	if payload["message"] != "contact_not_found" {
		t.Fatalf("message = %v, want %q", payload["message"], "contact_not_found")
	}
}

// TestHandlerInvalidPayloadAndQuery verifies invalid request payload and query handling.
func TestHandlerInvalidPayloadAndQuery(t *testing.T) {
	handler := newHandlerForTest(t, serviceMock{
		createFn: func(ctx context.Context, command application.CreateCommand) (*domain.Contact, error) {
			return &domain.Contact{}, nil
		},
		getFn: func(ctx context.Context, id string) (*domain.Contact, error) {
			return &domain.Contact{ID: id}, nil
		},
		listFn: func(ctx context.Context, query port.ListQuery) (*application.ListResult, error) {
			return &application.ListResult{}, nil
		},
		updateFn: func(ctx context.Context, id string, command application.UpdateCommand) (*domain.Contact, error) {
			return &domain.Contact{ID: id}, nil
		},
		deleteFn: func(ctx context.Context, id string) error { return nil },
	})
	server := newHTTPServerForHandler(t, handler)

	createReq, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/contacts", bytes.NewBufferString("{invalid"))
	createReq.Header.Set("Content-Type", "application/json")
	createResp := runRequest(t, server, createReq)
	if createResp.StatusCode != stdhttp.StatusBadRequest {
		t.Fatalf("POST invalid payload status = %d, want %d", createResp.StatusCode, stdhttp.StatusBadRequest)
	}

	listReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/contacts?page=abc&limit=1", nil)
	listResp := runRequest(t, server, listReq)
	if listResp.StatusCode != stdhttp.StatusBadRequest {
		t.Fatalf("GET invalid query status = %d, want %d", listResp.StatusCode, stdhttp.StatusBadRequest)
	}
}

// TestMapErrorVariants verifies direct error mapping branches.
func TestMapErrorVariants(t *testing.T) {
	handler := &Handler{}
	if mapped := handler.mapError(port.ErrNotFound); mapped == nil {
		t.Fatalf("expected mapped not-found error")
	}
	if mapped := handler.mapError(application.ErrInvalidID); mapped == nil {
		t.Fatalf("expected mapped invalid-id error")
	}
	if mapped := handler.mapError(domain.ErrEmailRequired); mapped == nil {
		t.Fatalf("expected mapped invalid-contact error")
	}
	if mapped := handler.mapError(ErrInvalidQuery); mapped == nil {
		t.Fatalf("expected mapped invalid-query error")
	}
	if mapped := handler.mapError(errors.New("boom")); mapped == nil {
		t.Fatalf("expected mapped generic error")
	}
}

// TestHandlerAuthEnforcement verifies route-level authentication and permission behavior.
func TestHandlerAuthEnforcement(t *testing.T) {
	unauthorizedError := errors.New("unauthorized")
	forbiddenError := errors.New("forbidden")

	handler := newHandlerForTest(t, serviceMock{
		createFn: func(ctx context.Context, command application.CreateCommand) (*domain.Contact, error) {
			return &domain.Contact{}, nil
		},
		getFn: func(ctx context.Context, id string) (*domain.Contact, error) {
			return &domain.Contact{ID: id}, nil
		},
		listFn: func(ctx context.Context, query port.ListQuery) (*application.ListResult, error) {
			return &application.ListResult{}, nil
		},
		updateFn: func(ctx context.Context, id string, command application.UpdateCommand) (*domain.Contact, error) {
			return &domain.Contact{ID: id}, nil
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
			if len(requiredPermissions) != 1 || requiredPermissions[0] != "contacts:read" {
				t.Fatalf("requiredPermissions = %#v, want contacts:read", requiredPermissions)
			}
			return nil
		},
		isUnauthorizedFn: func(err error) bool {
			return errors.Is(err, unauthorizedError)
		},
		isForbiddenFn: func(err error) bool {
			return errors.Is(err, forbiddenError)
		},
	})
	server := newHTTPServerForHandler(t, handler)

	unauthorizedReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/contacts?page=1&limit=1", nil)
	unauthorizedReq.Header.Set("Authorization", "Bearer unauthorized")
	unauthorizedResp := runRequest(t, server, unauthorizedReq)
	if unauthorizedResp.StatusCode != stdhttp.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", unauthorizedResp.StatusCode, stdhttp.StatusUnauthorized)
	}

	forbiddenReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/contacts?page=1&limit=1", nil)
	forbiddenReq.Header.Set("Authorization", "Bearer forbidden")
	forbiddenResp := runRequest(t, server, forbiddenReq)
	if forbiddenResp.StatusCode != stdhttp.StatusForbidden {
		t.Fatalf("status = %d, want %d", forbiddenResp.StatusCode, stdhttp.StatusForbidden)
	}

	allowedReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/contacts?page=1&limit=1", nil)
	allowedReq.Header.Set("Authorization", "Bearer ok")
	allowedResp := runRequest(t, server, allowedReq)
	if allowedResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("status = %d, want %d", allowedResp.StatusCode, stdhttp.StatusOK)
	}
}

// TestHandlerSetAuthorizer verifies authorizer setter behavior.
func TestHandlerSetAuthorizer(t *testing.T) {
	handler := &Handler{}
	handler.SetAuthorizer(nil)
}

// newHandlerForTest creates a handler and fails test on constructor errors.
func newHandlerForTest(t *testing.T, service application.Service, authorizers ...Authorizer) *Handler {
	t.Helper()

	handler, err := NewHandler(service, authorizers...)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	return handler
}

// newHTTPServerForHandler creates a core HTTP server and registers handler routes.
func newHTTPServerForHandler(t *testing.T, handler *Handler) *corehttp.Server {
	t.Helper()

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8100}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(handler.RegisterRoutes)

	return server
}

// runRequest executes an HTTP request against a test server.
func runRequest(t *testing.T, server *corehttp.Server, req *stdhttp.Request) *stdhttp.Response {
	t.Helper()

	resp, err := server.App().Test(req)
	if err != nil {
		t.Fatalf("App().Test() error = %v", err)
	}

	return resp
}

// decodeJSONMap decodes JSON response payloads into map values.
func decodeJSONMap(t *testing.T, resp *stdhttp.Response) map[string]any {
	t.Helper()

	defer func() {
		_ = resp.Body.Close()
	}()

	payload := map[string]any{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	return payload
}
