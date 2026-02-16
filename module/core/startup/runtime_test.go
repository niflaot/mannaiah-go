package startup

import (
	errorspkg "errors"
	"io"
	stdhttp "net/http"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	corehttp "mannaiah/module/core/http"
	"mannaiah/module/core/swagger"
)

// TestNewRuntimeValidation verifies runtime constructor dependency validation.
func TestNewRuntimeValidation(t *testing.T) {
	doc := swagger.NewDocument(swagger.Info{Title: "Mannaiah", Version: "1.0.0"})
	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8111}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}

	if _, err := NewRuntime(nil, doc); !errorspkg.Is(err, ErrNilServer) {
		t.Fatalf("NewRuntime(nil, doc) error = %v, want ErrNilServer", err)
	}
	if _, err := NewRuntime(server, nil); !errorspkg.Is(err, ErrNilDocument) {
		t.Fatalf("NewRuntime(server, nil) error = %v, want ErrNilDocument", err)
	}
}

// TestRuntimeRegisterRoutesAddSpecAndExposeOpenAPI verifies runtime route registration and OpenAPI exposure behavior.
func TestRuntimeRegisterRoutesAddSpecAndExposeOpenAPI(t *testing.T) {
	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8112}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	doc := swagger.NewDocument(swagger.Info{Title: "Mannaiah", Version: "1.0.0"})
	runtime, err := NewRuntime(server, doc)
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}

	if err := runtime.AddOpenAPISpec(&openapi3.T{
		OpenAPI: "3.0.3",
		Info:    &openapi3.Info{Title: "Test", Version: "1.0.0"},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/hello", &openapi3.PathItem{
				Get: &openapi3.Operation{
					Summary: "hello",
					Responses: openapi3.NewResponses(
						openapi3.WithStatus(200, &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription("ok")}),
					),
				},
			}),
		),
	}); err != nil {
		t.Fatalf("AddOpenAPISpec() error = %v", err)
	}
	runtime.RegisterRoutes(func(router corehttp.Router) {
		router.Get("/hello", func(ctx corehttp.Context) error {
			return ctx.Status(200).JSON(map[string]string{"status": "ok"})
		})
	})
	runtime.ExposeOpenAPI("/openapi.json")
	runtime.ExposeOpenAPIUI("/docs", "/openapi.json", "Mannaiah Docs")

	helloReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/hello", nil)
	helloResp, helloErr := server.App().Test(helloReq)
	if helloErr != nil {
		t.Fatalf("App().Test(/hello) error = %v", helloErr)
	}
	if helloResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("/hello status = %d, want %d", helloResp.StatusCode, stdhttp.StatusOK)
	}

	specReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/openapi.json", nil)
	specResp, specErr := server.App().Test(specReq)
	if specErr != nil {
		t.Fatalf("App().Test(/openapi.json) error = %v", specErr)
	}
	if specResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("/openapi.json status = %d, want %d", specResp.StatusCode, stdhttp.StatusOK)
	}

	docsRedirectReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/docs", nil)
	docsRedirectResp, docsRedirectErr := server.App().Test(docsRedirectReq)
	if docsRedirectErr != nil {
		t.Fatalf("App().Test(/docs) error = %v", docsRedirectErr)
	}
	if docsRedirectResp.StatusCode != stdhttp.StatusMovedPermanently {
		t.Fatalf("/docs status = %d, want %d", docsRedirectResp.StatusCode, stdhttp.StatusMovedPermanently)
	}
	if docsRedirectResp.Header.Get("Location") != "/docs/index.html" {
		t.Fatalf("/docs location = %q, want %q", docsRedirectResp.Header.Get("Location"), "/docs/index.html")
	}

	docsReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/docs/index.html", nil)
	docsResp, docsErr := server.App().Test(docsReq)
	if docsErr != nil {
		t.Fatalf("App().Test(/docs/index.html) error = %v", docsErr)
	}
	if docsResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("/docs/index.html status = %d, want %d", docsResp.StatusCode, stdhttp.StatusOK)
	}
	docsBody, readErr := io.ReadAll(docsResp.Body)
	if readErr != nil {
		t.Fatalf("ReadAll(/docs) error = %v", readErr)
	}
	if !strings.Contains(string(docsBody), "SwaggerUIBundle") {
		t.Fatalf("expected SwaggerUIBundle in docs html")
	}
}

