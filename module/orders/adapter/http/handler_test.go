package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	stdhttp "net/http"
	"testing"

	corehttp "mannaiah/module/core/http"
	ordersapplication "mannaiah/module/orders/application"
	ordersdomain "mannaiah/module/orders/domain"
	ordersport "mannaiah/module/orders/port"
)

// serviceMock defines orders application behavior for HTTP handler tests.
type serviceMock struct {
	// createFn defines create behavior.
	createFn func(ctx context.Context, command ordersapplication.CreateCommand) (*ordersdomain.Order, error)
	// getFn defines get behavior.
	getFn func(ctx context.Context, id string) (*ordersdomain.Order, error)
	// listFn defines list behavior.
	listFn func(ctx context.Context, query ordersapplication.ListQuery) (*ordersapplication.ListResult, error)
	// updateFn defines update behavior.
	updateFn func(ctx context.Context, id string, command ordersapplication.UpdateCommand) (*ordersdomain.Order, error)
	// updateStatusFn defines update-status behavior.
	updateStatusFn func(ctx context.Context, id string, command ordersapplication.UpdateStatusCommand) (*ordersdomain.Order, error)
	// addCommentFn defines add-comment behavior.
	addCommentFn func(ctx context.Context, id string, command ordersapplication.AddCommentCommand) (*ordersdomain.Order, error)
	// updateCommentFn defines update-comment behavior.
	updateCommentFn func(ctx context.Context, id string, commentID string, command ordersapplication.UpdateCommentCommand) (*ordersdomain.Order, error)
	// deleteCommentFn defines delete-comment behavior.
	deleteCommentFn func(ctx context.Context, id string, commentID string, command ordersapplication.DeleteCommentCommand) (*ordersdomain.Order, error)
}

// Create executes configured create behavior.
func (m serviceMock) Create(ctx context.Context, command ordersapplication.CreateCommand) (*ordersdomain.Order, error) {
	return m.createFn(ctx, command)
}

// Get executes configured get behavior.
func (m serviceMock) Get(ctx context.Context, id string) (*ordersdomain.Order, error) {
	return m.getFn(ctx, id)
}

// List executes configured list behavior.
func (m serviceMock) List(ctx context.Context, query ordersapplication.ListQuery) (*ordersapplication.ListResult, error) {
	return m.listFn(ctx, query)
}

// Update executes configured update behavior.
func (m serviceMock) Update(ctx context.Context, id string, command ordersapplication.UpdateCommand) (*ordersdomain.Order, error) {
	return m.updateFn(ctx, id, command)
}

// UpdateStatus executes configured update-status behavior.
func (m serviceMock) UpdateStatus(ctx context.Context, id string, command ordersapplication.UpdateStatusCommand) (*ordersdomain.Order, error) {
	return m.updateStatusFn(ctx, id, command)
}

// AddComment executes configured add-comment behavior.
func (m serviceMock) AddComment(ctx context.Context, id string, command ordersapplication.AddCommentCommand) (*ordersdomain.Order, error) {
	return m.addCommentFn(ctx, id, command)
}

// UpdateComment executes configured update-comment behavior.
func (m serviceMock) UpdateComment(ctx context.Context, id string, commentID string, command ordersapplication.UpdateCommentCommand) (*ordersdomain.Order, error) {
	return m.updateCommentFn(ctx, id, commentID, command)
}

