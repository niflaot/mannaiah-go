package http

import (
	"context"
	errorspkg "errors"
	stdhttp "net/http"
	"strings"
	"testing"

	corehttp "mannaiah/module/core/http"
	woocontactservice "mannaiah/module/woocommerce/application/contact/service"
	woocouponservice "mannaiah/module/woocommerce/application/coupon/service"
	wooorderservice "mannaiah/module/woocommerce/application/order/service"
)

// contactsServiceMock defines WooCommerce contacts service behavior for handler tests.
type contactsServiceMock struct {
	// summary defines sync summary responses.
	summary *woocontactservice.SyncSummary
	// syncErr defines sync execution errors.
	syncErr error
	// syncByEmailSummary defines targeted sync summary responses.
	syncByEmailSummary *woocontactservice.SyncSummary
	// syncByEmailErr defines targeted sync execution errors.
	syncByEmailErr error
	// lastEmail captures last targeted email values.
	lastEmail string
}

// ValidateIntegration validates integration state.
func (m *contactsServiceMock) ValidateIntegration(ctx context.Context) error {
	return nil
}

// SyncContacts performs sync behavior.
func (m *contactsServiceMock) SyncContacts(ctx context.Context, trigger string) (*woocontactservice.SyncSummary, error) {
	if m.syncErr != nil {
		return nil, m.syncErr
	}
	if m.summary != nil {
		return m.summary, nil
	}

	return &woocontactservice.SyncSummary{Trigger: trigger}, nil
}

// SyncContactByEmail performs targeted sync behavior.
func (m *contactsServiceMock) SyncContactByEmail(ctx context.Context, trigger string, email string) (*woocontactservice.SyncSummary, error) {
	m.lastEmail = strings.Clone(email)
	if m.syncByEmailErr != nil {
		return nil, m.syncByEmailErr
	}
	if m.syncByEmailSummary != nil {
		return m.syncByEmailSummary, nil
	}

	return &woocontactservice.SyncSummary{Trigger: trigger, Processed: 1}, nil
}

// ordersServiceMock defines WooCommerce orders service behavior for handler tests.
type ordersServiceMock struct {
	// summary defines sync summary responses.
	summary *wooorderservice.SyncSummary
	// syncErr defines sync execution errors.
	syncErr error
	// syncByIDSummary defines targeted sync summary responses.
	syncByIDSummary *wooorderservice.SyncSummary
	// syncByIDErr defines targeted sync execution errors.
	syncByIDErr error
	// lastOrderID captures last targeted order-id values.
	lastOrderID int
}

// ValidateIntegration validates integration state.
func (m *ordersServiceMock) ValidateIntegration(ctx context.Context) error {
	return nil
}

// SyncOrders performs sync behavior.
func (m *ordersServiceMock) SyncOrders(ctx context.Context, trigger string) (*wooorderservice.SyncSummary, error) {
	if m.syncErr != nil {
		return nil, m.syncErr
	}
	if m.summary != nil {
		return m.summary, nil
	}

	return &wooorderservice.SyncSummary{Trigger: trigger}, nil
}

// SyncOrderByID performs targeted sync behavior.
func (m *ordersServiceMock) SyncOrderByID(ctx context.Context, trigger string, orderID int) (*wooorderservice.SyncSummary, error) {
	m.lastOrderID = orderID
	if m.syncByIDErr != nil {
		return nil, m.syncByIDErr
	}
	if m.syncByIDSummary != nil {
		return m.syncByIDSummary, nil
	}

	return &wooorderservice.SyncSummary{Trigger: trigger, Processed: 1}, nil
}

// couponsServiceMock defines WooCommerce coupons service behavior for handler tests.
type couponsServiceMock struct {
	// summary defines sync summary responses.
	summary *woocouponservice.SyncSummary
	// syncErr defines sync execution errors.
	syncErr error
}

// ValidateIntegration validates integration state.
func (m *couponsServiceMock) ValidateIntegration(ctx context.Context) error {
	return nil
}

