package falabella

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mannaiah/module/falabella/port"
	"net/http"
	"strings"
	"testing"
	"time"
)

// doerMock defines outbound HTTP test doubles.
type doerMock struct {
	// fn resolves mocked response values from requests.
	fn func(req *http.Request, body []byte) (*http.Response, error)
	// requests captures outbound request values.
	requests []*http.Request
	// bodies captures outbound request body values.
	bodies [][]byte
}

// Do records outbound requests and delegates to configured response behavior.
func (d *doerMock) Do(req *http.Request) (*http.Response, error) {
	if d == nil {
		return nil, errors.New("nil doer")
	}

	var body []byte
	if req != nil && req.Body != nil {
		payload, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		body = append([]byte(nil), payload...)
	}

	clone := req.Clone(req.Context())
	d.requests = append(d.requests, clone)
	d.bodies = append(d.bodies, append([]byte(nil), body...))
	if d.fn != nil {
		return d.fn(clone, body)
	}

	return newHTTPResponse(http.StatusOK, http.Header{"Content-Type": []string{"application/json"}}, `{"SuccessResponse":{}}`), nil
}

// newHTTPResponse builds outbound HTTP response fixtures.
func newHTTPResponse(statusCode int, header http.Header, body string) *http.Response {
	if header == nil {
		header = http.Header{}
	}
	return &http.Response{
		StatusCode: statusCode,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
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

// TestNormalizeConfigDefaults verifies default configuration value behavior.
func TestNormalizeConfigDefaults(t *testing.T) {
	resolved, err := normalizeConfig(Config{
		URL:    "https://sellercenter-api.falabella.com",
		UserID: "user-1",
		APIKey: "key-1",
	})
	if err != nil {
		t.Fatalf("normalizeConfig() error = %v", err)
	}
	if resolved.UserAgent != defaultUserAgent {
		t.Fatalf("UserAgent = %q, want %q", resolved.UserAgent, defaultUserAgent)
	}
	if resolved.Version != "1.0" {
		t.Fatalf("Version = %q, want %q", resolved.Version, "1.0")
	}
}

// TestGetBrandsSuccess verifies signed GetBrands retrieval behavior.
func TestGetBrandsSuccess(t *testing.T) {
	timestamp := time.Date(2026, time.February, 16, 12, 30, 0, 0, time.UTC)
	doer := &doerMock{fn: func(req *http.Request, body []byte) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", req.Method)
		}
		if len(body) != 0 {
			t.Fatalf("expected empty body, got %d bytes", len(body))
		}
		return newHTTPResponse(http.StatusOK, http.Header{"Content-Type": []string{"application/json"}}, `{"SuccessResponse":{"Head":{"RequestId":"r1"}}}`), nil
	}}
	client := newClientWithDependencies(
		Config{URL: "https://sellercenter-api.falabella.com", UserID: "user-1", APIKey: "key-1", Version: "1.0", Timeout: 5 * time.Second},
		doer,
		func() time.Time { return timestamp },
	)

	payload, err := client.GetBrands(context.Background())
	if err != nil {
		t.Fatalf("GetBrands() error = %v", err)
	}
	if string(payload) != `{"SuccessResponse":{"Head":{"RequestId":"r1"}}}` {
		t.Fatalf("payload = %q", payload)
	}

	if len(doer.requests) != 1 {
		t.Fatalf("len(doer.requests) = %d, want 1", len(doer.requests))
	}
	query := doer.requests[0].URL.Query()
	if query.Get("Action") != getBrandsAction {
		t.Fatalf("Action = %q, want %q", query.Get("Action"), getBrandsAction)
	}
	if query.Get("Format") != defaultFormat {
		t.Fatalf("Format = %q, want %q", query.Get("Format"), defaultFormat)
	}
	if query.Get("Timestamp") != "2026-02-16T12:30:00+0000" {
		t.Fatalf("Timestamp = %q, want %q", query.Get("Timestamp"), "2026-02-16T12:30:00+0000")
	}
	if query.Get("Signature") == "" {
		t.Fatalf("Signature should not be empty")
	}
	if !strings.Contains(doer.requests[0].URL.RawQuery, "&Signature=") {
		t.Fatalf("raw query missing signature parameter: %s", doer.requests[0].URL.RawQuery)
	}
	if !strings.HasSuffix(doer.requests[0].URL.RawQuery, "Signature="+query.Get("Signature")) {
		t.Fatalf("expected signature parameter appended last in raw query: %s", doer.requests[0].URL.RawQuery)
	}
}