// DeleteComment executes configured delete-comment behavior.
func (m serviceMock) DeleteComment(ctx context.Context, id string, commentID string, command ordersapplication.DeleteCommentCommand) (*ordersdomain.Order, error) {
	return m.deleteCommentFn(ctx, id, commentID, command)
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

// IsUnauthorized executes configured auth error classification behavior.
func (m authorizerMock) IsUnauthorized(err error) bool {
	return m.isUnauthorizedFn(err)
}

// IsForbidden executes configured permission error classification behavior.
func (m authorizerMock) IsForbidden(err error) bool {
	return m.isForbiddenFn(err)
}

// TestNewHandlerRejectsNilService verifies constructor validation for nil services.
func TestNewHandlerRejectsNilService(t *testing.T) {
	if _, err := NewHandler(nil); !errors.Is(err, ErrNilService) {
		t.Fatalf("NewHandler() error = %v, want ErrNilService", err)
	}
}

// TestOrderEndpoints verifies order endpoint behavior.
func TestOrderEndpoints(t *testing.T) {
	handler := newHandlerForTest(t, serviceMock{
		createFn: func(ctx context.Context, command ordersapplication.CreateCommand) (*ordersdomain.Order, error) {
			if command.Items[0].SKU != "SKU-1" || command.ShippingAddress == nil || command.Metadata["source"] != "woo" || command.Items[0].Value != 12000 || len(command.ShippingCharges) != 1 || command.ShippingCharges[0].MethodID != "flat_rate" || len(command.AppliedCoupons) != 1 || command.AppliedCoupons[0].Code != "WELCOME10" {
				t.Fatalf("unexpected create command: %+v", command)
			}
			return &ordersdomain.Order{ID: "o-1", Identifier: command.Identifier, Realm: command.Realm}, nil
		},
		getFn: func(ctx context.Context, id string) (*ordersdomain.Order, error) {
			return &ordersdomain.Order{ID: id, Identifier: "wo-1"}, nil
		},
		listFn: func(ctx context.Context, query ordersapplication.ListQuery) (*ordersapplication.ListResult, error) {
			if query.Page != 1 || query.Limit != 2 || query.Status != ordersdomain.StatusPending {
				t.Fatalf("unexpected list query: %+v", query)
			}
			return &ordersapplication.ListResult{
				Data:       []ordersdomain.Order{{ID: "o-1", Identifier: "wo-1"}},
				Page:       1,
				Limit:      2,
				Total:      1,
				TotalPages: 1,
			}, nil
		},
		updateFn: func(ctx context.Context, id string, command ordersapplication.UpdateCommand) (*ordersdomain.Order, error) {
			if command.ShippingAddress == nil || command.ShippingAddress.CityCode != "05001" || command.Source != "woocommerce_plugin" || command.AppliedCoupons == nil || len(*command.AppliedCoupons) != 1 || (*command.AppliedCoupons)[0].Code != "WELCOME10" {
				t.Fatalf("unexpected update command: %+v", command)
			}
			return &ordersdomain.Order{ID: id, Identifier: "wo-1"}, nil
		},
		updateStatusFn: func(ctx context.Context, id string, command ordersapplication.UpdateStatusCommand) (*ordersdomain.Order, error) {
			if command.Status != ordersdomain.StatusCompleted || command.Author != "user" {
				t.Fatalf("unexpected update-status command: %+v", command)
			}
			return &ordersdomain.Order{ID: id, CurrentStatus: command.Status}, nil
		},
		addCommentFn: func(ctx context.Context, id string, command ordersapplication.AddCommentCommand) (*ordersdomain.Order, error) {
			if command.Author != "user" || command.Comment != "Order validated" || !command.Internal {
				t.Fatalf("unexpected add-comment command: %+v", command)
			}
			return &ordersdomain.Order{ID: id, Comments: []ordersdomain.Comment{{ID: "10", Author: command.Author, Comment: command.Comment, Internal: command.Internal}}}, nil
		},
		updateCommentFn: func(ctx context.Context, id string, commentID string, command ordersapplication.UpdateCommentCommand) (*ordersdomain.Order, error) {
			if commentID != "10" {
				t.Fatalf("commentID = %q, want %q", commentID, "10")
			}
			if command.Comment == nil || *command.Comment != "Order updated" || command.Source != "woocommerce_plugin" {
				t.Fatalf("unexpected update-comment command: %+v", command)
			}
			return &ordersdomain.Order{ID: id, Comments: []ordersdomain.Comment{{ID: "10", Author: "user", Comment: "Order updated"}}}, nil
		},
		deleteCommentFn: func(ctx context.Context, id string, commentID string, command ordersapplication.DeleteCommentCommand) (*ordersdomain.Order, error) {
			if commentID != "10" {
				t.Fatalf("commentID = %q, want %q", commentID, "10")
			}
			if command.Source != "woocommerce_plugin" {
				t.Fatalf("command.Source = %q, want %q", command.Source, "woocommerce_plugin")
			}
			return &ordersdomain.Order{ID: id, Comments: []ordersdomain.Comment{}}, nil
		},
	})
	server := newHTTPServerForHandler(t, handler)

	createReq, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/orders", bytes.NewBufferString(`{"identifier":"wo-1","realm":"woocommerce","contactId":"c-1","metadata":{"source":"woo"},"items":[{"sku":"SKU-1","quantity":1,"value":12000}],"shippingAddress":{"address":"A","cityCode":"11001"},"shippingCharges":[{"methodId":"flat_rate","methodTitle":"Flat Rate","price":9000}],"appliedCoupons":[{"code":"WELCOME10","discountType":"fixed","discountAmount":5000}]}`))
	createReq.Header.Set("Content-Type", "application/json")
	createResp := runRequest(t, server, createReq)
	if createResp.StatusCode != stdhttp.StatusCreated {
		t.Fatalf("POST /orders status = %d, want %d", createResp.StatusCode, stdhttp.StatusCreated)
	}

	listReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/orders?page=1&limit=2&status=pending", nil)
	listResp := runRequest(t, server, listReq)
	if listResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("GET /orders status = %d, want %d", listResp.StatusCode, stdhttp.StatusOK)
	}

	getReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/orders/o-1", nil)
	getResp := runRequest(t, server, getReq)
	if getResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("GET /orders/:id status = %d, want %d", getResp.StatusCode, stdhttp.StatusOK)
	}

	updateOrderReq, _ := stdhttp.NewRequest(stdhttp.MethodPatch, "/orders/o-1", bytes.NewBufferString(`{"shippingAddress":{"address":"Street 1","cityCode":"05001"},"appliedCoupons":[{"code":"WELCOME10","discountType":"fixed","discountAmount":5000}]}`))
	updateOrderReq.Header.Set("Content-Type", "application/json")
	updateOrderReq.Header.Set("X-Sync-Source", "woocommerce_plugin")
	updateOrderResp := runRequest(t, server, updateOrderReq)
	if updateOrderResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("PATCH /orders/:id status = %d, want %d", updateOrderResp.StatusCode, stdhttp.StatusOK)
	}

	updateReq, _ := stdhttp.NewRequest(stdhttp.MethodPatch, "/orders/o-1/status", bytes.NewBufferString(`{"status":"COMPLETED","author":"user","description":"done"}`))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp := runRequest(t, server, updateReq)
	if updateResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("PATCH /orders/:id/status status = %d, want %d", updateResp.StatusCode, stdhttp.StatusOK)
	}

	addCommentReq, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/orders/o-1/comments", bytes.NewBufferString(`{"author":"user","comment":"Order validated","internal":true}`))
	addCommentReq.Header.Set("Content-Type", "application/json")
	addCommentResp := runRequest(t, server, addCommentReq)
	if addCommentResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("POST /orders/:id/comments status = %d, want %d", addCommentResp.StatusCode, stdhttp.StatusOK)
	}

	updateCommentReq, _ := stdhttp.NewRequest(stdhttp.MethodPatch, "/orders/o-1/comments/10", bytes.NewBufferString(`{"comment":"Order updated"}`))
	updateCommentReq.Header.Set("Content-Type", "application/json")
	updateCommentReq.Header.Set("X-Sync-Source", "woocommerce_plugin")
	updateCommentResp := runRequest(t, server, updateCommentReq)
	if updateCommentResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("PATCH /orders/:id/comments/:commentId status = %d, want %d", updateCommentResp.StatusCode, stdhttp.StatusOK)
	}

	deleteCommentReq, _ := stdhttp.NewRequest(stdhttp.MethodDelete, "/orders/o-1/comments/10", nil)
	deleteCommentReq.Header.Set("X-Sync-Source", "woocommerce_plugin")
	deleteCommentResp := runRequest(t, server, deleteCommentReq)
	if deleteCommentResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("DELETE /orders/:id/comments/:commentId status = %d, want %d", deleteCommentResp.StatusCode, stdhttp.StatusOK)
	}
}