// SyncCoupons performs sync behavior.
func (m *couponsServiceMock) SyncCoupons(ctx context.Context, trigger string) (*woocouponservice.SyncSummary, error) {
	if m.syncErr != nil {
		return nil, m.syncErr
	}
	if m.summary != nil {
		return m.summary, nil
	}

	return &woocouponservice.SyncSummary{Trigger: trigger}, nil
}

// authorizerMock defines authorization behavior for handler tests.
type authorizerMock struct {
	// requireErr defines auth errors.
	requireErr error
}

// Require authenticates and authorizes requests.
func (m *authorizerMock) Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error {
	return m.requireErr
}

// IsUnauthorized reports unauthorized errors.
func (m *authorizerMock) IsUnauthorized(err error) bool {
	return errorspkg.Is(err, errUnauthorized)
}

// IsForbidden reports forbidden errors.
func (m *authorizerMock) IsForbidden(err error) bool {
	return errorspkg.Is(err, errForbidden)
}

var (
	// errUnauthorized defines unauthorized test errors.
	errUnauthorized = errorspkg.New("unauthorized")
	// errForbidden defines forbidden test errors.
	errForbidden = errorspkg.New("forbidden")
)

// TestNewHandlerValidation verifies constructor validation behavior.
func TestNewHandlerValidation(t *testing.T) {
	if _, err := NewHandler(nil, &ordersServiceMock{}); !errorspkg.Is(err, ErrNilContactService) {
		t.Fatalf("NewHandler(nil contacts) error = %v, want ErrNilContactService", err)
	}
	if _, err := NewHandler(&contactsServiceMock{}, nil); !errorspkg.Is(err, ErrNilOrderService) {
		t.Fatalf("NewHandler(nil orders) error = %v, want ErrNilOrderService", err)
	}
}

// TestRegisterRoutesAndSync verifies route registration and successful sync behavior.
func TestRegisterRoutesAndSync(t *testing.T) {
	contactsMock := &contactsServiceMock{
		summary: &woocontactservice.SyncSummary{Trigger: "manual", Processed: 2},
	}
	ordersMock := &ordersServiceMock{
		summary: &wooorderservice.SyncSummary{Trigger: "manual", Processed: 3},
	}
	couponsMock := &couponsServiceMock{
		summary: &woocouponservice.SyncSummary{Trigger: "manual", Processed: 4},
	}
	handler, err := NewHandler(contactsMock, ordersMock)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	handler.SetCouponSyncService(couponsMock)

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8121}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(handler.RegisterRoutes)

	request, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/woo/sync/contacts", nil)
	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != stdhttp.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, stdhttp.StatusOK)
	}

	orderRequest, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/woo/sync/orders", nil)
	orderResponse, orderErr := server.App().Test(orderRequest)
	if orderErr != nil {
		t.Fatalf("App().Test() error = %v", orderErr)
	}
	if orderResponse.StatusCode != stdhttp.StatusOK {
		t.Fatalf("status = %d, want %d", orderResponse.StatusCode, stdhttp.StatusOK)
	}

	targetContactRequest, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/woo/sync/contacts?email=target@example.com", nil)
	targetContactResponse, targetContactErr := server.App().Test(targetContactRequest)
	if targetContactErr != nil {
		t.Fatalf("App().Test() error = %v", targetContactErr)
	}
	if targetContactResponse.StatusCode != stdhttp.StatusOK {
		t.Fatalf("status = %d, want %d", targetContactResponse.StatusCode, stdhttp.StatusOK)
	}

	targetOrderRequest, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/woo/sync/orders?id=1001", nil)
	targetOrderResponse, targetOrderErr := server.App().Test(targetOrderRequest)
	if targetOrderErr != nil {
		t.Fatalf("App().Test() error = %v", targetOrderErr)
	}
	if targetOrderResponse.StatusCode != stdhttp.StatusOK {
		t.Fatalf("status = %d, want %d", targetOrderResponse.StatusCode, stdhttp.StatusOK)
	}

	couponRequest, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/woo/sync/coupons", nil)
	couponResponse, couponErr := server.App().Test(couponRequest)
	if couponErr != nil {
		t.Fatalf("App().Test() error = %v", couponErr)
	}
	if couponResponse.StatusCode != stdhttp.StatusOK {
		t.Fatalf("status = %d, want %d", couponResponse.StatusCode, stdhttp.StatusOK)
	}
	if contactsMock.lastEmail != "target@example.com" {
		t.Fatalf("contacts lastEmail = %q, want %q", contactsMock.lastEmail, "target@example.com")
	}
	if ordersMock.lastOrderID != 1001 {
		t.Fatalf("orders lastOrderID = %d, want %d", ordersMock.lastOrderID, 1001)
	}
}

