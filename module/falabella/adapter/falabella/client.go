package falabella

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	falabellasdk "github.com/ianfedev/antigravity/chatwoot-go/falabella-go/client"
	"mannaiah/module/falabella/port"
)

const (
	// getBrandsAction defines Falabella action values used for connection checks.
	getBrandsAction = "GetBrands"
	// productCreateAction defines Falabella action values used for product creation.
	productCreateAction = "ProductCreate"
	// productUpdateAction defines Falabella action values used for product updates.
	productUpdateAction = "ProductUpdate"
	// defaultFormat defines Falabella request format values.
	defaultFormat = "JSON"
)

var (
	// ErrMissingURL is returned when Falabella URL values are missing.
	ErrMissingURL = errors.New("falabella url is required")
	// ErrMissingUserID is returned when Falabella user values are missing.
	ErrMissingUserID = errors.New("falabella user id is required")
	// ErrMissingAPIKey is returned when Falabella api key values are missing.
	ErrMissingAPIKey = errors.New("falabella api key is required")
	// ErrInvalidURL is returned when Falabella URL values are invalid.
	ErrInvalidURL = errors.New("falabella url is invalid")
	// ErrEmptyResponse is returned when Falabella responses contain no payload.
	ErrEmptyResponse = errors.New("falabella response body is empty")
)

// generatedClient defines generated Falabella client behavior required by this adapter.
type generatedClient interface {
	// GetbrandsWithResponse invokes Falabella GetBrands requests and returns parsed responses.
	GetbrandsWithResponse(ctx context.Context, params *falabellasdk.GetbrandsParams, reqEditors ...falabellasdk.RequestEditorFn) (*falabellasdk.GetbrandsResp, error)
	// ProductupdateWithBodyWithResponse invokes Falabella ProductUpdate requests.
	ProductupdateWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...falabellasdk.RequestEditorFn) (*falabellasdk.ProductupdateResp, error)
	// ProductcreateWithBodyWithResponse invokes Falabella ProductCreate requests.
	ProductcreateWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...falabellasdk.RequestEditorFn) (*falabellasdk.ProductcreateResp, error)
}

// Client defines Falabella source adapter behavior.
type Client struct {
	// cfg defines Falabella client configuration values.
	cfg Config
	// generated defines generated Falabella client dependencies.
	generated generatedClient
	// now resolves current timestamp values for signed requests.
	now func() time.Time
}

var (
	// _ ensures Client satisfies port contracts.
	_ interface {
		Validate(ctx context.Context) error
		GetBrands(ctx context.Context) ([]byte, error)
		SyncProduct(ctx context.Context, request port.SyncProductRequest) ([]byte, error)
	} = (*Client)(nil)
)

// NewClient creates Falabella source adapters backed by falabella-go generated clients.
func NewClient(cfg Config) (*Client, error) {
	resolvedCfg, err := normalizeConfig(cfg)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{Timeout: resolvedCfg.Timeout}
	generated, err := falabellasdk.NewClientWithResponses(
		resolvedCfg.URL,
		falabellasdk.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, fmt.Errorf("create generated falabella client: %w", err)
	}

	return newClientWithDependencies(resolvedCfg, generated, time.Now), nil
}

// Validate verifies integration availability by executing Falabella GetBrands.
func (c *Client) Validate(ctx context.Context) error {
	_, err := c.GetBrands(ctx)
	if err != nil {
		return err
	}

	return nil
}

// GetBrands retrieves Falabella brand payload using signed requests.
func (c *Client) GetBrands(ctx context.Context) ([]byte, error) {
	response, err := c.generated.GetbrandsWithResponse(ctx, nil, c.newGetBrandsEditor())
	if err != nil {
		return nil, fmt.Errorf("falabella get brands request: %w", err)
	}
	if response == nil || response.HTTPResponse == nil {
		return nil, errors.New("falabella get brands response is nil")
	}
	if response.StatusCode() >= http.StatusBadRequest {
		return nil, fmt.Errorf(
			"falabella get brands status %d: %s",
			response.StatusCode(),
			trimBody(response.Body),
		)
	}
	if len(response.Body) == 0 {
		return nil, ErrEmptyResponse
	}

	return response.Body, nil
}

