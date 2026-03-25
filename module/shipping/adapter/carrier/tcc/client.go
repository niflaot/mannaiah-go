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

	"go.uber.org/zap"
)

const (
	// defaultRequestTimeout defines outbound TCC HTTP timeout values.
	defaultRequestTimeout = 10 * time.Second
	// sandboxBaseURL defines the official TCC sandbox API base URL.
	sandboxBaseURL = "https://testsomos.tcc.com.co"
	// productionBaseURL defines the official TCC production API base URL.
	productionBaseURL = "https://somos.tcc.com.co"
	// quotePath defines quotation endpoint paths.
	quotePath = "/api/clientes/tarifas/v5/consultarliquidacion"
	// dispatchPath defines shipment-generation endpoint paths.
	dispatchPath = "/api/clientes/remesas/grabardespacho7"
	// trackingPath defines remittance-tracking endpoint paths.
	trackingPath = "/api/clientes/remesas/consultarestatusremesasv3"
)

// ClientConfig defines TCC API client configuration values.
type ClientConfig struct {
	// IsSandbox defines whether requests target TCC sandbox URLs.
	IsSandbox bool
	// BaseURLOverride defines optional base URL override values (for local tests only).
	BaseURLOverride string
	// AccessToken defines TCC access-token values.
	AccessToken string
	// RequestTimeout defines outbound request timeout values.
	RequestTimeout time.Duration
}

// Client defines TCC API client behavior.
type Client struct {
	// baseURL defines normalized TCC base URL values.
	baseURL string
	// accessToken defines TCC access-token values.
	accessToken string
	// httpClient defines outbound HTTP dependencies.
	httpClient *http.Client
}

// NewClient creates TCC API clients.
func NewClient(config ClientConfig) (*Client, error) {
	baseURL := productionBaseURL
	if config.IsSandbox {
		baseURL = sandboxBaseURL
	}
	if strings.TrimSpace(config.BaseURLOverride) != "" {
		baseURL = strings.TrimSpace(config.BaseURLOverride)
	}
	baseURL = strings.TrimRight(baseURL, "/")
	accessToken := strings.TrimSpace(config.AccessToken)
	if accessToken == "" {
		return nil, fmt.Errorf("tcc access token is required")
	}
	timeout := config.RequestTimeout
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}

	return &Client{baseURL: baseURL, accessToken: accessToken, httpClient: &http.Client{Timeout: timeout}}, nil
}

// Quote requests one quotation from the TCC quotation endpoint.
func (c *Client) Quote(ctx context.Context, request QuoteRequest) (*QuoteResponse, error) {
	response := QuoteResponse{}
	if err := c.postJSON(ctx, quotePath, request, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// Dispatch creates one shipment in TCC.
func (c *Client) Dispatch(ctx context.Context, request DispatchRequest) (*DispatchResponse, error) {
	response := DispatchResponse{}
	if err := c.postJSON(ctx, dispatchPath, request, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// Track requests one tracking response from TCC.
func (c *Client) Track(ctx context.Context, request TrackingRequest) (*TrackingResponse, error) {
	response := TrackingResponse{}
	if err := c.postJSON(ctx, trackingPath, request, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) postJSON(ctx context.Context, path string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal tcc request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build tcc request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("AccessToken", c.accessToken)

	start := time.Now()
	response, err := c.httpClient.Do(request)
	if err != nil {
		zap.L().Error("tcc http request failed",
			zap.String("path", path),
			zap.Duration("latency", time.Since(start)),
			zap.Error(err),
		)
		return fmt.Errorf("request tcc endpoint %s: %w", path, err)
	}
	defer func() { _ = response.Body.Close() }()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 5*1024*1024))
	if err != nil {
		zap.L().Error("tcc response read failed",
			zap.String("path", path),
			zap.Int("status", response.StatusCode),
			zap.Duration("latency", time.Since(start)),
			zap.Error(err),
		)
		return fmt.Errorf("read tcc response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		zap.L().Error("tcc http error",
			zap.String("path", path),
			zap.Int("status", response.StatusCode),
			zap.Duration("latency", time.Since(start)),
			zap.String("response_body", strings.TrimSpace(string(responseBody))),
		)
		return fmt.Errorf("tcc endpoint %s returned status %d: %s", path, response.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	if len(responseBody) == 0 {
		zap.L().Error("tcc empty response",
			zap.String("path", path),
			zap.Int("status", response.StatusCode),
			zap.Duration("latency", time.Since(start)),
		)
		return fmt.Errorf("tcc endpoint %s returned an empty response", path)
	}
	if err := json.Unmarshal(responseBody, out); err != nil {
		zap.L().Error("tcc response decode failed",
			zap.String("path", path),
			zap.Int("status", response.StatusCode),
			zap.Duration("latency", time.Since(start)),
			zap.String("response_body", strings.TrimSpace(string(responseBody))),
			zap.Error(err),
		)
		return fmt.Errorf("decode tcc response: %w", err)
	}
	zap.L().Info("tcc api call",
		zap.String("path", path),
		zap.Int("status", response.StatusCode),
		zap.Duration("latency", time.Since(start)),
	)

	return nil
}
