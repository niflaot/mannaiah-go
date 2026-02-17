package falabella

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	falabellasdk "github.com/ianfedev/antigravity/chatwoot-go/falabella-go/client"
)

// generatedClientMock defines generated-client test doubles.
type generatedClientMock struct {
	// response defines generated response payload values.
	response *falabellasdk.GetbrandsResp
	// updateResponse defines ProductUpdate response values.
	updateResponse *falabellasdk.ProductupdateResp
	// createResponse defines ProductCreate fallback response values.
	createResponse *falabellasdk.ProductcreateResp
	// err defines generated request errors.
	err error
	// updateErr defines ProductUpdate request errors.
	updateErr error
	// createErr defines ProductCreate request errors.
	createErr error
	// request captures request values after editor application.
	request *http.Request
	// updateRequest captures ProductUpdate request values.
	updateRequest *http.Request
	// createRequest captures ProductCreate request values.
	createRequest *http.Request
	// updateBody captures ProductUpdate request payload values.
	updateBody []byte
	// createBody captures ProductCreate request payload values.
	createBody []byte
}

// GetbrandsWithResponse captures request editors and returns configured values.
func (m *generatedClientMock) GetbrandsWithResponse(ctx context.Context, params *falabellasdk.GetbrandsParams, reqEditors ...falabellasdk.RequestEditorFn) (*falabellasdk.GetbrandsResp, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://sellercenter-api.falabella.com/?Action=GetBrands", nil)
	if err != nil {
		return nil, err
	}
	for _, editor := range reqEditors {
		if editorErr := editor(ctx, request); editorErr != nil {
			return nil, editorErr
		}
	}
	m.request = request

	if m.err != nil {
		return nil, m.err
	}

	return m.response, nil
}

// ProductupdateWithBodyWithResponse captures update payloads and returns configured values.
func (m *generatedClientMock) ProductupdateWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...falabellasdk.RequestEditorFn) (*falabellasdk.ProductupdateResp, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://sellercenter-api.falabella.com/?Action=ProductUpdate", nil)
	if err != nil {
		return nil, err
	}
	for _, editor := range reqEditors {
		if editorErr := editor(ctx, request); editorErr != nil {
			return nil, editorErr
		}
	}
	m.updateRequest = request

	if body != nil {
		payload, readErr := io.ReadAll(body)
		if readErr != nil {
			return nil, readErr
		}
		m.updateBody = append([]byte(nil), payload...)
	}

	if m.updateErr != nil {
		return nil, m.updateErr
	}

	return m.updateResponse, nil
}

// ProductcreateWithBodyWithResponse captures create payloads and returns configured values.
func (m *generatedClientMock) ProductcreateWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...falabellasdk.RequestEditorFn) (*falabellasdk.ProductcreateResp, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://sellercenter-api.falabella.com/?Action=ProductCreate", nil)
	if err != nil {
		return nil, err
	}
	for _, editor := range reqEditors {
		if editorErr := editor(ctx, request); editorErr != nil {
			return nil, editorErr
		}
	}
	m.createRequest = request

	if body != nil {
		payload, readErr := io.ReadAll(body)
		if readErr != nil {
			return nil, readErr
		}
		m.createBody = append([]byte(nil), payload...)
	}

	if m.createErr != nil {
		return nil, m.createErr
	}

	return m.createResponse, nil
}

// TestNormalizeConfigValidation verifies configuration validation behavior.
func TestNormalizeConfigValidation(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want error
	}{
		{name: "missing_url", cfg: Config{}, want: ErrMissingURL},
		{name: "invalid_url", cfg: Config{URL: "not-url", UserID: "u", APIKey: "k"}, want: ErrInvalidURL},
		{name: "missing_user", cfg: Config{URL: "https://example.com", APIKey: "k"}, want: ErrMissingUserID},
		{name: "missing_key", cfg: Config{URL: "https://example.com", UserID: "u"}, want: ErrMissingAPIKey},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := normalizeConfig(tt.cfg)
			if !errors.Is(err, tt.want) {
				t.Fatalf("normalizeConfig() error = %v, want %v", err, tt.want)
			}
		})
	}
}

