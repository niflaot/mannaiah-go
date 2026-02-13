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
	status, payload := doAssetUploadRequest(t, harness, "", "image.png", []byte("payload"), map[string]string{"name": "Asset One"})
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

	harness.tracer.Step("create folder for logical organization")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/assets/folders", assetsCreateToken, []byte(`{"name":"Catalog","tags":[{"name":"hero","color":"#ff0000"}]}`))
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", status, http.StatusCreated)
	}
	parentFolderID, _ := payload["_id"].(string)
	if parentFolderID == "" {
		t.Fatalf("expected parent folder id in response")
	}

	harness.tracer.Step("create child folder inside parent folder")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/assets/folders", assetsCreateToken, []byte(`{"name":"Catalog Child","parentFolderId":"`+parentFolderID+`","tags":[{"name":"child","color":"#00aa00"}]}`))
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", status, http.StatusCreated)
	}
	childFolderID, _ := payload["_id"].(string)
	if childFolderID == "" {
		t.Fatalf("expected child folder id in response")
	}

	harness.tracer.Step("list nested folders by parent filter")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/assets/folders?page=1&limit=10&parentFolderId="+parentFolderID, assetsReadToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	folderData, ok := payload["data"].([]any)
	if !ok || len(folderData) != 1 {
		t.Fatalf("payload.data = %v, want one child folder", payload["data"])
	}
	folderRow, ok := folderData[0].(map[string]any)
	if !ok || folderRow["_id"] != childFolderID {
		t.Fatalf("folder row = %v, want child folder id %q", folderData[0], childFolderID)
	}

	harness.tracer.Step("upload asset with create scope")
	status, payload = doAssetUploadRequest(t, harness, assetsCreateToken, "image.png", []byte("payload"), map[string]string{
		"name":     "Hero",
		"folderId": childFolderID,
		"tags":     `[{"name":"cover","color":"#00aa11"}]`,
		"metadata": `{"alt":"homepage hero"}`,
	})
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", status, http.StatusCreated)
	}
	assetID, _ := payload["_id"].(string)
	if assetID == "" {
		t.Fatalf("expected asset id in response")
	}
	harness.AwaitAssetCreatedEvent(t)

	harness.tracer.Step("verify uploaded asset persisted folder assignment and metadata")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/assets/"+assetID, assetsReadToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if payload["folderId"] != childFolderID {
		t.Fatalf("payload.folderId = %v, want %q", payload["folderId"], childFolderID)
	}

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

	harness.tracer.Step("rename and soft-delete folder while detaching assets")
	status, payload = harness.DoJSONRequest(t, http.MethodPatch, "/assets/folders/"+parentFolderID, assetsUpdateToken, []byte(`{"name":"Catalog 2026","tags":[{"name":"catalog","color":"#ffaa00"}]}`))
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}

	harness.tracer.Step("reject cyclic parent move")
	status, payload = harness.DoJSONRequest(t, http.MethodPatch, "/assets/folders/"+parentFolderID, assetsUpdateToken, []byte(`{"parentFolderId":"`+childFolderID+`"}`))
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", status, http.StatusBadRequest)
	}
	if payload["message"] != "invalid_folder_parent" {
		t.Fatalf("payload.message = %v, want %q", payload["message"], "invalid_folder_parent")
	}

	harness.tracer.Step("delete parent folder recursively and detach assets")
	status, payload = harness.DoJSONRequest(t, http.MethodDelete, "/assets/folders/"+parentFolderID, assetsDeleteToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/assets/"+assetID, assetsReadToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if payload["folderId"] != nil && payload["folderId"] != "" {
		t.Fatalf("payload.folderId = %v, want empty after folder deletion", payload["folderId"])
	}

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
	harness.tracer.AssertStepCount(13)
}

// doAssetUploadRequest uploads asset payloads as multipart/form-data.
func doAssetUploadRequest(t *testing.T, harness *contactsE2EHarness, token string, fileName string, content []byte, fields map[string]string) (int, map[string]any) {
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
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("WriteField(%s) error = %v", key, err)
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
