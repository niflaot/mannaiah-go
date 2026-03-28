package e2e_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"mannaiah/module/auth"
	"mannaiah/module/contacts"
	contactport "mannaiah/module/contacts/port"
	coredatabase "mannaiah/module/core/database"
	coredatabasemigration "mannaiah/module/core/database/migration"
	corehttp "mannaiah/module/core/http"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// failingIntegrationEventPublisher defines deterministic event publication failures for resilience tests.
type failingIntegrationEventPublisher struct{}

// Publish always returns a transport failure for resilience testing.
func (failingIntegrationEventPublisher) Publish(ctx context.Context, event contactport.IntegrationEvent) error {
	return errors.New("integration publisher unavailable")
}

// TestContactsAuthJWKSUnavailableE2E verifies authentication behavior when JWKS endpoints are unreachable.
func TestContactsAuthJWKSUnavailableE2E(t *testing.T) {
	tracer := newStepTracer(t)

	tracer.Step("generate jwt signing key")
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey() error = %v", err)
	}

	tracer.Step("initialize auth module with unreachable jwks endpoint")
	issuer := "http://127.0.0.1:1"
	authModule, err := auth.New(auth.Config{
		Issuer:                 issuer,
		Audience:               e2eAudience,
		JWKSRateLimitPerMinute: 5,
		JWKSCacheTTLMS:         60000,
		JWKSHTTPTimeoutMS:      150,
	}, "production", tracer.logger)
	if err != nil {
		t.Fatalf("auth.New() error = %v", err)
	}

	tracer.Step("open sqlite database")
	db := newE2EDatabase(t, tracer)
	defer closeE2EDatabase(t, db)

	tracer.Step("initialize contacts module")
	contactsModule, err := contacts.New(db)
	if err != nil {
		t.Fatalf("contacts.New() error = %v", err)
	}
	contactsModule.SetAuthorizer(authModule)

	tracer.Step("initialize http server")
	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8012}, tracer.logger)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(contactsModule.RegisterRoutes)

	tracer.Step("sign token against unreachable issuer")
	token := signTokenForIssuer(t, key, issuer, e2eAudience, "contact:manage", "unreachable-kid")

	tracer.Step("request protected endpoint and expect unauthorized")
	status, payload, headers, err := doJSONRequestRaw(server, http.MethodPost, "/contacts", token, []byte(`{"email":"jwks-unavailable@example.com","legalName":"JWKS Unavailable"}`))
	if err != nil {
		t.Fatalf("doJSONRequestRaw() error = %v", err)
	}
	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", status, http.StatusUnauthorized)
	}
	if payload["message"] != "unauthorized" {
		t.Fatalf("payload.message = %v, want %q", payload["message"], "unauthorized")
	}
	if strings.TrimSpace(headers.Get(corehttp.HeaderRayID)) == "" {
		t.Fatalf("expected %s response header", corehttp.HeaderRayID)
	}

	tracer.Step("assert resilience trace logs")
	tracer.AssertStepCount(7)
}

// TestContactsDBConnectionFailureE2E verifies request behavior when database connections become unavailable.
func TestContactsDBConnectionFailureE2E(t *testing.T) {
	harness := newContactsE2EHarness(t)
	defer harness.Close(t)

	readToken := harness.SignToken(t, "contact:view")

	harness.tracer.Step("close database handle to simulate outage")
	harness.CloseDatabase(t)

	harness.tracer.Step("request contacts list after database outage")
	status, payload, headers, err := doJSONRequestRaw(harness.server, http.MethodGet, "/contacts?page=1&limit=10", readToken, nil)
	if err != nil {
		t.Fatalf("doJSONRequestRaw() error = %v", err)
	}
	if status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", status, http.StatusInternalServerError)
	}
	if payload["message"] != "internal_server_error" {
		t.Fatalf("payload.message = %v, want %q", payload["message"], "internal_server_error")
	}
	if strings.TrimSpace(headers.Get(corehttp.HeaderRayID)) == "" {
		t.Fatalf("expected %s response header", corehttp.HeaderRayID)
	}

	harness.tracer.Step("assert resilience trace logs")
	harness.tracer.AssertStepCount(12)
}

