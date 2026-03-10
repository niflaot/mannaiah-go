package e2e_test

import (
	"context"
	"crypto/rsa"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/gorm"
	"mannaiah/module/assets"
	"mannaiah/module/auth"
	"mannaiah/module/contacts"
	corehttp "mannaiah/module/core/http"
	corewatermill "mannaiah/module/core/messaging/watermill"
	"mannaiah/module/orders"
	"mannaiah/module/products"
)

const (
	// e2eAudience defines JWT audience values used by E2E scenarios.
	e2eAudience = "https://api.mannaiah.e2e"
	// e2eIssuerSuffix defines JWKS suffix paths used by E2E JWKS servers.
	e2eIssuerSuffix = "/jwks"
	// e2eTokenKid defines JWT key identifiers used by E2E token signing.
	e2eTokenKid = "e2e-key"
	// harnessEventBufferSize defines event sink capacity to avoid blocking high-volume producers.
	harnessEventBufferSize = 512
)

// stepTracer defines structured step-level tracing behavior for E2E scenarios.
type stepTracer struct {
	// t defines testing runtime dependency.
	t *testing.T
	// logger defines step-level structured logger dependency.
	logger *zap.Logger
	// observed stores emitted logs for test assertions.
	observed *observer.ObservedLogs
	// stepCounter tracks deterministic step ordering.
	stepCounter int64
}

// newStepTracer creates a structured step tracer for E2E scenarios.
func newStepTracer(t *testing.T) *stepTracer {
	t.Helper()

	observedCore, observed := observer.New(zap.DebugLevel)
	logger := zap.New(observedCore)

	return &stepTracer{
		t:        t,
		logger:   logger,
		observed: observed,
	}
}

// Step logs a test step using structured Zap fields.
func (s *stepTracer) Step(name string, fields ...zap.Field) {
	s.t.Helper()

	step := atomic.AddInt64(&s.stepCounter, 1)
	baseFields := []zap.Field{
		zap.Int64("step", step),
		zap.String("name", name),
	}
	baseFields = append(baseFields, fields...)

	s.logger.Info("e2e-step", baseFields...)
	s.t.Logf("step %d: %s", step, name)
}

// AssertStepCount verifies the total number of emitted step logs.
func (s *stepTracer) AssertStepCount(minimum int) {
	s.t.Helper()

	if s.observed.Len() < minimum {
		s.t.Fatalf("step logs = %d, want at least %d", s.observed.Len(), minimum)
	}
}

// contactEventRecord defines captured integration event values for assertions.
type contactEventRecord struct {
	// Topic defines captured event topic names.
	Topic string
	// Payload defines decoded JSON event payload values.
	Payload map[string]any
	// Metadata defines transport metadata values.
	Metadata map[string]string
}

// contactsE2EHarness defines reusable runtime dependencies for contacts/auth/event E2E scenarios.
type contactsE2EHarness struct {
	// tracer defines test step tracer dependency.
	tracer *stepTracer
	// key defines RSA signing key used by JWT generation.
	key *rsa.PrivateKey
	// jwksServer defines JWKS endpoint server dependency.
	jwksServer *httptest.Server
	// authModule defines auth runtime dependency.
	authModule *auth.Module
	// db defines database runtime dependency.
	db *gorm.DB
	// messaging defines messaging runtime dependency.
	messaging *corewatermill.InMemoryPlatform
	// messagingCancel defines messaging context cancellation behavior.
	messagingCancel context.CancelFunc
	// messagingErrs receives messaging runtime errors.
	messagingErrs chan error
	// server defines HTTP runtime dependency.
	server *corehttp.Server
	// contactsModule defines contacts module runtime dependency.
	contactsModule *contacts.Module
	// assetsModule defines assets module runtime dependency.
	assetsModule *assets.Module
	// assetStorage defines in-memory asset storage dependency.
	assetStorage *inMemoryAssetStorage
	// productsModule defines products module runtime dependency.
	productsModule *products.Module
	// ordersModule defines orders module runtime dependency.
	ordersModule *orders.Module
	// createdEvents defines created-event capture channel.
	createdEvents chan contactEventRecord
	// updatedEvents defines updated-event capture channel.
	updatedEvents chan contactEventRecord
	// assetCreatedEvents defines asset-created event capture channel.
	assetCreatedEvents chan contactEventRecord
	// assetUpdatedEvents defines asset-updated event capture channel.
	assetUpdatedEvents chan contactEventRecord
	// assetDeletedEvents defines asset-deleted event capture channel.
	assetDeletedEvents chan contactEventRecord
	// dbClosed reports whether the database handle was already closed.
	dbClosed bool
}