// TestGetBrandsSignatureMismatchFallback verifies fallback signing-profile behavior on E007 responses.
func TestGetBrandsSignatureMismatchFallback(t *testing.T) {
	calls := 0
	timestamps := make([]string, 0, 8)
	formats := make([]string, 0, 8)
	signatures := make([]string, 0, 8)
	doer := &doerMock{fn: func(req *http.Request, body []byte) (*http.Response, error) {
		calls++
		query := req.URL.Query()
		timestamps = append(timestamps, query.Get("Timestamp"))
		formats = append(formats, query.Get("Format"))
		signatures = append(signatures, query.Get("Signature"))
		if calls < 8 {
			return newHTTPResponse(http.StatusOK, http.Header{"Content-Type": []string{"application/json"}}, `{"ErrorResponse":{"Head":{"ErrorCode":"7","ErrorMessage":"E007: Login failed. Signature mismatch"}}}`), nil
		}
		return newHTTPResponse(http.StatusOK, http.Header{"Content-Type": []string{"application/json"}}, `{"SuccessResponse":{"Head":{"RequestId":"r1"}}}`), nil
	}}
	client := newClientWithDependencies(
		Config{URL: "https://sellercenter-api.falabella.com", UserID: "user-1", APIKey: "key-1", Version: "1.0", Timeout: 5 * time.Second},
		doer,
		func() time.Time { return time.Date(2026, time.February, 16, 12, 30, 0, 0, time.UTC) },
	)

	payload, err := client.GetBrands(context.Background())
	if err != nil {
		t.Fatalf("GetBrands() error = %v", err)
	}
	if string(payload) != `{"SuccessResponse":{"Head":{"RequestId":"r1"}}}` {
		t.Fatalf("payload = %q", payload)
	}
	if calls != 8 {
		t.Fatalf("calls = %d, want 8", calls)
	}
	if want := []string{"XML", "XML", "XML", "XML", "", "", "", ""}; strings.Join(formats, ",") != strings.Join(want, ",") {
		t.Fatalf("formats = %+v, want %+v", formats, want)
	}
	if len(signatures) != 8 {
		t.Fatalf("len(signatures) = %d, want 8", len(signatures))
	}
	if signatures[0] == "" || signatures[1] == "" {
		t.Fatalf("expected populated signatures for first fallback attempts: %+v", signatures[:2])
	}
	if timestamps[0] != timestamps[1] {
		t.Fatalf("timestamps[0] = %q, timestamps[1] = %q; want equal for same timestamp layout", timestamps[0], timestamps[1])
	}
	if strings.ToUpper(signatures[0]) != signatures[1] {
		t.Fatalf("signatures[1] = %q, want uppercase of signatures[0] = %q", signatures[1], strings.ToUpper(signatures[0]))
	}
}