// TestContactsPublisherFailureE2E verifies create-flow behavior when integration event transport is unavailable.
func TestContactsPublisherFailureE2E(t *testing.T) {
	tracer, key, jwksServer, db, server := newContactsRuntimeWithPublisher(t, failingIntegrationEventPublisher{})
	defer jwksServer.Close()
	defer closeE2EDatabase(t, db)

	manageToken := signTokenForIssuer(t, key, strings.TrimSuffix(jwksServer.URL, e2eIssuerSuffix), e2eAudience, "contact:manage", e2eTokenKid)
	readToken := signTokenForIssuer(t, key, strings.TrimSuffix(jwksServer.URL, e2eIssuerSuffix), e2eAudience, "contact:view", e2eTokenKid)

	tracer.Step("request create endpoint with failing integration publisher")
	status, payload, headers, err := doJSONRequestRaw(server, http.MethodPost, "/contacts", manageToken, []byte(`{"email":"publisher-failure@example.com","legalName":"Publisher Failure"}`))
	if err != nil {
		t.Fatalf("doJSONRequestRaw() error = %v", err)
	}
	if status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", status, http.StatusInternalServerError)
	}
	if payload["message"] != "internal_server_error" {
		t.Fatalf("payload.message = %v, want %q", payload["message"], "internal_server_error")
	}
	if strings.TrimSpace(headers.Get(corehttp.HeaderRayID)) == "" {
		t.Fatalf("expected %s response header", corehttp.HeaderRayID)
	}

	tracer.Step("verify contact persistence outcome remains queryable")
	status, payload, _, err = doJSONRequestRaw(server, http.MethodGet, "/contacts?page=1&limit=10", readToken, nil)
	if err != nil {
		t.Fatalf("doJSONRequestRaw() error = %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}

	meta, ok := payload["meta"].(map[string]any)
	if !ok {
		t.Fatalf("expected list meta payload")
	}
	if meta["total"] != float64(1) {
		t.Fatalf("meta.total = %v, want %v", meta["total"], float64(1))
	}

	tracer.Step("assert resilience trace logs")
	tracer.AssertStepCount(9)
}

// TestContactsConcurrentCreateConflictE2E verifies concurrent duplicate writes produce one success and conflicts.
func TestContactsConcurrentCreateConflictE2E(t *testing.T) {
	harness := newContactsE2EHarness(t)
	defer harness.Close(t)

	manageToken := harness.SignToken(t, "contact:manage")
	readToken := harness.SignToken(t, "contact:view")

	harness.tracer.Step("run concurrent create requests with same unique fields")

	const workers = 8
	requestBody := []byte(`{"email":"race@example.com","legalName":"Race Contact","documentType":"CC","documentNumber":"999"}`)
	statuses := make(chan int, workers)
	errorsCh := make(chan error, workers)

	var waitGroup sync.WaitGroup
	waitGroup.Add(workers)
	for index := 0; index < workers; index++ {
		go func() {
			defer waitGroup.Done()

			status, _, _, err := doJSONRequestRaw(harness.server, http.MethodPost, "/contacts", manageToken, requestBody)
			if err != nil {
				errorsCh <- err
				return
			}
			if status != http.StatusCreated && status != http.StatusConflict {
				errorsCh <- fmt.Errorf("unexpected status %d", status)
				return
			}

			statuses <- status
		}()
	}
	waitGroup.Wait()
	close(statuses)
	close(errorsCh)

	for err := range errorsCh {
		t.Fatalf("concurrent request error = %v", err)
	}

	createdCount := 0
	conflictCount := 0
	for status := range statuses {
		if status == http.StatusCreated {
			createdCount++
			continue
		}
		if status == http.StatusConflict {
			conflictCount++
		}
	}

	if createdCount != 1 {
		t.Fatalf("createdCount = %d, want %d", createdCount, 1)
	}
	if conflictCount != workers-1 {
		t.Fatalf("conflictCount = %d, want %d", conflictCount, workers-1)
	}

	harness.tracer.Step("verify total contacts after race")
	status, payload := harness.DoJSONRequest(t, http.MethodGet, "/contacts?page=1&limit=10", readToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}

	meta, ok := payload["meta"].(map[string]any)
	if !ok {
		t.Fatalf("expected list meta payload")
	}
	if meta["total"] != float64(1) {
		t.Fatalf("meta.total = %v, want %v", meta["total"], float64(1))
	}

	harness.tracer.Step("assert race trace logs")
	harness.tracer.AssertStepCount(12)
}

// newContactsRuntimeWithPublisher creates an auth-protected contacts runtime with a configurable event publisher.
func newContactsRuntimeWithPublisher(t *testing.T, publisher contactport.IntegrationEventPublisher) (*stepTracer, *rsa.PrivateKey, *httptest.Server, *gorm.DB, *corehttp.Server) {
	t.Helper()

	tracer := newStepTracer(t)

	tracer.Step("generate jwt signing key")
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey() error = %v", err)
	}

	tracer.Step("start jwks server")
	jwksServer := newJWKSServer(t, key.PublicKey)

	tracer.Step("initialize auth module")
	authModule, err := auth.New(auth.Config{
		Issuer:                 strings.TrimSuffix(jwksServer.URL, e2eIssuerSuffix),
		Audience:               e2eAudience,
		JWKSRateLimitPerMinute: 5,
		JWKSCacheTTLMS:         60000,
		JWKSHTTPTimeoutMS:      2000,
	}, "production", tracer.logger)
	if err != nil {
		t.Fatalf("auth.New() error = %v", err)
	}

	tracer.Step("open sqlite database")
	db := newE2EDatabase(t, tracer)

	tracer.Step("initialize contacts module")
	contactsModule, err := contacts.New(db, publisher)
	if err != nil {
		t.Fatalf("contacts.New() error = %v", err)
	}
	contactsModule.SetAuthorizer(authModule)

	tracer.Step("initialize http server")
	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8013}, tracer.logger)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(contactsModule.RegisterRoutes)

	return tracer, key, jwksServer, db, server
}

