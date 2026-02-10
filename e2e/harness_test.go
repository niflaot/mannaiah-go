package e2e_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/gorm"
	"mannaiah/module/auth"
	"mannaiah/module/contacts"
	contactevent "mannaiah/module/contacts/adapter/event"
	contactapplication "mannaiah/module/contacts/application"
	coredatabase "mannaiah/module/core/database"
	corehttp "mannaiah/module/core/http"
	coremsgbus "mannaiah/module/core/messaging/bus"
	coremsgplatform "mannaiah/module/core/messaging/platform"
	corewatermill "mannaiah/module/core/messaging/watermill"
)

const (
	// e2eAudience defines JWT audience values used by E2E scenarios.
	e2eAudience = "https://api.mannaiah.e2e"
	// e2eIssuerSuffix defines JWKS suffix paths used by E2E JWKS servers.
	e2eIssuerSuffix = "/jwks"
	// e2eTokenKid defines JWT key identifiers used by E2E token signing.
	e2eTokenKid = "e2e-key"
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
	// createdEvents defines created-event capture channel.
	createdEvents chan contactEventRecord
	// updatedEvents defines updated-event capture channel.
	updatedEvents chan contactEventRecord
	// dbClosed reports whether the database handle was already closed.
	dbClosed bool
}

// newContactsE2EHarness creates a fully wired contacts/auth/event E2E runtime harness.
func newContactsE2EHarness(t *testing.T) *contactsE2EHarness {
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
	db, err := coredatabase.Open(coredatabase.Config{
		Driver:       "sqlite",
		DSN:          "file::memory:?cache=shared",
		MaxOpenConns: 1,
	}, tracer.logger)
	if err != nil {
		t.Fatalf("coredatabase.Open() error = %v", err)
	}

	tracer.Step("initialize in-memory messaging platform")
	messaging, err := corewatermill.NewInMemoryPlatform(coremsgplatform.Config{}, tracer.logger)
	if err != nil {
		t.Fatalf("corewatermill.NewInMemoryPlatform() error = %v", err)
	}

	createdEvents := make(chan contactEventRecord, 4)
	updatedEvents := make(chan contactEventRecord, 4)

	tracer.Step("register event listeners")
	registerContactTopicHandler(t, messaging, contactapplication.TopicContactCreated, createdEvents)
	registerContactTopicHandler(t, messaging, contactapplication.TopicContactUpdated, updatedEvents)

	messagingCtx, messagingCancel := context.WithCancel(context.Background())
	messagingErrs := make(chan error, 1)

	tracer.Step("run messaging router")
	go func() {
		messagingErrs <- messaging.Run(messagingCtx)
	}()

	select {
	case <-messaging.Running():
	case <-time.After(2 * time.Second):
		t.Fatalf("messaging router did not start")
	}

	tracer.Step("initialize contacts module")
	publisher, err := contactevent.NewPublisher(messaging.Publisher())
	if err != nil {
		t.Fatalf("contactevent.NewPublisher() error = %v", err)
	}

	contactsModule, err := contacts.New(db, publisher)
	if err != nil {
		t.Fatalf("contacts.New() error = %v", err)
	}
	contactsModule.SetAuthorizer(authModule)

	tracer.Step("initialize http server")
	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8011}, tracer.logger)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(contactsModule.RegisterRoutes)

	return &contactsE2EHarness{
		tracer:          tracer,
		key:             key,
		jwksServer:      jwksServer,
		authModule:      authModule,
		db:              db,
		messaging:       messaging,
		messagingCancel: messagingCancel,
		messagingErrs:   messagingErrs,
		server:          server,
		createdEvents:   createdEvents,
		updatedEvents:   updatedEvents,
	}
}

// Close releases harness resources.
func (h *contactsE2EHarness) Close(t *testing.T) {
	t.Helper()

	h.tracer.Step("shutdown messaging context")
	h.messagingCancel()

	select {
	case err := <-h.messagingErrs:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("messaging.Run() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("messaging shutdown timeout")
	}

	h.tracer.Step("close messaging platform")
	if err := h.messaging.Close(); err != nil {
		t.Fatalf("messaging.Close() error = %v", err)
	}

	h.tracer.Step("close database handle")
	h.CloseDatabase(t)

	h.tracer.Step("close jwks server")
	h.jwksServer.Close()
}

// CloseDatabase closes the harness database handle and tolerates double-close behavior.
func (h *contactsE2EHarness) CloseDatabase(t *testing.T) {
	t.Helper()

	if h == nil || h.db == nil || h.dbClosed {
		return
	}

	sqlDB, err := h.db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}
	if err := sqlDB.Close(); err != nil && !isClosedDBError(err) {
		t.Fatalf("sqlDB.Close() error = %v", err)
	}

	h.dbClosed = true
}