// TestValidateNoHTTPCalls verifies Validate() does not issue outbound HTTP requests.
func TestValidateNoHTTPCalls(t *testing.T) {
	doer := &doerMock{fn: func(req *http.Request, body []byte) (*http.Response, error) {
		t.Fatalf("Validate() should not issue HTTP requests")
		return nil, nil
	}}
	client := newClientWithDependencies(
		Config{URL: "https://sellercenter-api.falabella.com", UserID: "user-1", APIKey: "key-1", Version: "1.0", Timeout: 5 * time.Second},
		doer,
		time.Now,
	)

	if err := client.Validate(context.Background()); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if len(doer.requests) != 0 {
		t.Fatalf("len(doer.requests) = %d, want 0", len(doer.requests))
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
	calls := 0
	doer := &doerMock{fn: func(req *http.Request, body []byte) (*http.Response, error) {
		calls++
		switch req.URL.Query().Get("Action") {
		case getProductsAction:
			return newHTTPResponse(http.StatusOK, http.Header{"Content-Type": []string{"application/json"}}, `{"SuccessResponse":{"Body":{"Products":{"Product":[{"SellerSku":"SKU-1"}]}}}}`), nil
		case productUpdateAction:
			if req.URL.Query().Get("Format") != "XML" {
				t.Fatalf("Format = %q, want %q", req.URL.Query().Get("Format"), "XML")
			}
			if req.Header.Get("Content-Type") != xmlContentType {
				t.Fatalf("Content-Type = %q, want %q", req.Header.Get("Content-Type"), xmlContentType)
			}
			payload := string(body)
			if !strings.Contains(payload, "<SellerSku>SKU-1</SellerSku>") {
				t.Fatalf("payload missing SellerSku: %s", payload)
			}
			if !strings.Contains(payload, "<BusinessUnits>") || !strings.Contains(payload, "<BusinessUnit>") {
				t.Fatalf("payload missing BusinessUnits structure: %s", payload)
			}
			if !strings.Contains(payload, "<OperatorCode>FACO</OperatorCode>") {
				t.Fatalf("payload missing OperatorCode: %s", payload)
			}
			if !strings.Contains(payload, "<Color>Navy</Color>") {
				t.Fatalf("payload missing Color: %s", payload)
			}
			if !strings.Contains(payload, "<Talla>L</Talla>") {
				t.Fatalf("payload missing Talla: %s", payload)
			}
			if !strings.Contains(payload, "<ColorBasico>Blue</ColorBasico>") {
				t.Fatalf("payload missing ColorBasico: %s", payload)
			}

			return newHTTPResponse(http.StatusOK, http.Header{"Content-Type": []string{"application/json"}}, `{"SuccessResponse":{}}`), nil
		default:
			t.Fatalf("unexpected action: %s", req.URL.Query().Get("Action"))
			return nil, nil
		}
	}}
	client := newClientWithDependencies(
		Config{URL: "https://sellercenter-api.falabella.com", UserID: "user-1", APIKey: "key-1", Version: "1.0", Timeout: 5 * time.Second},
		doer,
		time.Now,
	)

	request := syncProductRequestFixture()
	request.ParentSKU = ""
	payload, err := client.SyncProduct(context.Background(), request)
	if err != nil {
		t.Fatalf("SyncProduct() error = %v", err)
	}
	if string(payload) != `{"SuccessResponse":{}}` {
		t.Fatalf("payload = %s", payload)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

// TestSyncProductCreateWhenMissing verifies ProductCreate behavior when SKU is not found.
func TestSyncProductCreateWhenMissing(t *testing.T) {
	calls := 0
	doer := &doerMock{fn: func(req *http.Request, body []byte) (*http.Response, error) {
		calls++
		switch req.URL.Query().Get("Action") {
		case getProductsAction:
			return newHTTPResponse(http.StatusOK, http.Header{"Content-Type": []string{"application/json"}}, `{"SuccessResponse":{"Body":{"Products":{"Product":[]}}}}`), nil
		case productCreateAction:
			if req.URL.Query().Get("Format") != "XML" {
				t.Fatalf("create Format = %q, want %q", req.URL.Query().Get("Format"), "XML")
			}
			if !strings.Contains(string(body), "<SellerSku>SKU-1</SellerSku>") {
				t.Fatalf("expected xml product payload on create")
			}
			if !strings.Contains(string(body), "<OperatorCode>FACO</OperatorCode>") {
				t.Fatalf("expected OperatorCode in create payload")
			}
			return newHTTPResponse(http.StatusOK, http.Header{"Content-Type": []string{"application/json"}}, `{"SuccessResponse":{}}`), nil
		default:
			t.Fatalf("unexpected action: %s", req.URL.Query().Get("Action"))
			return nil, nil
		}
	}}
	client := newClientWithDependencies(
		Config{URL: "https://sellercenter-api.falabella.com", UserID: "user-1", APIKey: "key-1", Version: "1.0", Timeout: 5 * time.Second},
		doer,
		time.Now,
	)

	request := syncProductRequestFixture()
	request.ParentSKU = ""
	payload, err := client.SyncProduct(context.Background(), request)
	if err != nil {
		t.Fatalf("SyncProduct() error = %v", err)
	}
	if string(payload) != `{"SuccessResponse":{}}` {
		t.Fatalf("payload = %s", payload)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

// TestSyncProductUpdateWhenParentSKUExists verifies ProductUpdate behavior when child SKU is missing but parent SKU exists.
func TestSyncProductUpdateWhenParentSKUExists(t *testing.T) {
	calls := 0
	doer := &doerMock{fn: func(req *http.Request, body []byte) (*http.Response, error) {
		calls++
		action := req.URL.Query().Get("Action")
		skuList := req.URL.Query().Get("SkuSellerList")

		switch action {
		case getProductsAction:
			switch {
			case strings.Contains(skuList, "SKU-CHILD"):
				return newHTTPResponse(http.StatusOK, http.Header{"Content-Type": []string{"application/json"}}, `{"SuccessResponse":{"Body":{"Products":{"Product":[]}}}}`), nil
			case strings.Contains(skuList, "SKU-PARENT"):
				return newHTTPResponse(http.StatusOK, http.Header{"Content-Type": []string{"application/json"}}, `{"SuccessResponse":{"Body":{"Products":{"Product":[{"SellerSku":"SKU-PARENT"}]}}}}`), nil
			default:
				t.Fatalf("unexpected SkuSellerList = %q", skuList)
				return nil, nil
			}
		case productUpdateAction:
			if !strings.Contains(string(body), "<SellerSku>SKU-CHILD</SellerSku>") {
				t.Fatalf("expected update payload with child sku, got: %s", string(body))
			}
			return newHTTPResponse(http.StatusOK, http.Header{"Content-Type": []string{"application/json"}}, `{"SuccessResponse":{}}`), nil
		default:
			t.Fatalf("unexpected action: %s", action)
			return nil, nil
		}
	}}
	client := newClientWithDependencies(
		Config{URL: "https://sellercenter-api.falabella.com", UserID: "user-1", APIKey: "key-1", Version: "1.0", Timeout: 5 * time.Second},
		doer,
		time.Now,
	)

	request := syncProductRequestFixture()
	request.SKU = "SKU-CHILD"
	request.ParentSKU = "SKU-PARENT"

	payload, err := client.SyncProduct(context.Background(), request)
	if err != nil {
		t.Fatalf("SyncProduct() error = %v", err)
	}
	if string(payload) != `{"SuccessResponse":{}}` {
		t.Fatalf("payload = %s", payload)
	}
	if calls != 3 {
		t.Fatalf("calls = %d, want 3", calls)
	}
}

// TestSyncProductCreateWhenParentAndChildMissing verifies ProductCreate behavior when neither child nor parent SKU exists.
func TestSyncProductCreateWhenParentAndChildMissing(t *testing.T) {
	calls := 0
	doer := &doerMock{fn: func(req *http.Request, body []byte) (*http.Response, error) {
		calls++
		action := req.URL.Query().Get("Action")

		switch action {
		case getProductsAction:
			return newHTTPResponse(http.StatusOK, http.Header{"Content-Type": []string{"application/json"}}, `{"SuccessResponse":{"Body":{"Products":{"Product":[]}}}}`), nil
		case productCreateAction:
			if !strings.Contains(string(body), "<SellerSku>SKU-CHILD</SellerSku>") {
				t.Fatalf("expected create payload with child sku, got: %s", string(body))
			}
			return newHTTPResponse(http.StatusOK, http.Header{"Content-Type": []string{"application/json"}}, `{"SuccessResponse":{}}`), nil
		default:
			t.Fatalf("unexpected action: %s", action)
			return nil, nil
		}
	}}
	client := newClientWithDependencies(
		Config{URL: "https://sellercenter-api.falabella.com", UserID: "user-1", APIKey: "key-1", Version: "1.0", Timeout: 5 * time.Second},
		doer,
		time.Now,
	)

	request := syncProductRequestFixture()
	request.SKU = "SKU-CHILD"
	request.ParentSKU = "SKU-PARENT"

	payload, err := client.SyncProduct(context.Background(), request)
	if err != nil {
		t.Fatalf("SyncProduct() error = %v", err)
	}
	if string(payload) != `{"SuccessResponse":{}}` {
		t.Fatalf("payload = %s", payload)
	}
	if calls != 3 {
		t.Fatalf("calls = %d, want 3", calls)
	}
}

// TestSyncProductFailure verifies sync failure behavior.
func TestSyncProductFailure(t *testing.T) {
	doer := &doerMock{fn: func(req *http.Request, body []byte) (*http.Response, error) {
		switch req.URL.Query().Get("Action") {
		case getProductsAction:
			return newHTTPResponse(http.StatusOK, http.Header{"Content-Type": []string{"application/json"}}, `{"SuccessResponse":{"Body":{"Products":{"Product":[{"SellerSku":"SKU-1"}]}}}}`), nil
		case productUpdateAction:
			return newHTTPResponse(http.StatusBadRequest, http.Header{"Content-Type": []string{"application/json"}}, `{"ErrorResponse":{"Head":{"ErrorCode":"1000","ErrorMessage":"failed"}}}`), nil
		default:
			t.Fatalf("unexpected action: %s", req.URL.Query().Get("Action"))
			return nil, nil
		}
	}}
	client := newClientWithDependencies(
		Config{URL: "https://sellercenter-api.falabella.com", UserID: "user-1", APIKey: "key-1", Version: "1.0", Timeout: 5 * time.Second},
		doer,
		time.Now,
	)

	_, err := client.SyncProduct(context.Background(), syncProductRequestFixture())
	if err == nil {
		t.Fatalf("SyncProduct() expected error")
	}
}

// TestSyncProductImagesSuccess verifies Image success behavior.
func TestSyncProductImagesSuccess(t *testing.T) {
	doer := &doerMock{fn: func(req *http.Request, body []byte) (*http.Response, error) {
		if req.URL.Query().Get("Action") != imageAction {
			t.Fatalf("Action = %q, want %q", req.URL.Query().Get("Action"), imageAction)
		}
		if req.Header.Get("Content-Type") != xmlContentType {
			t.Fatalf("Content-Type = %q, want %q", req.Header.Get("Content-Type"), xmlContentType)
		}
		if !bytes.Contains(body, []byte("<ProductImage>")) {
			t.Fatalf("expected ProductImage in payload")
		}
		return newHTTPResponse(http.StatusOK, http.Header{"Content-Type": []string{"application/xml"}}, `<SuccessResponse/>`), nil
	}}
	client := newClientWithDependencies(
		Config{URL: "https://sellercenter-api.falabella.com", UserID: "user-1", APIKey: "key-1", Version: "1.0", Timeout: 5 * time.Second},
		doer,
		time.Now,
	)

	payload, err := client.SyncProductImages(context.Background(), port.SyncProductImagesRequest{
		SKU:  "SKU-1",
		URLs: []string{"https://cdn.example.com/front.jpg", "https://cdn.example.com/back.jpg"},
	})
	if err != nil {
		t.Fatalf("SyncProductImages() error = %v", err)
	}
	if string(payload) != `<SuccessResponse/>` {
		t.Fatalf("payload = %s", payload)
	}
}

// TestHasBusinessError verifies payload business-error detection behavior.
func TestHasBusinessError(t *testing.T) {
	if !hasBusinessError([]byte(`<ErrorResponse/>`)) {
		t.Fatalf("expected xml error payload to be detected")
	}
	if !hasBusinessError([]byte(`{"ErrorResponse":{"Head":{"ErrorType":"Sender"}}}`)) {
		t.Fatalf("expected json error payload to be detected")
	}
	if hasBusinessError([]byte(`<SuccessResponse/>`)) {
		t.Fatalf("expected success payload to be ignored")
	}
}

// TestRequestQueryParamsForLog verifies query parameter extraction for debug diagnostics.
func TestRequestQueryParamsForLog(t *testing.T) {
	params := requestQueryParamsForLog(&signingContext{
		Canonical: "Action=ProductUpdate&Format=XML&Timestamp=2026-02-18T04%3A37%3A05%2B00%3A00&UserID=coccostoreco%40gmail.com&Version=1.0",
		Signature: "abc123",
	})
	if params["Action"] != "ProductUpdate" {
		t.Fatalf("Action = %q, want %q", params["Action"], "ProductUpdate")
	}
	if params["Format"] != "XML" {
		t.Fatalf("Format = %q, want %q", params["Format"], "XML")
	}
	if params["Timestamp"] != "2026-02-18T04:37:05+00:00" {
		t.Fatalf("Timestamp = %q, want %q", params["Timestamp"], "2026-02-18T04:37:05+00:00")
	}
	if params["UserID"] != "coccostoreco@gmail.com" {
		t.Fatalf("UserID = %q, want %q", params["UserID"], "coccostoreco@gmail.com")
	}
	if params["Version"] != "1.0" {
		t.Fatalf("Version = %q, want %q", params["Version"], "1.0")
	}
	if params["Signature"] != "abc123" {
		t.Fatalf("Signature = %q, want %q", params["Signature"], "abc123")
	}
}

// TestRequestBodyParamsForLogXMLExcludesDescription verifies XML body parameter logging behavior.
func TestRequestBodyParamsForLogXMLExcludesDescription(t *testing.T) {
	payload := []byte(`<?xml version="1.0" encoding="UTF-8"?><Request><Product><SellerSku>SKU-1</SellerSku><Name>Backpack</Name><Description>Should not appear</Description><BusinessUnits>MEN</BusinessUnits></Product></Request>`)
	params := requestBodyParamsForLog(xmlContentType, payload)
	if params["SellerSku"] != "SKU-1" {
		t.Fatalf("SellerSku = %#v, want %q", params["SellerSku"], "SKU-1")
	}
	if params["Name"] != "Backpack" {
		t.Fatalf("Name = %#v, want %q", params["Name"], "Backpack")
	}
	if params["BusinessUnits"] != "MEN" {
		t.Fatalf("BusinessUnits = %#v, want %q", params["BusinessUnits"], "MEN")
	}
	if _, exists := params["Description"]; exists {
		t.Fatalf("Description should be excluded from logged body params")
	}
}

// TestRequestBodyParamsForLogJSONExcludesDescription verifies JSON body parameter logging behavior.
func TestRequestBodyParamsForLogJSONExcludesDescription(t *testing.T) {
	payload := []byte(`{"Request":{"Product":{"SellerSku":"SKU-1","Description":"Should not appear","Name":"Backpack"}}}`)
	params := requestBodyParamsForLog(jsonContentType, payload)
	requestValue, ok := params["Request"].(map[string]any)
	if !ok {
		t.Fatalf("Request should be an object: %#v", params["Request"])
	}
	productValue, ok := requestValue["Product"].(map[string]any)
	if !ok {
		t.Fatalf("Product should be an object: %#v", requestValue["Product"])
	}
	if productValue["SellerSku"] != "SKU-1" {
		t.Fatalf("SellerSku = %#v, want %q", productValue["SellerSku"], "SKU-1")
	}
	if _, exists := productValue["Description"]; exists {
		t.Fatalf("Description should be excluded from logged json body params")
	}
}

// TestResponseContainsSellerSKU verifies SKU existence parsing for GetProducts responses.
func TestResponseContainsSellerSKU(t *testing.T) {
	jsonBody := []byte(`{"SuccessResponse":{"Body":{"Products":{"Product":[{"SellerSku":"SKU-1"}]}}}}`)
	if !responseContainsSellerSKU(jsonBody, "SKU-1") {
		t.Fatalf("expected SKU-1 to be detected in json response")
	}
	xmlBody := []byte(`<?xml version="1.0" encoding="UTF-8"?><SuccessResponse><Body><Products><Product><SellerSku>SKU-2</SellerSku></Product></Products></Body></SuccessResponse>`)
	if !responseContainsSellerSKU(xmlBody, "SKU-2") {
		t.Fatalf("expected SKU-2 to be detected in xml response")
	}
	if responseContainsSellerSKU(jsonBody, "SKU-404") {
		t.Fatalf("expected absent SKU to be reported as not found")
	}
}

// TestGetFeedStatusSuccess verifies FeedStatus retrieval behavior per Falabella Seller Center API.
func TestGetFeedStatusSuccess(t *testing.T) {
	feedXML := `<?xml version="1.0" encoding="UTF-8"?>
<SuccessResponse>
  <Head><RequestAction>FeedStatus</RequestAction></Head>
  <Body><FeedDetail><Feed>feed-abc</Feed><Status>Finished</Status></FeedDetail></Body>
</SuccessResponse>`

	doer := &doerMock{fn: func(req *http.Request, body []byte) (*http.Response, error) {
		if req.URL.Query().Get("Action") != feedStatusAction {
			t.Fatalf("Action = %q, want %q", req.URL.Query().Get("Action"), feedStatusAction)
		}
		if req.URL.Query().Get("FeedID") != "feed-abc" {
			t.Fatalf("FeedID = %q, want %q", req.URL.Query().Get("FeedID"), "feed-abc")
		}
		if req.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", req.Method)
		}
		return newHTTPResponse(http.StatusOK, http.Header{"Content-Type": []string{"application/xml"}}, feedXML), nil
	}}
	client := newClientWithDependencies(
		Config{URL: "https://sellercenter-api.falabella.com", UserID: "user-1", APIKey: "key-1", Version: "1.0", Timeout: 5 * time.Second},
		doer,
		time.Now,
	)

	payload, err := client.GetFeedStatus(context.Background(), "feed-abc")
	if err != nil {
		t.Fatalf("GetFeedStatus() error = %v", err)
	}
	if !strings.Contains(string(payload), "<Status>Finished</Status>") {
		t.Fatalf("payload missing Finished status: %s", payload)
	}
}

// TestGetFeedStatusEmptyID verifies empty feed ID validation behavior.
func TestGetFeedStatusEmptyID(t *testing.T) {
	client := newClientWithDependencies(
		Config{URL: "https://sellercenter-api.falabella.com", UserID: "user-1", APIKey: "key-1", Version: "1.0", Timeout: 5 * time.Second},
		&doerMock{},
		time.Now,
	)

	if _, err := client.GetFeedStatus(context.Background(), "  "); err == nil {
		t.Fatalf("GetFeedStatus(empty) expected error")
	}
}

// TestGetFeedStatusAPIError verifies upstream error propagation behavior.
func TestGetFeedStatusAPIError(t *testing.T) {
	doer := &doerMock{fn: func(req *http.Request, body []byte) (*http.Response, error) {
		return newHTTPResponse(http.StatusBadRequest, http.Header{"Content-Type": []string{"application/xml"}},
			`<ErrorResponse><Head><ErrorCode>1000</ErrorCode><ErrorMessage>Invalid FeedID</ErrorMessage></Head></ErrorResponse>`), nil
	}}
	client := newClientWithDependencies(
		Config{URL: "https://sellercenter-api.falabella.com", UserID: "user-1", APIKey: "key-1", Version: "1.0", Timeout: 5 * time.Second},
		doer,
		time.Now,
	)

	if _, err := client.GetFeedStatus(context.Background(), "bad-feed"); err == nil {
		t.Fatalf("GetFeedStatus() expected error for bad request")
	}
}

// TestSignRequestRawSignatureInput verifies raw-HMAC input with encoded outbound query behavior.
func TestSignRequestRawSignatureInput(t *testing.T) {
	client := newClientWithDependencies(
		Config{URL: "https://sellercenter-api.falabella.com", UserID: "test@example.com", APIKey: "secret", Version: "1.0", Timeout: 5 * time.Second},
		&doerMock{},
		func() time.Time { return time.Date(2026, time.February, 18, 4, 47, 31, 0, time.UTC) },
	)
	req, err := client.newRequest(context.Background(), http.MethodGet, "", nil)
	if err != nil {
		t.Fatalf("newRequest() error = %v", err)
	}

	signing := &signingContext{}
	profile := signingProfile{
		Name:               "xml_raw_lower",
		TimestampLayout:    timestampLayoutNoColon,
		Format:             "XML",
		UppercaseSignature: false,
		SignatureInput:     signatureInputRaw,
	}
	if err := client.signRequest(req, "GetBrands", signing, profile, nil); err != nil {
		t.Fatalf("signRequest() error = %v", err)
	}

	const rawCanonical = "Action=GetBrands&Format=XML&Timestamp=2026-02-18T04:47:31+0000&UserID=test@example.com&Version=1.0"
	if signing.Canonical != rawCanonical {
		t.Fatalf("signing canonical = %q, want %q", signing.Canonical, rawCanonical)
	}

	query := req.URL.Query()
	if query.Get("Timestamp") != "2026-02-18T04:47:31+0000" {
		t.Fatalf("Timestamp = %q, want %q", query.Get("Timestamp"), "2026-02-18T04:47:31+0000")
	}
	if query.Get("UserID") != "test@example.com" {
		t.Fatalf("UserID = %q, want %q", query.Get("UserID"), "test@example.com")
	}
	if !strings.Contains(req.URL.RawQuery, "UserID=test%40example.com") {
		t.Fatalf("expected encoded UserID in raw query: %s", req.URL.RawQuery)
	}
}
