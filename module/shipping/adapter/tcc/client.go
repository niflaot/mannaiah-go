package tcc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

const (
	// quotePath defines TCC quote endpoint path values.
	quotePath = "/api/clientes/tarifas/v5/consultarliquidacion"
)

// Client defines TCC quote gateway behavior.
type Client struct {
	// cfg defines adapter configuration values.
	cfg Config
	// httpClient defines HTTP client dependencies.
	httpClient *http.Client
	// tracer defines tracing dependencies.
	tracer trace.Tracer
	// requestCounter defines quote request counters.
	requestCounter metric.Int64Counter
	// durationHistogram defines quote duration histograms.
	durationHistogram metric.Float64Histogram
}

var (
	// _ ensures Client satisfies quote gateway contracts.
	_ port.RateQuoteGateway = (*Client)(nil)
)

// NewClient creates TCC quote adapter clients.
func NewClient(cfg Config) (*Client, error) {
	resolved, err := normalizeConfig(cfg)
	if err != nil {
		return nil, err
	}

	meter := otel.GetMeterProvider().Meter("mannaiah/shipping/tcc")
	requestCounter, err := meter.Int64Counter("shipping_tcc_quote_requests_total")
	if err != nil {
		return nil, err
	}
	durationHistogram, err := meter.Float64Histogram("shipping_tcc_quote_duration_ms")
	if err != nil {
		return nil, err
	}

	return &Client{
		cfg:               resolved,
		httpClient:        resolveHTTPClient(resolved),
		tracer:            otel.GetTracerProvider().Tracer("mannaiah/shipping/tcc"),
		requestCounter:    requestCounter,
		durationHistogram: durationHistogram,
	}, nil
}

// Quote retrieves one shipping quote from TCC.
func (c *Client) Quote(ctx context.Context, request domain.QuoteRequest) (*domain.QuoteResult, error) {
	ctx, span := c.tracer.Start(
		ctx,
		"shipping.tcc.quote",
		trace.WithAttributes(
			attribute.String("shipping.carrier", "tcc"),
			attribute.String("shipping.business_unit", string(request.BusinessUnit)),
		),
	)
	defer span.End()

	startedAt := time.Now()
	outcome := "success"
	defer func() {
		attrs := metric.WithAttributes(
			attribute.String("carrier", "tcc"),
			attribute.String("business_unit", string(request.BusinessUnit)),
			attribute.String("outcome", outcome),
		)
		c.requestCounter.Add(ctx, 1, attrs)
		c.durationHistogram.Record(ctx, float64(time.Since(startedAt).Milliseconds()), attrs)
	}()

	payload, err := c.buildQuotePayload(request)
	if err != nil {
		outcome = "invalid_request"
		return nil, err
	}

	body, err := json.Marshal(payload)
	if err != nil {
		outcome = "serialization_error"
		return nil, fmt.Errorf("%w: marshal quote request: %v", domain.ErrIntegrationUnavailable, err)
	}

	endpoint := strings.TrimRight(c.cfg.BaseURL, "/") + quotePath
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		outcome = "request_error"
		return nil, fmt.Errorf("%w: build quote request: %v", domain.ErrIntegrationUnavailable, err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("AccessToken", c.cfg.AccessToken)
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(httpRequest.Header))

	httpResponse, err := c.httpClient.Do(httpRequest)
	if err != nil {
		outcome = "transport_error"
		return nil, fmt.Errorf("%w: execute quote request: %v", domain.ErrIntegrationUnavailable, err)
	}
	defer func() {
		_ = httpResponse.Body.Close()
	}()

	responseBody, err := io.ReadAll(io.LimitReader(httpResponse.Body, 1<<20))
	if err != nil {
		outcome = "read_error"
		return nil, fmt.Errorf("%w: read quote response: %v", domain.ErrIntegrationUnavailable, err)
	}

	if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusMultipleChoices {
		outcome = "provider_error"
		message := strings.TrimSpace(string(responseBody))
		if message == "" {
			message = httpResponse.Status
		}
		return nil, fmt.Errorf("%w: tcc status %d: %s", domain.ErrQuoteRejected, httpResponse.StatusCode, message)
	}

	providerResponse := tccQuoteResponse{}
	if err := json.Unmarshal(responseBody, &providerResponse); err != nil {
		outcome = "invalid_response"
		return nil, fmt.Errorf("%w: parse quote response: %v", domain.ErrQuoteRejected, err)
	}

	if strings.TrimSpace(providerResponse.CodigoResultado) != "0" {
		outcome = "rejected"
		message := strings.TrimSpace(providerResponse.MensajeResultado)
		if message == "" {
			message = "quote rejected"
		}
		return nil, fmt.Errorf("%w: %s", domain.ErrQuoteRejected, message)
	}
	if providerResponse.Total == nil {
		outcome = "invalid_response"
		return nil, fmt.Errorf("%w: missing total payload", domain.ErrQuoteRejected)
	}

	return &domain.QuoteResult{
		CarrierMessage: strings.TrimSpace(providerResponse.MensajeResultado),
		QuoteValue:     float64(providerResponse.Total.TotalDespacho),
		BusinessUnit:   request.BusinessUnit,
	}, nil
}
