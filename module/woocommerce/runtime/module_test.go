package runtime

import (
	"context"
	errorspkg "errors"
	stdhttp "net/http"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"go.uber.org/zap"
	contactapplication "mannaiah/module/contacts/application"
	contactdomain "mannaiah/module/contacts/domain"
	contactport "mannaiah/module/contacts/port"
	corecron "mannaiah/module/core/cron"
	corehttp "mannaiah/module/core/http"
	"mannaiah/module/core/messaging/bus"
	ordersapplication "mannaiah/module/orders/application"
	ordersdomain "mannaiah/module/orders/domain"
)

// contactServiceMock defines contacts service behavior for module tests.
type contactServiceMock struct{}

// Create creates contacts.
func (contactServiceMock) Create(ctx context.Context, command contactapplication.CreateCommand) (*contactdomain.Contact, error) {
	return &contactdomain.Contact{ID: "created", Email: command.Email}, nil
}

// Get retrieves contacts by id.
func (contactServiceMock) Get(ctx context.Context, id string) (*contactdomain.Contact, error) {
	return nil, nil
}

// List retrieves contact pages.
func (contactServiceMock) List(ctx context.Context, query contactport.ListQuery) (*contactapplication.ListResult, error) {
	return &contactapplication.ListResult{}, nil
}

// Update updates contacts.
func (contactServiceMock) Update(ctx context.Context, id string, command contactapplication.UpdateCommand) (*contactdomain.Contact, error) {
	return &contactdomain.Contact{ID: id}, nil
}

// Delete deletes contacts.
func (contactServiceMock) Delete(ctx context.Context, id string) error {
	return nil
}

// orderServiceMock defines orders service behavior for module tests.
type orderServiceMock struct{}

// Create creates orders.
func (orderServiceMock) Create(ctx context.Context, command ordersapplication.CreateCommand) (*ordersdomain.Order, error) {
	return &ordersdomain.Order{ID: "created", Identifier: command.Identifier, Realm: command.Realm}, nil
}

// Get retrieves orders by id.
func (orderServiceMock) Get(ctx context.Context, id string) (*ordersdomain.Order, error) {
	return &ordersdomain.Order{ID: id}, nil
}

// List lists order pages.
func (orderServiceMock) List(ctx context.Context, query ordersapplication.ListQuery) (*ordersapplication.ListResult, error) {
	return &ordersapplication.ListResult{}, nil
}

// Update updates mutable order rows.
func (orderServiceMock) Update(ctx context.Context, id string, command ordersapplication.UpdateCommand) (*ordersdomain.Order, error) {
	return &ordersdomain.Order{ID: id}, nil
}

// UpdateStatus updates order status rows.
func (orderServiceMock) UpdateStatus(ctx context.Context, id string, command ordersapplication.UpdateStatusCommand) (*ordersdomain.Order, error) {
	return &ordersdomain.Order{ID: id, CurrentStatus: command.Status}, nil
}

// AddComment appends order comment rows.
func (orderServiceMock) AddComment(ctx context.Context, id string, command ordersapplication.AddCommentCommand) (*ordersdomain.Order, error) {
	return &ordersdomain.Order{ID: id}, nil
}

// UpdateComment updates order comment rows.
func (orderServiceMock) UpdateComment(ctx context.Context, id string, commentID string, command ordersapplication.UpdateCommentCommand) (*ordersdomain.Order, error) {
	return &ordersdomain.Order{ID: id}, nil
}

// DeleteComment deletes order comment rows.
func (orderServiceMock) DeleteComment(ctx context.Context, id string, commentID string, command ordersapplication.DeleteCommentCommand) (*ordersdomain.Order, error) {
	return &ordersdomain.Order{ID: id}, nil
}

// schedulerMock defines scheduler behavior for module tests.
type schedulerMock struct {
	// addErr defines add-operation errors.
	addErr error
	// stopErr defines stop-operation errors.
	stopErr error
	// addedSpecs defines added spec values.
	addedSpecs []string
	// addCalled reports add-operation calls.
	addCalled bool
	// removeCalled reports remove-operation calls.
	removeCalled bool
	// startCalled reports start-operation calls.
	startCalled bool
	// stopCalled reports stop-operation calls.
	stopCalled bool
}

// registrarMock defines message-registrar behavior for module tests.
type registrarMock struct {
	// topics defines registered topic values.
	topics []string
}

// AddHandler stores registered topic values.
func (m *registrarMock) AddHandler(topic string, handler bus.Handler) error {
	m.topics = append(m.topics, topic)
	return nil
}

// Add registers jobs.
func (m *schedulerMock) Add(spec string, job corecron.Job) (corecron.EntryID, error) {
	if m.addErr != nil {
		return 0, m.addErr
	}
	m.addCalled = true
	m.addedSpecs = append(m.addedSpecs, spec)
	return corecron.EntryID(len(m.addedSpecs)), nil
}

