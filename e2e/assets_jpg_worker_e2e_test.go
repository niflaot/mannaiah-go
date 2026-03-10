package e2e_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"strings"
	"testing"

	assetsapplication "mannaiah/module/assets/application"
)

// TestAssetsJPGWorkerE2E verifies tagged asset JPG conversion and object replacement behavior.
func TestAssetsJPGWorkerE2E(t *testing.T) {
	harness := newContactsE2EHarness(t)
	defer harness.Close(t)

	assetsCreateToken := harness.SignToken(t, "assets:create")
	assetsReadToken := harness.SignToken(t, "assets:read")

	harness.tracer.Step("upload tagged asset eligible for jpg worker")
	status, payload := doAssetUploadRequest(t, harness, assetsCreateToken, "morral_traveler_camo_negro_frente.webp", samplePNGFixture(t), map[string]string{
		"name": "Traveler",
		"tags": `[{"name":"marketplaces","color":"#ff0000"}]`,
	})
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", status, http.StatusCreated)
	}
	assetID, _ := payload["_id"].(string)
	if assetID == "" {
		t.Fatalf("expected asset id in response")
	}
	oldKey, _ := payload["key"].(string)
	if !strings.HasSuffix(oldKey, ".webp") {
		t.Fatalf("oldKey = %q, want .webp suffix", oldKey)
	}

	harness.tracer.Step("run jpg worker with marketplaces tag filter")
	result, runErr := harness.assetsModule.Service().RunJPGWorker(context.Background(), assetsapplication.JPGWorkerCommand{
		Tags:      []string{"marketplaces"},
		BatchSize: 10,
	})
	if runErr != nil {
		t.Fatalf("RunJPGWorker() error = %v", runErr)
	}
	if result.Converted != 1 || result.Failed != 0 {
		t.Fatalf("result = %#v, want converted=1 failed=0", result)
	}

	harness.tracer.Step("fetch updated asset and verify jpg metadata")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/assets/"+assetID, assetsReadToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	newKey, _ := payload["key"].(string)
	if !strings.HasSuffix(newKey, ".jpg") {
		t.Fatalf("newKey = %q, want .jpg suffix", newKey)
	}
	if payload["mimeType"] != "image/jpeg" {
		t.Fatalf("payload.mimeType = %v, want %q", payload["mimeType"], "image/jpeg")
	}
	if payload["originalName"] != "morral_traveler_camo_negro_frente.jpg" {
		t.Fatalf("payload.originalName = %v, want %q", payload["originalName"], "morral_traveler_camo_negro_frente.jpg")
	}

	harness.tracer.Step("verify in-memory storage replaced old object key")
	oldExists, oldExistsErr := harness.assetStorage.Exists(context.Background(), oldKey)
	if oldExistsErr != nil {
		t.Fatalf("assetStorage.Exists(old) error = %v", oldExistsErr)
	}
	if oldExists {
		t.Fatalf("old object key still exists: %q", oldKey)
	}
	newExists, newExistsErr := harness.assetStorage.Exists(context.Background(), newKey)
	if newExistsErr != nil {
		t.Fatalf("assetStorage.Exists(new) error = %v", newExistsErr)
	}
	if !newExists {
		t.Fatalf("new object key missing: %q", newKey)
	}

	harness.tracer.Step("assert e2e trace logs")
	harness.tracer.AssertStepCount(5)
}

// samplePNGFixture defines a deterministic image fixture for worker e2e tests.
func samplePNGFixture(t *testing.T) []byte {
	t.Helper()

	payload := image.NewRGBA(image.Rect(0, 0, 2, 2))
	payload.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	payload.Set(1, 0, color.RGBA{R: 0, G: 255, B: 0, A: 255})
	payload.Set(0, 1, color.RGBA{R: 0, G: 0, B: 255, A: 255})
	payload.Set(1, 1, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	buffer := bytes.Buffer{}
	if err := png.Encode(&buffer, payload); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}
	return buffer.Bytes()
}
