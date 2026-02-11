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

// schedulerMock defines scheduler behavior for module tests.
type schedulerMock struct {
	// addErr defines add-operation errors.
	addErr error
	// stopErr defines stop-operation errors.
	stopErr error
	// addedSpec defines added spec values.
	addedSpec string
	// addCalled reports add-operation calls.
	addCalled bool
	// removeCalled reports remove-operation calls.
	removeCalled bool
	// startCalled reports start-operation calls.
	startCalled bool
	// stopCalled reports stop-operation calls.
	stopCalled bool
}

// Add registers jobs.
func (m *schedulerMock) Add(spec string, job corecron.Job) (corecron.EntryID, error) {
	if m.addErr != nil {
		return 0, m.addErr
	}
	m.addCalled = true
	m.addedSpec = spec
	return corecron.EntryID(1), nil
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
	if _, err := New(Config{}, nil, nil, nil); !errorspkg.Is(err, ErrNilContactService) {
		t.Fatalf("New(nil service) error = %v, want ErrNilContactService", err)
	}
	if _, err := New(Config{SyncContacts: true}, contactServiceMock{}, nil, nil); !errorspkg.Is(err, ErrNilSchedulerWhenEnabled) {
		t.Fatalf("New(sync enabled nil scheduler) error = %v, want ErrNilSchedulerWhenEnabled", err)
	}
}

// TestLoadRegisterRoutes verifies module route/spec registration behavior.
func TestLoadRegisterRoutes(t *testing.T) {
	module, err := New(Config{}, contactServiceMock{}, nil, zap.NewNop())
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

// TestRegisterRoutesServer verifies endpoint registration behavior.
func TestRegisterRoutesServer(t *testing.T) {
	module, err := New(Config{}, contactServiceMock{}, nil, zap.NewNop())
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
}

// TestSetAuthorizer verifies optional authorizer wiring behavior.
func TestSetAuthorizer(t *testing.T) {
	module, err := New(Config{}, contactServiceMock{}, nil, zap.NewNop())
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
	}, contactServiceMock{}, scheduler, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if !scheduler.addCalled || scheduler.addedSpec != "0 0 * * *" {
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
	}, contactServiceMock{}, scheduler, zap.NewNop())
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
	}, contactServiceMock{}, scheduler, zap.NewNop())
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

// TestResolveHelpers verifies private helper behavior.
func TestResolveHelpers(t *testing.T) {
	if resolveRequestTimeout(0) != 5000 {
		t.Fatalf("resolveRequestTimeout(0) should fallback to 5000")
	}
	if resolveValidationTimeout(0) != 3*time.Second {
		t.Fatalf("resolveValidationTimeout(0) should fallback to 3s")
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
