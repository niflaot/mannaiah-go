package http

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	neturl "net/url"
	"testing"

	corehttp "mannaiah/module/core/http"
)

// TestTranscodeImageRouteDisabled verifies disabled transcode endpoint behavior.
func TestTranscodeImageRouteDisabled(t *testing.T) {
	handler, err := NewHandler(&serviceMock{payload: []byte(`{"ok":true}`)}, &productSyncServiceMock{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8311}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(handler.RegisterRoutes)

	request, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/falabella/images/transcoded?src=https%3A%2F%2Fcdn.example.com%2Fa.png", nil)
	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != stdhttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.StatusCode, stdhttp.StatusServiceUnavailable)
	}
}

// TestTranscodeImageRouteSuccess verifies source image transcoding behavior.
func TestTranscodeImageRouteSuccess(t *testing.T) {
	sourceImage := image.NewRGBA(image.Rect(0, 0, 2, 2))
	sourceImage.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	sourceImage.Set(1, 0, color.RGBA{R: 0, G: 255, B: 0, A: 255})
	sourceImage.Set(0, 1, color.RGBA{R: 0, G: 0, B: 255, A: 255})
	sourceImage.Set(1, 1, color.RGBA{R: 255, G: 255, B: 0, A: 255})

	sourceBody := bytes.Buffer{}
	if err := png.Encode(&sourceBody, sourceImage); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}

	sourceServer := httptest.NewServer(stdhttp.HandlerFunc(func(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
		_ = request
		writer.Header().Set("Content-Type", "image/png")
		writer.WriteHeader(stdhttp.StatusOK)
		_, _ = writer.Write(sourceBody.Bytes())
	}))
	defer sourceServer.Close()

	handler, err := NewHandler(&serviceMock{payload: []byte(`{"ok":true}`)}, &productSyncServiceMock{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	handler.SetImageTranscodeConfig(ImageTranscodeConfig{
		Enabled:               true,
		AllowedSourcePrefixes: []string{sourceServer.URL},
	})

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8312}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(handler.RegisterRoutes)

	request, _ := stdhttp.NewRequest(
		stdhttp.MethodGet,
		"/falabella/images/transcoded?src="+neturl.QueryEscape(sourceServer.URL+"/image.png"),
		nil,
	)
	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != stdhttp.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, stdhttp.StatusOK)
	}
	if response.Header.Get("Content-Type") != "image/jpeg" {
		t.Fatalf("Content-Type = %q, want %q", response.Header.Get("Content-Type"), "image/jpeg")
	}

	payload, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		t.Fatalf("ReadAll() error = %v", readErr)
	}
	if _, decodeErr := jpeg.Decode(bytes.NewReader(payload)); decodeErr != nil {
		t.Fatalf("jpeg.Decode() error = %v", decodeErr)
	}
}

// TestTranscodeImageRouteForbiddenSource verifies source-prefix validation behavior.
func TestTranscodeImageRouteForbiddenSource(t *testing.T) {
	handler, err := NewHandler(&serviceMock{payload: []byte(`{"ok":true}`)}, &productSyncServiceMock{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	handler.SetImageTranscodeConfig(ImageTranscodeConfig{
		Enabled:               true,
		AllowedSourcePrefixes: []string{"https://cdn.allowed.example.com"},
	})

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8313}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(handler.RegisterRoutes)

	request, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/falabella/images/transcoded?src=https%3A%2F%2Fevil.example.com%2Fimg.png", nil)
	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != stdhttp.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.StatusCode, stdhttp.StatusForbidden)
	}
}

// TestIsTranscodeSourceAllowed verifies source-prefix host and path matching behavior.
func TestIsTranscodeSourceAllowed(t *testing.T) {
	if !isTranscodeSourceAllowed(
		"https://cdn.example.com/assets/products/a.webp",
		[]string{"https://cdn.example.com/assets"},
	) {
		t.Fatalf("expected source to be allowed")
	}
	if isTranscodeSourceAllowed(
		"https://cdn.example.com.evil.com/assets/products/a.webp",
		[]string{"https://cdn.example.com/assets"},
	) {
		t.Fatalf("expected source host bypass to be rejected")
	}
	if isTranscodeSourceAllowed(
		"https://cdn.example.com/private/a.webp",
		[]string{"https://cdn.example.com/assets"},
	) {
		t.Fatalf("expected source path outside allowed prefix to be rejected")
	}
}