// TestGetBrandsSuccess verifies signed GetBrands retrieval behavior.
func TestGetBrandsSuccess(t *testing.T) {
	timestamp := time.Date(2026, time.February, 16, 12, 30, 0, 0, time.UTC)
	response := &falabellasdk.GetbrandsResp{
		Body: []byte(`{"SuccessResponse":{"Head":{"RequestId":"r1"}}}`),
		HTTPResponse: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		},
	}
	generated := &generatedClientMock{response: response}
	client := newClientWithDependencies(
		Config{URL: "https://sellercenter-api.falabella.com", UserID: "user-1", APIKey: "key-1", Version: "1.0", Timeout: 5 * time.Second},
		generated,
		func() time.Time { return timestamp },
	)

	payload, err := client.GetBrands(context.Background())
	if err != nil {
		t.Fatalf("GetBrands() error = %v", err)
	}
	if string(payload) != string(response.Body) {
		t.Fatalf("GetBrands() payload = %q, want %q", string(payload), string(response.Body))
	}

	query := generated.request.URL.Query()
	if query.Get("Action") != getBrandsAction {
		t.Fatalf("Action = %q, want %q", query.Get("Action"), getBrandsAction)
	}
	if query.Get("Format") != defaultFormat {
		t.Fatalf("Format = %q, want %q", query.Get("Format"), defaultFormat)
	}
	if query.Get("UserID") != "user-1" {
		t.Fatalf("UserID = %q, want %q", query.Get("UserID"), "user-1")
	}
	if query.Get("Version") != "1.0" {
		t.Fatalf("Version = %q, want %q", query.Get("Version"), "1.0")
	}
	if query.Get("Timestamp") != "2026-02-16T12:30:00Z" {
		t.Fatalf("Timestamp = %q, want %q", query.Get("Timestamp"), "2026-02-16T12:30:00Z")
	}
	if query.Get("Signature") == "" {
		t.Fatalf("Signature should not be empty")
	}
}

// TestGetBrandsFailure verifies non-success response behavior.
func TestGetBrandsFailure(t *testing.T) {
	generated := &generatedClientMock{
		response: &falabellasdk.GetbrandsResp{
			Body: []byte(`{"error":"bad"}`),
			HTTPResponse: &http.Response{
				StatusCode: http.StatusBadRequest,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			},
		},
	}
	client := newClientWithDependencies(
		Config{URL: "https://sellercenter-api.falabella.com", UserID: "user-1", APIKey: "key-1", Version: "1.0", Timeout: 5 * time.Second},
		generated,
		time.Now,
	)

	_, err := client.GetBrands(context.Background())
	if err == nil {
		t.Fatalf("GetBrands() expected error")
	}
}

// TestValidateDelegates verifies Validate() delegation behavior.
func TestValidateDelegates(t *testing.T) {
	generated := &generatedClientMock{
		response: &falabellasdk.GetbrandsResp{
			Body: []byte(`{"ok":true}`),
			HTTPResponse: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			},
		},
	}
	client := newClientWithDependencies(
		Config{URL: "https://sellercenter-api.falabella.com", UserID: "user-1", APIKey: "key-1", Version: "1.0", Timeout: 5 * time.Second},
		generated,
		time.Now,
	)

	if err := client.Validate(context.Background()); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

// TestTrimBody verifies trimmed diagnostics behavior.
func TestTrimBody(t *testing.T) {
	if value := trimBody(nil); value != "<empty>" {
		t.Fatalf("trimBody(nil) = %q, want %q", value, "<empty>")
	}
	if value := trimBody([]byte("  hello  ")); value != "hello" {
		t.Fatalf("trimBody() = %q, want %q", value, "hello")
	}
}