// DoJSONRequest executes HTTP requests against the in-memory server and decodes JSON responses.
func (h *contactsE2EHarness) DoJSONRequest(t *testing.T, method string, path string, token string, body []byte) (int, map[string]any) {
	t.Helper()

	status, payload, _, err := doJSONRequestRaw(h.server, method, path, token, body)
	if err != nil {
		t.Fatalf("DoJSONRequest() error = %v", err)
	}

	return status, payload
}

// SignToken creates a signed JWT token for E2E requests.
func (h *contactsE2EHarness) SignToken(t *testing.T, scopes string) string {
	t.Helper()

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, jwtlib.MapClaims{
		"sub":   "e2e-user",
		"iss":   strings.TrimSuffix(h.jwksServer.URL, e2eIssuerSuffix),
		"aud":   e2eAudience,
		"scope": scopes,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(10 * time.Minute).Unix(),
	})
	token.Header["kid"] = e2eTokenKid

	signed, err := token.SignedString(h.key)
	if err != nil {
		t.Fatalf("token.SignedString() error = %v", err)
	}

	return signed
}

// AwaitCreatedEvent waits for a created-contact integration event.
func (h *contactsE2EHarness) AwaitCreatedEvent(t *testing.T) contactEventRecord {
	t.Helper()

	return awaitEventRecord(t, h.createdEvents, "contacts.v1.created")
}

// AwaitUpdatedEvent waits for an updated-contact integration event.
func (h *contactsE2EHarness) AwaitUpdatedEvent(t *testing.T) contactEventRecord {
	t.Helper()

	return awaitEventRecord(t, h.updatedEvents, "contacts.v1.updated")
}

// registerContactTopicHandler registers event listeners for a topic and pushes decoded events to a channel.
func registerContactTopicHandler(t *testing.T, messaging *corewatermill.InMemoryPlatform, topic string, sink chan<- contactEventRecord) {
	t.Helper()

	err := messaging.Registrar().AddHandler(topic, func(ctx context.Context, msg coremsgbus.Message) error {
		payload := map[string]any{}
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return err
		}

		sink <- contactEventRecord{
			Topic:    msg.Topic,
			Payload:  payload,
			Metadata: msg.Metadata,
		}

		return nil
	})
	if err != nil {
		t.Fatalf("Registrar().AddHandler(%q) error = %v", topic, err)
	}
}

// awaitEventRecord waits for an event record on the provided channel.
func awaitEventRecord(t *testing.T, source <-chan contactEventRecord, expectedTopic string) contactEventRecord {
	t.Helper()

	select {
	case event := <-source:
		if event.Topic != expectedTopic {
			t.Fatalf("event.Topic = %q, want %q", event.Topic, expectedTopic)
		}
		return event
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting for topic %q", expectedTopic)
		return contactEventRecord{}
	}
}

// newJWKSServer creates an HTTP JWKS endpoint server for token verification tests.
func newJWKSServer(t *testing.T, publicKey rsa.PublicKey) *httptest.Server {
	t.Helper()

	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != e2eIssuerSuffix {
			writer.WriteHeader(http.StatusNotFound)
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"keys": []any{map[string]any{
				"kty": "RSA",
				"kid": e2eTokenKid,
				"alg": "RS256",
				"use": "sig",
				"n":   encodeBigInt(publicKey.N),
				"e":   encodeBigInt(big.NewInt(int64(publicKey.E))),
			}},
		})
	})

	return httptest.NewServer(handler)
}

// encodeBigInt encodes big-int values into base64url strings.
func encodeBigInt(value *big.Int) string {
	if value == nil {
		return ""
	}

	return base64.RawURLEncoding.EncodeToString(value.Bytes())
}

// doJSONRequestRaw executes HTTP requests and decodes JSON responses without testing-side failures.
func doJSONRequestRaw(server *corehttp.Server, method string, path string, token string, body []byte) (int, map[string]any, http.Header, error) {
	requestBody := bytes.NewReader(body)
	request, err := http.NewRequest(method, path, requestBody)
	if err != nil {
		return 0, nil, nil, err
	}
	if len(body) > 0 {
		request.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(token) != "" {
		request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}

	response, err := server.App().Test(request)
	if err != nil {
		return 0, nil, nil, err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	result := map[string]any{}
	if response.ContentLength != 0 {
		payload, readErr := io.ReadAll(response.Body)
		if readErr != nil {
			return 0, nil, nil, readErr
		}
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &result); err != nil {
				return 0, nil, nil, err
			}
		}
	}

	return response.StatusCode, result, response.Header, nil
}

// isClosedDBError reports whether a DB close failure is caused by an already-closed handle.
func isClosedDBError(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(strings.ToLower(strings.TrimSpace(err.Error())), "closed")
}