// TestHandlerInvalidPayloadAndQuery verifies invalid request payload and query behavior.
func TestHandlerInvalidPayloadAndQuery(t *testing.T) {
	handler := newHandlerForTest(t, serviceMock{
		createFn: func(ctx context.Context, command ordersapplication.CreateCommand) (*ordersdomain.Order, error) {
			return &ordersdomain.Order{}, nil
		},
		getFn: func(ctx context.Context, id string) (*ordersdomain.Order, error) {
			return &ordersdomain.Order{ID: id}, nil
		},
		listFn: func(ctx context.Context, query ordersapplication.ListQuery) (*ordersapplication.ListResult, error) {
			return &ordersapplication.ListResult{}, nil
		},
		updateFn: func(ctx context.Context, id string, command ordersapplication.UpdateCommand) (*ordersdomain.Order, error) {
			return &ordersdomain.Order{ID: id}, nil
		},
		updateStatusFn: func(ctx context.Context, id string, command ordersapplication.UpdateStatusCommand) (*ordersdomain.Order, error) {
			return &ordersdomain.Order{ID: id}, nil
		},
		addCommentFn: func(ctx context.Context, id string, command ordersapplication.AddCommentCommand) (*ordersdomain.Order, error) {
			return &ordersdomain.Order{ID: id}, nil
		},
		updateCommentFn: func(ctx context.Context, id string, commentID string, command ordersapplication.UpdateCommentCommand) (*ordersdomain.Order, error) {
			return &ordersdomain.Order{ID: id}, nil
		},
		deleteCommentFn: func(ctx context.Context, id string, commentID string, command ordersapplication.DeleteCommentCommand) (*ordersdomain.Order, error) {
			return &ordersdomain.Order{ID: id}, nil
		},
	})
	server := newHTTPServerForHandler(t, handler)

	createReq, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/orders", bytes.NewBufferString("{invalid"))
	createReq.Header.Set("Content-Type", "application/json")
	createResp := runRequest(t, server, createReq)
	if createResp.StatusCode != stdhttp.StatusBadRequest {
		t.Fatalf("POST invalid payload status = %d, want %d", createResp.StatusCode, stdhttp.StatusBadRequest)
	}

	listReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/orders?page=abc&limit=2", nil)
	listResp := runRequest(t, server, listReq)
	if listResp.StatusCode != stdhttp.StatusBadRequest {
		t.Fatalf("GET invalid query status = %d, want %d", listResp.StatusCode, stdhttp.StatusBadRequest)
	}

	updateReq, _ := stdhttp.NewRequest(stdhttp.MethodPatch, "/orders/o-1", bytes.NewBufferString("{invalid"))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp := runRequest(t, server, updateReq)
	if updateResp.StatusCode != stdhttp.StatusBadRequest {
		t.Fatalf("PATCH invalid payload status = %d, want %d", updateResp.StatusCode, stdhttp.StatusBadRequest)
	}

	updateCommentReq, _ := stdhttp.NewRequest(stdhttp.MethodPatch, "/orders/o-1/comments/10", bytes.NewBufferString("{invalid"))
	updateCommentReq.Header.Set("Content-Type", "application/json")
	updateCommentResp := runRequest(t, server, updateCommentReq)
	if updateCommentResp.StatusCode != stdhttp.StatusBadRequest {
		t.Fatalf("PATCH comment invalid payload status = %d, want %d", updateCommentResp.StatusCode, stdhttp.StatusBadRequest)
	}
}

// TestMapErrorVariants verifies direct error mapping branches.
func TestMapErrorVariants(t *testing.T) {
	handler := &Handler{}
	for _, value := range []error{
		ordersport.ErrNotFound,
		ordersport.ErrCommentNotFound,
		ordersapplication.ErrInvalidID,
		ordersapplication.ErrInvalidCommentID,
		ordersapplication.ErrEmptyCommentUpdate,
		ordersport.ErrDuplicateIdentifier,
		ordersport.ErrCustomerNotFound,
		ErrInvalidQuery,
		ordersapplication.ErrStatusAuthorRequired,
		ordersapplication.ErrEmptyOrderUpdate,
		ordersdomain.ErrIdentifierRequired,
		ordersdomain.ErrRealmRequired,
		ordersdomain.ErrContactIDRequired,
		ordersdomain.ErrItemsRequired,
		ordersdomain.ErrItemIdentifierRequired,
		ordersdomain.ErrItemQuantityInvalid,
		ordersdomain.ErrStatusInvalid,
		ordersdomain.ErrStatusAuthorRequired,
		errors.New("boom"),
	} {
		if mapped := handler.mapError(value); mapped == nil {
			t.Fatalf("expected mapped error for %v", value)
		}
	}
}

// TestHandlerAuthEnforcement verifies route-level authentication and authorization behavior.
func TestHandlerAuthEnforcement(t *testing.T) {
	unauthorizedError := errors.New("unauthorized")
	forbiddenError := errors.New("forbidden")

	handler := newHandlerForTest(t, serviceMock{
		createFn: func(ctx context.Context, command ordersapplication.CreateCommand) (*ordersdomain.Order, error) {
			return &ordersdomain.Order{}, nil
		},
		getFn: func(ctx context.Context, id string) (*ordersdomain.Order, error) {
			return &ordersdomain.Order{}, nil
		},
		listFn: func(ctx context.Context, query ordersapplication.ListQuery) (*ordersapplication.ListResult, error) {
			return &ordersapplication.ListResult{}, nil
		},
		updateFn: func(ctx context.Context, id string, command ordersapplication.UpdateCommand) (*ordersdomain.Order, error) {
			return &ordersdomain.Order{}, nil
		},
		updateStatusFn: func(ctx context.Context, id string, command ordersapplication.UpdateStatusCommand) (*ordersdomain.Order, error) {
			return &ordersdomain.Order{}, nil
		},
		addCommentFn: func(ctx context.Context, id string, command ordersapplication.AddCommentCommand) (*ordersdomain.Order, error) {
			return &ordersdomain.Order{}, nil
		},
		updateCommentFn: func(ctx context.Context, id string, commentID string, command ordersapplication.UpdateCommentCommand) (*ordersdomain.Order, error) {
			return &ordersdomain.Order{}, nil
		},
		deleteCommentFn: func(ctx context.Context, id string, commentID string, command ordersapplication.DeleteCommentCommand) (*ordersdomain.Order, error) {
			return &ordersdomain.Order{}, nil
		},
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
		isUnauthorizedFn: func(err error) bool { return errors.Is(err, unauthorizedError) },
		isForbiddenFn:    func(err error) bool { return errors.Is(err, forbiddenError) },
	})

	server := newHTTPServerForHandler(t, handler)

	unauthorizedReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/orders", nil)
	unauthorizedReq.Header.Set("Authorization", "Bearer unauthorized")
	unauthorizedResp := runRequest(t, server, unauthorizedReq)
	if unauthorizedResp.StatusCode != stdhttp.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", unauthorizedResp.StatusCode, stdhttp.StatusUnauthorized)
	}

	forbiddenReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/orders", nil)
	forbiddenReq.Header.Set("Authorization", "Bearer forbidden")
	forbiddenResp := runRequest(t, server, forbiddenReq)
	if forbiddenResp.StatusCode != stdhttp.StatusForbidden {
		t.Fatalf("status = %d, want %d", forbiddenResp.StatusCode, stdhttp.StatusForbidden)
	}
}