// TestRegisterRoutesWithAuth verifies protected route behavior.
func TestRegisterRoutesWithAuth(t *testing.T) {
	handler, err := NewHandler(&contactsServiceMock{}, &ordersServiceMock{}, &authorizerMock{requireErr: errUnauthorized})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8122}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(handler.RegisterRoutes)

	request, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/woo/sync/contacts", nil)
	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != stdhttp.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.StatusCode, stdhttp.StatusUnauthorized)
	}
}

// TestMapError verifies sync error mapping behavior.
func TestMapError(t *testing.T) {
	handler, err := NewHandler(&contactsServiceMock{}, &ordersServiceMock{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	if appErr := handler.mapError(woocontactservice.ErrSyncDisabled); appErr == nil {
		t.Fatalf("expected mapError(sync disabled)")
	}
	if appErr := handler.mapError(woocontactservice.ErrInvalidEmail); appErr == nil {
		t.Fatalf("expected mapError(invalid email)")
	}
	if appErr := handler.mapError(woocontactservice.ErrContactNotFound); appErr == nil {
		t.Fatalf("expected mapError(contact not found)")
	}
	if appErr := handler.mapError(woocontactservice.ErrIntegrationUnavailable); appErr == nil {
		t.Fatalf("expected mapError(integration unavailable)")
	}
	if appErr := handler.mapError(wooorderservice.ErrSyncDisabled); appErr == nil {
		t.Fatalf("expected mapError(order sync disabled)")
	}
	if appErr := handler.mapError(wooorderservice.ErrInvalidOrderID); appErr == nil {
		t.Fatalf("expected mapError(invalid order id)")
	}
	if appErr := handler.mapError(wooorderservice.ErrOrderNotFound); appErr == nil {
		t.Fatalf("expected mapError(order not found)")
	}
	if appErr := handler.mapError(wooorderservice.ErrIntegrationUnavailable); appErr == nil {
		t.Fatalf("expected mapError(order integration unavailable)")
	}
	if appErr := handler.mapError(woocouponservice.ErrSyncDisabled); appErr == nil {
		t.Fatalf("expected mapError(coupon sync disabled)")
	}
	if appErr := handler.mapError(woocouponservice.ErrIntegrationUnavailable); appErr == nil {
		t.Fatalf("expected mapError(coupon integration unavailable)")
	}
	if appErr := handler.mapError(errorspkg.New("unknown")); appErr == nil {
		t.Fatalf("expected mapError(unknown)")
	}
}

// TestSetAuthorizer verifies optional authorizer wiring behavior.
func TestSetAuthorizer(t *testing.T) {
	handler, err := NewHandler(&contactsServiceMock{}, &ordersServiceMock{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	handler.SetAuthorizer(nil)
}

// TestSyncRoutesInvalidOrderIDQuery verifies invalid targeted order query behavior.
func TestSyncRoutesInvalidOrderIDQuery(t *testing.T) {
	handler, err := NewHandler(&contactsServiceMock{}, &ordersServiceMock{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8123}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(handler.RegisterRoutes)

	request, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/woo/sync/orders?id=abc", nil)
	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != stdhttp.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.StatusCode, stdhttp.StatusBadRequest)
	}
}

// TestSyncCouponsWithoutServiceReturnsServiceUnavailable verifies coupon sync behavior without optional service wiring.
func TestSyncCouponsWithoutServiceReturnsServiceUnavailable(t *testing.T) {
	handler, err := NewHandler(&contactsServiceMock{}, &ordersServiceMock{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8124}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(handler.RegisterRoutes)

	request, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/woo/sync/coupons", nil)
	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != stdhttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.StatusCode, stdhttp.StatusServiceUnavailable)
	}
}