// AddFunc registers function jobs.
func (m *schedulerMock) AddFunc(spec string, job func()) (corecron.EntryID, error) {
	return m.Add(spec, corecron.JobFunc(job))
}

// Remove removes jobs.
func (m *schedulerMock) Remove(id corecron.EntryID) {
	m.removeCalled = true
}

// Entries lists jobs.
func (m *schedulerMock) Entries() []corecron.Entry {
	return nil
}

// Start starts scheduling.
func (m *schedulerMock) Start() {
	m.startCalled = true
}

// Run runs scheduling.
func (m *schedulerMock) Run() {}

// Stop stops scheduling.
func (m *schedulerMock) Stop(ctx context.Context) error {
	m.stopCalled = true
	return m.stopErr
}

// loaderProbe defines startup loader behavior for module tests.
type loaderProbe struct {
	// registered indicates whether routes were registered.
	registered bool
	// specAdded indicates whether OpenAPI specs were added.
	specAdded bool
}

// RegisterRoutes captures route registration calls.
func (l *loaderProbe) RegisterRoutes(register func(router corehttp.Router)) {
	l.registered = true
}

// AddOpenAPISpec captures OpenAPI spec merge calls.
func (l *loaderProbe) AddOpenAPISpec(spec *openapi3.T) error {
	l.specAdded = spec != nil
	return nil
}

// authorizerMock defines authorizer behavior for module tests.
type authorizerMock struct{}

// Require authenticates requests.
func (authorizerMock) Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error {
	return nil
}

// IsUnauthorized reports auth failures.
func (authorizerMock) IsUnauthorized(err error) bool {
	return false
}

// IsForbidden reports authorization failures.
func (authorizerMock) IsForbidden(err error) bool {
	return false
}

// TestNewValidation verifies constructor validation behavior.
func TestNewValidation(t *testing.T) {
	if _, err := New(Config{}, nil, orderServiceMock{}, nil, nil, nil); !errorspkg.Is(err, ErrNilContactService) {
		t.Fatalf("New(nil service) error = %v, want ErrNilContactService", err)
	}
	if _, err := New(Config{}, contactServiceMock{}, nil, nil, nil, nil); !errorspkg.Is(err, ErrNilOrderService) {
		t.Fatalf("New(nil order service) error = %v, want ErrNilOrderService", err)
	}
	if _, err := New(Config{SyncContacts: true}, contactServiceMock{}, orderServiceMock{}, nil, nil, nil); !errorspkg.Is(err, ErrNilSchedulerWhenEnabled) {
		t.Fatalf("New(sync enabled nil scheduler) error = %v, want ErrNilSchedulerWhenEnabled", err)
	}
}

// TestLoadRegisterRoutes verifies module route/spec registration behavior.
func TestLoadRegisterRoutes(t *testing.T) {
	module, err := New(Config{}, contactServiceMock{}, orderServiceMock{}, nil, zap.NewNop(), nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	probe := &loaderProbe{}
	if err := module.Load(probe); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !probe.registered {
		t.Fatalf("expected route registration")
	}
	if !probe.specAdded {
		t.Fatalf("expected spec merge")
	}
}

// TestNewWithRegistrarRegistersOrderConsumers verifies cross-module order consumer registration behavior.
func TestNewWithRegistrarRegistersOrderConsumers(t *testing.T) {
	registrar := &registrarMock{}
	module, err := New(Config{
		URL:            "https://example.com",
		ConsumerKey:    "key",
		ConsumerSecret: "secret",
	}, contactServiceMock{}, orderServiceMock{}, nil, zap.NewNop(), registrar)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if module.orderEventConsumer == nil {
		t.Fatalf("expected orderEventConsumer to be configured")
	}
	if len(registrar.topics) != 3 {
		t.Fatalf("len(registrar.topics) = %d, want %d", len(registrar.topics), 3)
	}
}

// TestRegisterRoutesServer verifies endpoint registration behavior.
func TestRegisterRoutesServer(t *testing.T) {
	module, err := New(Config{}, contactServiceMock{}, orderServiceMock{}, nil, zap.NewNop(), nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8123}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(module.RegisterRoutes)

	req, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/woo/sync/contacts", nil)
	resp, testErr := server.App().Test(req)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if resp.StatusCode != stdhttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", resp.StatusCode, stdhttp.StatusServiceUnavailable)
	}

	orderReq, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/woo/sync/orders", nil)
	orderResp, orderTestErr := server.App().Test(orderReq)
	if orderTestErr != nil {
		t.Fatalf("App().Test() error = %v", orderTestErr)
	}
	if orderResp.StatusCode != stdhttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", orderResp.StatusCode, stdhttp.StatusServiceUnavailable)
	}
}