// TestResolveCommandSource verifies payload and header source resolution behavior.
func TestResolveCommandSource(t *testing.T) {
	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8161}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(func(router corehttp.Router) {
		router.Get("/source", func(ctx corehttp.Context) error {
			return ctx.JSON(map[string]string{
				"body":   resolveCommandSource(ctx, "body_source"),
				"header": resolveCommandSource(ctx, ""),
			})
		})
	})

	request, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/source", nil)
	request.Header.Set("X-Sync-Source", "header_source")
	response := runRequest(t, server, request)
	if response.StatusCode != stdhttp.StatusOK {
		t.Fatalf("GET /source status = %d, want %d", response.StatusCode, stdhttp.StatusOK)
	}

	body := decodeJSONBody(t, response)
	if body["body"] != "body_source" {
		t.Fatalf("body source = %q, want %q", body["body"], "body_source")
	}
	if body["header"] != "header_source" {
		t.Fatalf("header source = %q, want %q", body["header"], "header_source")
	}
}

// decodeJSONBody decodes response bodies to JSON map payloads.
func decodeJSONBody(t *testing.T, response *stdhttp.Response) map[string]string {
	t.Helper()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("ReadAll(response.Body) error = %v", err)
	}

	result := map[string]string{}
	if err := json.Unmarshal(payload, &result); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	return result
}

// newHandlerForTest creates handlers for tests.
func newHandlerForTest(t *testing.T, service ordersapplication.Service, authorizers ...Authorizer) *Handler {
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

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8160}, nil)
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