// TestCoreSpec verifies startup core spec generation.
func TestCoreSpec(t *testing.T) {
	spec := CoreSpec()
	if spec.Paths.Value("/status") == nil {
		t.Fatalf("expected /status spec")
	}
	if spec.Paths.Value("/openapi.json") == nil {
		t.Fatalf("expected /openapi.json spec")
	}
	if spec.Paths.Value("/docs") == nil {
		t.Fatalf("expected /docs spec")
	}
}

// TestResolveDocsPath verifies docs path normalization behavior.
func TestResolveDocsPath(t *testing.T) {
	if value := resolveDocsPath(""); value != "/docs" {
		t.Fatalf("resolveDocsPath(\"\") = %q, want %q", value, "/docs")
	}
	if value := resolveDocsPath("docs"); value != "/docs" {
		t.Fatalf("resolveDocsPath(\"docs\") = %q, want %q", value, "/docs")
	}
	if value := resolveDocsPath("/docs/"); value != "/docs" {
		t.Fatalf("resolveDocsPath(\"/docs/\") = %q, want %q", value, "/docs")
	}
}

// TestResolveDocsSpecPath verifies docs spec path normalization behavior.
func TestResolveDocsSpecPath(t *testing.T) {
	if value := resolveDocsSpecPath(""); value != "/openapi.json" {
		t.Fatalf("resolveDocsSpecPath(\"\") = %q, want %q", value, "/openapi.json")
	}
	if value := resolveDocsSpecPath(" /spec.json "); value != "/spec.json" {
		t.Fatalf("resolveDocsSpecPath(\" /spec.json \") = %q, want %q", value, "/spec.json")
	}
	if value := resolveDocsSpecPath("spec.json"); value != "/spec.json" {
		t.Fatalf("resolveDocsSpecPath(\"spec.json\") = %q, want %q", value, "/spec.json")
	}
	if value := resolveDocsSpecPath("https://example.com/spec.json"); value != "https://example.com/spec.json" {
		t.Fatalf("resolveDocsSpecPath(\"https://example.com/spec.json\") = %q, want %q", value, "https://example.com/spec.json")
	}
}

// TestResolveDocsTitle verifies docs title normalization behavior.
func TestResolveDocsTitle(t *testing.T) {
	if value := resolveDocsTitle(""); value != "API Docs" {
		t.Fatalf("resolveDocsTitle(\"\") = %q, want %q", value, "API Docs")
	}
	if value := resolveDocsTitle(" Custom Docs "); value != "Custom Docs" {
		t.Fatalf("resolveDocsTitle(\" Custom Docs \") = %q, want %q", value, "Custom Docs")
	}
}

// TestJoinDocsIndexPath verifies docs index-path generation behavior.
func TestJoinDocsIndexPath(t *testing.T) {
	if value := joinDocsIndexPath("/docs"); value != "/docs/index.html" {
		t.Fatalf("joinDocsIndexPath(\"/docs\") = %q, want %q", value, "/docs/index.html")
	}
	if value := joinDocsIndexPath("/docs/"); value != "/docs/index.html" {
		t.Fatalf("joinDocsIndexPath(\"/docs/\") = %q, want %q", value, "/docs/index.html")
	}
	if value := joinDocsIndexPath("/"); value != "/index.html" {
		t.Fatalf("joinDocsIndexPath(\"/\") = %q, want %q", value, "/index.html")
	}
}

// TestIsAbsoluteURL verifies absolute URL detection behavior.
func TestIsAbsoluteURL(t *testing.T) {
	if !isAbsoluteURL("http://example.com/spec.json") {
		t.Fatalf("expected http URL to be absolute")
	}
	if !isAbsoluteURL("https://example.com/spec.json") {
		t.Fatalf("expected https URL to be absolute")
	}
	if isAbsoluteURL("/openapi.json") {
		t.Fatalf("expected path URL to be non-absolute")
	}
}