// TestSetAuthorizer verifies optional authorizer wiring behavior.
func TestSetAuthorizer(t *testing.T) {
	module, err := New(Config{}, contactServiceMock{}, orderServiceMock{}, nil, zap.NewNop(), nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	module.SetAuthorizer(authorizerMock{})
}

// TestStartStopWithScheduler verifies scheduler startup and shutdown behavior.
func TestStartStopWithScheduler(t *testing.T) {
	scheduler := &schedulerMock{}
	module, err := New(Config{
		SyncContacts:     true,
		SyncContactsCron: "0 0 * * *",
		URL:              "https://example.com",
		ConsumerKey:      "key",
		ConsumerSecret:   "secret",
	}, contactServiceMock{}, orderServiceMock{}, scheduler, zap.NewNop(), nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if !scheduler.addCalled || len(scheduler.addedSpecs) != 1 || scheduler.addedSpecs[0] != "0 0 * * *" {
		t.Fatalf("expected scheduler AddFunc call with cron spec")
	}
	if !scheduler.startCalled {
		t.Fatalf("expected scheduler Start() call")
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := module.Stop(stopCtx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if !scheduler.removeCalled {
		t.Fatalf("expected scheduler Remove() call")
	}
	if !scheduler.stopCalled {
		t.Fatalf("expected scheduler Stop() call")
	}
}

// TestStartStopNilModule verifies nil-module behavior.
func TestStartStopNilModule(t *testing.T) {
	var module *Module
	if err := module.Start(context.Background()); !errorspkg.Is(err, ErrModuleNotInitialized) {
		t.Fatalf("Start(nil) error = %v, want ErrModuleNotInitialized", err)
	}
	if err := module.Stop(context.Background()); err != nil {
		t.Fatalf("Stop(nil) error = %v", err)
	}
}

// TestStartSchedulerError verifies scheduler add failures.
func TestStartSchedulerError(t *testing.T) {
	scheduler := &schedulerMock{addErr: errorspkg.New("add failed")}
	module, err := New(Config{
		SyncContacts:     true,
		SyncContactsCron: "0 0 * * *",
		URL:              "https://example.com",
		ConsumerKey:      "key",
		ConsumerSecret:   "secret",
	}, contactServiceMock{}, orderServiceMock{}, scheduler, zap.NewNop(), nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := module.Start(context.Background()); err == nil {
		t.Fatalf("expected Start() error")
	}
}

// TestStopSchedulerError verifies scheduler stop failures.
func TestStopSchedulerError(t *testing.T) {
	scheduler := &schedulerMock{stopErr: errorspkg.New("stop failed")}
	module, err := New(Config{
		SyncContacts:     true,
		SyncContactsCron: "0 0 * * *",
		URL:              "https://example.com",
		ConsumerKey:      "key",
		ConsumerSecret:   "secret",
	}, contactServiceMock{}, orderServiceMock{}, scheduler, zap.NewNop(), nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := module.Stop(context.Background()); err == nil {
		t.Fatalf("expected Stop() error")
	}
}

// TestStartStopWithBothSchedulers verifies scheduler registration behavior for contact and order sync jobs.
func TestStartStopWithBothSchedulers(t *testing.T) {
	scheduler := &schedulerMock{}
	module, err := New(Config{
		SyncContacts:     true,
		SyncContactsCron: "0 0 * * *",
		SyncOrders:       true,
		SyncOrdersCron:   "0 1 * * *",
		URL:              "https://example.com",
		ConsumerKey:      "key",
		ConsumerSecret:   "secret",
	}, contactServiceMock{}, orderServiceMock{}, scheduler, zap.NewNop(), nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if len(scheduler.addedSpecs) != 2 {
		t.Fatalf("len(addedSpecs) = %d, want 2", len(scheduler.addedSpecs))
	}
	if scheduler.addedSpecs[0] != "0 0 * * *" || scheduler.addedSpecs[1] != "0 1 * * *" {
		t.Fatalf("addedSpecs = %v, want [0 0 * * * 0 1 * * *]", scheduler.addedSpecs)
	}

	if err := module.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

// TestResolveHelpers verifies private helper behavior.
func TestResolveHelpers(t *testing.T) {
	if resolveRequestTimeout(0) != 5000 {
		t.Fatalf("resolveRequestTimeout(0) should fallback to 5000")
	}
	if resolveValidationTimeout(0) != 3*time.Second {
		t.Fatalf("resolveValidationTimeout(0) should fallback to 3s")
	}
	if resolveSyncTimeout(0) != 10*time.Minute {
		t.Fatalf("resolveSyncTimeout(0) should fallback to 10m")
	}
	if resolveContext(nil) == nil {
		t.Fatalf("resolveContext(nil) should return background context")
	}
	if newSourceCircuitBreaker(Config{CircuitBreakerEnabled: false}, zap.NewNop()) != nil {
		t.Fatalf("newSourceCircuitBreaker() should return nil when disabled")
	}
	if newSourceCircuitBreaker(Config{CircuitBreakerEnabled: true}, zap.NewNop()) == nil {
		t.Fatalf("newSourceCircuitBreaker() should return breaker when enabled")
	}
}
