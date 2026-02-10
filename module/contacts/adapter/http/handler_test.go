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
	if mapped := mapError(port.ErrNotFound); mapped == nil {
		t.Fatalf("expected mapped not-found error")
	}
	if mapped := mapError(application.ErrInvalidID); mapped == nil {
		t.Fatalf("expected mapped invalid-id error")
	}
	if mapped := mapError(domain.ErrEmailRequired); mapped == nil {
		t.Fatalf("expected mapped invalid-contact error")
	}
	if mapped := mapError(ErrInvalidQuery); mapped == nil {
		t.Fatalf("expected mapped invalid-query error")
	}
	if mapped := mapError(errors.New("boom")); mapped == nil {
		t.Fatalf("expected mapped generic error")
	}
}

// newHandlerForTest creates a handler and fails test on constructor errors.
func newHandlerForTest(t *testing.T, service application.Service) *Handler {
	t.Helper()

	handler, err := NewHandler(service)
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