// TestSyncProductUpdateSuccess verifies ProductUpdate success behavior.
func TestSyncProductUpdateSuccess(t *testing.T) {
	timestamp := time.Date(2026, time.February, 16, 12, 30, 0, 0, time.UTC)
	generated := &generatedClientMock{
		updateResponse: &falabellasdk.ProductupdateResp{
			Body: []byte(`<SuccessResponse/>`),
			HTTPResponse: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/xml"}},
			},
		},
	}
	client := newClientWithDependencies(
		Config{URL: "https://sellercenter-api.falabella.com", UserID: "user-1", APIKey: "key-1", Version: "1.0", Timeout: 5 * time.Second},
		generated,
		func() time.Time { return timestamp },
	)

	payload, err := client.SyncProduct(context.Background(), syncProductRequestFixture())
	if err != nil {
		t.Fatalf("SyncProduct() error = %v", err)
	}
	if string(payload) != "<SuccessResponse/>" {
		t.Fatalf("SyncProduct() payload = %q, want %q", string(payload), "<SuccessResponse/>")
	}
	if query := generated.updateRequest.URL.Query(); query.Get("Action") != productUpdateAction {
		t.Fatalf("Action = %q, want %q", query.Get("Action"), productUpdateAction)
	}
	if !bytes.Contains(generated.updateBody, []byte("<SellerSku>SKU-1</SellerSku>")) {
		t.Fatalf("expected SellerSku in payload")
	}
}

// TestSyncProductCreateFallback verifies ProductCreate fallback behavior.
func TestSyncProductCreateFallback(t *testing.T) {
	generated := &generatedClientMock{
		updateResponse: &falabellasdk.ProductupdateResp{
			Body: []byte(`<ErrorResponse/>`),
			HTTPResponse: &http.Response{
				StatusCode: http.StatusBadRequest,
				Header:     http.Header{"Content-Type": []string{"application/xml"}},
			},
		},
		createResponse: &falabellasdk.ProductcreateResp{
			Body: []byte(`<SuccessResponse/>`),
			HTTPResponse: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/xml"}},
			},
		},
	}
	client := newClientWithDependencies(
		Config{URL: "https://sellercenter-api.falabella.com", UserID: "user-1", APIKey: "key-1", Version: "1.0", Timeout: 5 * time.Second},
		generated,
		time.Now,
	)

	payload, err := client.SyncProduct(context.Background(), syncProductRequestFixture())
	if err != nil {
		t.Fatalf("SyncProduct() error = %v", err)
	}
	if string(payload) != "<SuccessResponse/>" {
		t.Fatalf("SyncProduct() payload = %q, want %q", string(payload), "<SuccessResponse/>")
	}
	if query := generated.createRequest.URL.Query(); query.Get("Action") != productCreateAction {
		t.Fatalf("Action = %q, want %q", query.Get("Action"), productCreateAction)
	}
}

// TestSyncProductFailure verifies sync failure behavior.
func TestSyncProductFailure(t *testing.T) {
	generated := &generatedClientMock{
		updateResponse: &falabellasdk.ProductupdateResp{
			Body: []byte(`<ErrorResponse>update</ErrorResponse>`),
			HTTPResponse: &http.Response{
				StatusCode: http.StatusBadRequest,
				Header:     http.Header{"Content-Type": []string{"application/xml"}},
			},
		},
		createResponse: &falabellasdk.ProductcreateResp{
			Body: []byte(`<ErrorResponse>create</ErrorResponse>`),
			HTTPResponse: &http.Response{
				StatusCode: http.StatusBadRequest,
				Header:     http.Header{"Content-Type": []string{"application/xml"}},
			},
		},
	}
	client := newClientWithDependencies(
		Config{URL: "https://sellercenter-api.falabella.com", UserID: "user-1", APIKey: "key-1", Version: "1.0", Timeout: 5 * time.Second},
		generated,
		time.Now,
	)

	_, err := client.SyncProduct(context.Background(), syncProductRequestFixture())
	if err == nil {
		t.Fatalf("SyncProduct() expected error")
	}
}
