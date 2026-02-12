package e2e_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"testing"
)

// TestAssetsAndProductsIntegrationE2E verifies asset CRUD and product-gallery integration behavior.
func TestAssetsAndProductsIntegrationE2E(t *testing.T) {
	harness := newContactsE2EHarness(t)
	defer harness.Close(t)

	harness.tracer.Step("request asset upload without authorization header")
	status, payload := doAssetUploadRequest(t, harness, "", "image.png", []byte("payload"), "Asset One")
	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", status, http.StatusUnauthorized)
	}
	if payload["message"] != "unauthorized" {
		t.Fatalf("payload.message = %v, want %q", payload["message"], "unauthorized")
	}

	assetsCreateToken := harness.SignToken(t, "assets:create")
	assetsReadToken := harness.SignToken(t, "assets:read")
	assetsUpdateToken := harness.SignToken(t, "assets:update")
	assetsDeleteToken := harness.SignToken(t, "assets:delete")
	productsManageToken := harness.SignToken(t, "products:manage")

	harness.tracer.Step("upload asset with create scope")
	status, payload = doAssetUploadRequest(t, harness, assetsCreateToken, "image.png", []byte("payload"), "Hero")
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", status, http.StatusCreated)
	}
	assetID, _ := payload["_id"].(string)
	if assetID == "" {
		t.Fatalf("expected asset id in response")
	}
	harness.AwaitAssetCreatedEvent(t)

	harness.tracer.Step("list assets with read scope")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/assets?page=1&limit=10", assetsReadToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	data, ok := payload["data"].([]any)
	if !ok || len(data) != 1 {
		t.Fatalf("payload.data = %v, want one asset", payload["data"])
	}

	harness.tracer.Step("update asset name")
	status, payload = harness.DoJSONRequest(t, http.MethodPatch, "/assets/"+assetID, assetsUpdateToken, []byte(`{"name":"Hero Updated"}`))
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if payload["name"] != "Hero Updated" {
		t.Fatalf("payload.name = %v, want %q", payload["name"], "Hero Updated")
	}
	harness.AwaitAssetUpdatedEvent(t)

	harness.tracer.Step("create product referencing existing asset")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/products", productsManageToken, []byte(`{"sku":"SKU-ASSET-1","gallery":[{"assetId":"`+assetID+`","isMain":true}]}`))
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", status, http.StatusCreated)
	}

	harness.tracer.Step("reject product referencing missing asset")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/products", productsManageToken, []byte(`{"sku":"SKU-ASSET-2","gallery":[{"assetId":"missing-asset","isMain":true}]}`))
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", status, http.StatusBadRequest)
	}
	if payload["message"] != "invalid_product_asset_reference" {
		t.Fatalf("payload.message = %v, want %q", payload["message"], "invalid_product_asset_reference")
	}

	harness.tracer.Step("delete asset with delete scope")
	status, payload = harness.DoJSONRequest(t, http.MethodDelete, "/assets/"+assetID, assetsDeleteToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if payload["status"] != "deleted" {
		t.Fatalf("payload.status = %v, want %q", payload["status"], "deleted")
	}
	harness.AwaitAssetDeletedEvent(t)

	harness.tracer.Step("assert e2e trace logs")
	harness.tracer.AssertStepCount(7)
}

// doAssetUploadRequest uploads asset payloads as multipart/form-data.
func doAssetUploadRequest(t *testing.T, harness *contactsE2EHarness, token string, fileName string, content []byte, name string) (int, map[string]any) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	filePart, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := filePart.Write(content); err != nil {
		t.Fatalf("filePart.Write() error = %v", err)
	}
	if name != "" {
		if err := writer.WriteField("name", name); err != nil {
			t.Fatalf("WriteField(name) error = %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}

	request, err := http.NewRequest(http.MethodPost, "/assets", bytes.NewReader(body.Bytes()))
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	response, err := harness.server.App().Test(request)
	if err != nil {
		t.Fatalf("App().Test() error = %v", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	result := map[string]any{}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("json.NewDecoder().Decode() error = %v", err)
	}

	return response.StatusCode, result
}