// SyncProduct upserts a product into Falabella by running ProductUpdate and ProductCreate fallback.
func (c *Client) SyncProduct(ctx context.Context, request port.SyncProductRequest) ([]byte, error) {
	payload, err := buildProductRequestXML(request)
	if err != nil {
		return nil, fmt.Errorf("build falabella product payload: %w", err)
	}

	updateResponse, err := c.generated.ProductupdateWithBodyWithResponse(
		ctx,
		"application/xml",
		bytes.NewReader(payload),
		c.newActionEditor(productUpdateAction),
	)
	if err != nil {
		return nil, fmt.Errorf("falabella product update request: %w", err)
	}
	if updateResponse == nil || updateResponse.HTTPResponse == nil {
		return nil, errors.New("falabella product update response is nil")
	}
	if updateResponse.StatusCode() < http.StatusBadRequest {
		return updateResponse.Body, nil
	}

	createResponse, err := c.generated.ProductcreateWithBodyWithResponse(
		ctx,
		"application/xml",
		bytes.NewReader(payload),
		c.newActionEditor(productCreateAction),
	)
	if err != nil {
		return nil, fmt.Errorf("falabella product create fallback request: %w", err)
	}
	if createResponse == nil || createResponse.HTTPResponse == nil {
		return nil, errors.New("falabella product create response is nil")
	}
	if createResponse.StatusCode() >= http.StatusBadRequest {
		return nil, fmt.Errorf(
			"falabella product upsert failed (update=%d create=%d): update=%s create=%s",
			updateResponse.StatusCode(),
			createResponse.StatusCode(),
			trimBody(updateResponse.Body),
			trimBody(createResponse.Body),
		)
	}

	return createResponse.Body, nil
}

// newGetBrandsEditor builds signed request editors for Falabella GetBrands.
func (c *Client) newGetBrandsEditor() falabellasdk.RequestEditorFn {
	return c.newActionEditor(getBrandsAction)
}

// newActionEditor builds signed request editors for Falabella actions.
func (c *Client) newActionEditor(action string) falabellasdk.RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		if req == nil || req.URL == nil {
			return errors.New("falabella request is nil")
		}

		timestamp := c.now().UTC().Format(time.RFC3339)
		params := map[string]string{
			"Action":    action,
			"Format":    defaultFormat,
			"Timestamp": timestamp,
			"UserID":    c.cfg.UserID,
			"Version":   c.cfg.Version,
		}
		signature := signParams(c.cfg.APIKey, params)

		query := req.URL.Query()
		for key, value := range params {
			query.Set(key, value)
		}
		query.Set("Signature", signature)
		req.URL.RawQuery = query.Encode()
		return nil
	}
}

// newClientWithDependencies creates client instances with injected dependencies.
func newClientWithDependencies(cfg Config, generated generatedClient, now func() time.Time) *Client {
	return &Client{cfg: cfg, generated: generated, now: now}
}

// normalizeConfig resolves config defaults and validates mandatory values.
func normalizeConfig(cfg Config) (Config, error) {
	resolved := cfg
	resolved.URL = strings.TrimSpace(resolved.URL)
	resolved.UserID = strings.TrimSpace(resolved.UserID)
	resolved.APIKey = strings.TrimSpace(resolved.APIKey)
	resolved.Version = strings.TrimSpace(resolved.Version)

	if resolved.URL == "" {
		return Config{}, ErrMissingURL
	}
	if _, err := url.ParseRequestURI(resolved.URL); err != nil {
		return Config{}, fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}
	if resolved.UserID == "" {
		return Config{}, ErrMissingUserID
	}
	if resolved.APIKey == "" {
		return Config{}, ErrMissingAPIKey
	}
	if resolved.Timeout <= 0 {
		resolved.Timeout = 5 * time.Second
	}
	if resolved.Version == "" {
		resolved.Version = defaultVersion
	}

	return resolved, nil
}

// trimBody resolves trimmed response body strings for diagnostics.
func trimBody(body []byte) string {
	value := strings.TrimSpace(string(body))
	if value == "" {
		return "<empty>"
	}
	if len(value) > 256 {
		return value[:256] + "..."
	}

	return value
}
