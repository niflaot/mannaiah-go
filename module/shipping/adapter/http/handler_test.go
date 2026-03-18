package http

import (
	"bytes"
	"context"
	"errors"
	stdhttp "net/http"
	"testing"

	corehttp "mannaiah/module/core/http"
	"mannaiah/module/shipping/domain"
)

// serviceMock defines quote service behavior for handler tests.
type serviceMock struct {
	// result defines successful quote values.
	result *domain.QuoteResult
	// err defines quote errors.
	err error
	// request captures quote request values.
	request domain.QuoteRequest
}

// Quote returns configured quote result values.
func (m *serviceMock) Quote(ctx context.Context, request domain.QuoteRequest) (*domain.QuoteResult, error) {
	m.request = request
	if m.err != nil {
		return nil, m.err
	}

	return m.result, nil
}

// authorizerMock defines authorization behavior for handler tests.
type authorizerMock struct {
	// err defines auth errors.
	err error
}

// Require authenticates and authorizes requests.
func (m *authorizerMock) Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error {
	return m.err
}

// IsUnauthorized reports unauthorized errors.
func (m *authorizerMock) IsUnauthorized(err error) bool {
	return errors.Is(err, errUnauthorized)
}

// IsForbidden reports forbidden errors.
func (m *authorizerMock) IsForbidden(err error) bool {
	return errors.Is(err, errForbidden)
}

var (
	// errUnauthorized defines unauthorized test errors.
	errUnauthorized = errors.New("unauthorized")
	// errForbidden defines forbidden test errors.
	errForbidden = errors.New("forbidden")
)

// TestNewHandlerValidation verifies constructor validation behavior.
func TestNewHandlerValidation(t *testing.T) {
	if _, err := NewHandler(nil); !errors.Is(err, ErrNilService) {
		t.Fatalf("NewHandler() error = %v, want %v", err, ErrNilService)
	}
}

// TestQuoteRoute verifies successful quote route behavior.
func TestQuoteRoute(t *testing.T) {
	service := &serviceMock{result: &domain.QuoteResult{CarrierMessage: "ok", QuoteValue: 25800, BusinessUnit: domain.BusinessUnitCourier}}
	handler, err := NewHandler(service)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8301}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(handler.RegisterRoutes)

	request, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/shipping/quotes", bytes.NewBufferString(`{"carrier":"tcc","businessUnit":"courier","originCityCode":"05001","destinationCityCode":"11001","declaredValue":100000,"units":[{"number":1,"realWeight":2.5,"height":15,"width":20,"length":30}]}`))
	request.Header.Set("Content-Type", "application/json")

	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != stdhttp.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, stdhttp.StatusOK)
	}
	if service.request.Carrier != domain.CarrierTCC {
		t.Fatalf("service.request.Carrier = %q, want %q", service.request.Carrier, domain.CarrierTCC)
	}
}

// TestQuoteRouteInvalidPayload verifies invalid payload behavior.
func TestQuoteRouteInvalidPayload(t *testing.T) {
	handler, err := NewHandler(&serviceMock{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8302}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(handler.RegisterRoutes)

	request, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/shipping/quotes", bytes.NewBufferString("{invalid"))
	request.Header.Set("Content-Type", "application/json")

	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != stdhttp.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.StatusCode, stdhttp.StatusBadRequest)
	}
}

// TestQuoteRouteErrorMapping verifies quote error mapping behavior.
func TestQuoteRouteErrorMapping(t *testing.T) {
	testCases := []struct {
		name       string
		err        error
		wantStatus int
		port       int
	}{
		{name: "invalid", err: domain.ErrUnitNumberSequenceInvalid, wantStatus: stdhttp.StatusBadRequest, port: 8303},
		{name: "rejected", err: domain.ErrQuoteRejected, wantStatus: stdhttp.StatusBadGateway, port: 8304},
		{name: "unavailable", err: domain.ErrIntegrationUnavailable, wantStatus: stdhttp.StatusServiceUnavailable, port: 8305},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			handler, err := NewHandler(&serviceMock{err: testCase.err})
			if err != nil {
				t.Fatalf("NewHandler() error = %v", err)
			}

			server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: testCase.port}, nil)
			if err != nil {
				t.Fatalf("corehttp.New() error = %v", err)
			}
			server.RegisterRoutes(handler.RegisterRoutes)

			request, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/shipping/quotes", bytes.NewBufferString(`{"carrier":"tcc","businessUnit":"courier","originCityCode":"05001","destinationCityCode":"11001","declaredValue":100000,"units":[{"number":1,"realWeight":2.5,"height":15,"width":20,"length":30}]}`))
			request.Header.Set("Content-Type", "application/json")

			response, testErr := server.App().Test(request)
			if testErr != nil {
				t.Fatalf("App().Test() error = %v", testErr)
			}
			if response.StatusCode != testCase.wantStatus {
				t.Fatalf("status = %d, want %d", response.StatusCode, testCase.wantStatus)
			}
		})
	}
}

// TestQuoteRouteAuth verifies protected route behavior.
func TestQuoteRouteAuth(t *testing.T) {
	handler, err := NewHandler(&serviceMock{result: &domain.QuoteResult{}})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	handler.SetAuthorizer(&authorizerMock{err: errUnauthorized})

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8306}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(handler.RegisterRoutes)

	request, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/shipping/quotes", bytes.NewBufferString(`{"carrier":"tcc","businessUnit":"courier","originCityCode":"05001","destinationCityCode":"11001","declaredValue":100000,"units":[{"number":1,"realWeight":2.5,"height":15,"width":20,"length":30}]}`))
	request.Header.Set("Content-Type", "application/json")

	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != stdhttp.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.StatusCode, stdhttp.StatusUnauthorized)
	}
}