// newE2EDatabase opens a sqlite database for E2E scenarios.
func newE2EDatabase(t *testing.T, tracer *stepTracer) *gorm.DB {
	t.Helper()

	db, err := coredatabase.Open(coredatabase.Config{
		Driver:       "sqlite",
		DSN:          "file::memory:?cache=shared",
		MaxOpenConns: 1,
	}, tracer.logger)
	if err != nil {
		t.Fatalf("coredatabase.Open() error = %v", err)
	}
	if err := coredatabasemigration.Apply(context.Background(), db, coredatabasemigration.Config{Enabled: true, Driver: "sqlite", Table: "schema_migrations"}, tracer.logger); err != nil {
		t.Fatalf("coredatabasemigration.Apply() error = %v", err)
	}

	return db
}

// closeE2EDatabase closes the sql database handle.
func closeE2EDatabase(t *testing.T, db *gorm.DB) {
	t.Helper()

	if db == nil {
		return
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}
	if err := sqlDB.Close(); err != nil && !isClosedDBError(err) {
		t.Fatalf("sqlDB.Close() error = %v", err)
	}
}

// signTokenForIssuer signs JWT tokens for a specific issuer/audience/scope tuple.
func signTokenForIssuer(t *testing.T, key *rsa.PrivateKey, issuer string, audience string, scope string, kid string) string {
	t.Helper()

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, jwtlib.MapClaims{
		"sub":   "e2e-user",
		"iss":   strings.TrimSpace(issuer),
		"aud":   strings.TrimSpace(audience),
		"scope": strings.TrimSpace(scope),
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(10 * time.Minute).Unix(),
	})
	token.Header["kid"] = strings.TrimSpace(kid)

	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("token.SignedString() error = %v", err)
	}

	return signed
}
