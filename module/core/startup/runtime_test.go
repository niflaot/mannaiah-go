package startup

import (
	errorspkg "errors"
	stdhttp "net/http"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	corehttp "mannaiah/module/core/http"
	"mannaiah/module/core/swagger"
)

// TestNewRuntimeValidation verifies runtime constructor dependency validation.
func TestNewRuntimeValidation(t *testing.T) {
	doc := swagger.NewDocument(swagger.Info{Title: "Mannaiah", Version: "0.0.1"})
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
	doc := swagger.NewDocument(swagger.Info{Title: "Mannaiah", Version: "0.0.1"})
	runtime, err := NewRuntime(server, doc)
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}

	if err := runtime.AddOpenAPISpec(&openapi3.T{
		OpenAPI: "3.0.3",
		Info:    &openapi3.Info{Title: "Test", Version: "0.0.1"},
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
}
