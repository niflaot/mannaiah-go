package e2e_test

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"testing"

	corehttp "mannaiah/module/core/http"
	"mannaiah/module/falabella"
)

// TestFalabellaImageTranscodeEndpointE2E verifies the Falabella image transcode endpoint returns JPEG payloads.
func TestFalabellaImageTranscodeEndpointE2E(t *testing.T) {
	tracer := newStepTracer(t)

	tracer.Step("start source image server")
	sourceImage := image.NewRGBA(image.Rect(0, 0, 3, 3))
	sourceImage.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	sourceImage.Set(1, 1, color.RGBA{R: 0, G: 255, B: 0, A: 255})
	sourceImage.Set(2, 2, color.RGBA{R: 0, G: 0, B: 255, A: 255})
	sourceBody := bytes.Buffer{}
	if err := png.Encode(&sourceBody, sourceImage); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}
	sourceServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_ = request
		writer.Header().Set("Content-Type", "image/png")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(sourceBody.Bytes())
	}))
	defer sourceServer.Close()

	tracer.Step("initialize falabella module with transcode endpoint enabled")
	module, err := falabella.New(falabella.Config{
		URL:                                  sourceServer.URL,
		UserID:                               "e2e@test.com",
		APIKey:                               "e2e-key-abc",
		RequestTimeoutMS:                     2000,
		ValidationTimeoutMS:                  1000,
		ProductImageTranscodeEnabled:         true,
		ProductImageTranscodeAllowedPrefixes: sourceServer.URL,
		ProductImageTranscodePublicBaseURL:   "http://localhost:8080",
		ProductImageTranscodeTimeoutMS:       5000,
	}, tracer.logger)
	if err != nil {
		t.Fatalf("falabella.New() error = %v", err)
	}

	tracer.Step("initialize http server")
	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8504}, tracer.logger)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(module.RegisterRoutes)

	tracer.Step("request transcoded image")
	req, _ := http.NewRequest(http.MethodGet, "/falabella/images/transcoded?src="+neturl.QueryEscape(sourceServer.URL+"/sample.png"), nil)
	resp, testErr := server.App().Test(req)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /falabella/images/transcoded status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if resp.Header.Get("Content-Type") != "image/jpeg" {
		t.Fatalf("Content-Type = %q, want %q", resp.Header.Get("Content-Type"), "image/jpeg")
	}
	payload := bytes.Buffer{}
	if _, readErr := payload.ReadFrom(resp.Body); readErr != nil {
		t.Fatalf("ReadFrom(response body) error = %v", readErr)
	}
	if _, decodeErr := jpeg.Decode(bytes.NewReader(payload.Bytes())); decodeErr != nil {
		t.Fatalf("jpeg.Decode() error = %v", decodeErr)
	}

	tracer.AssertStepCount(4)
}
