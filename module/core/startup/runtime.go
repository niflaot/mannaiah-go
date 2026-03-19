package startup

import (
	"errors"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gofiber/fiber/v2"
	fiberswagger "github.com/gofiber/swagger"
	corehttp "mannaiah/module/core/http"
	"mannaiah/module/core/swagger"
)

var (
	// ErrNilServer is returned when a nil HTTP server is provided.
	ErrNilServer = errors.New("startup server must not be nil")
	// ErrNilDocument is returned when a nil swagger document is provided.
	ErrNilDocument = errors.New("startup swagger document must not be nil")
)

// Runtime defines startup composition helpers shared across modules.
type Runtime struct {
	// server defines HTTP server runtime.
	server *corehttp.Server
	// document defines centralized OpenAPI aggregation document.
	document *swagger.Document
}

// Loader defines bootstrap hooks exposed by runtime to modules.
type Loader interface {
	// RegisterRoutes registers module route handlers.
	RegisterRoutes(register func(router corehttp.Router))
	// AddOpenAPISpec merges module OpenAPI specifications.
	AddOpenAPISpec(spec *openapi3.T) error
}

var (
	// _ ensures Runtime satisfies module loader contracts.
	_ Loader = (*Runtime)(nil)
)

// NewRuntime creates a startup runtime over HTTP server and swagger document dependencies.
func NewRuntime(server *corehttp.Server, document *swagger.Document) (*Runtime, error) {
	if server == nil {
		return nil, ErrNilServer
	}
	if document == nil {
		return nil, ErrNilDocument
	}

	return &Runtime{server: server, document: document}, nil
}

// RegisterRoutes registers route handlers into the HTTP server.
func (r *Runtime) RegisterRoutes(register func(router corehttp.Router)) {
	r.server.RegisterRoutes(register)
}

// AddOpenAPISpec merges module OpenAPI specs into the centralized document.
func (r *Runtime) AddOpenAPISpec(spec *openapi3.T) error {
	return r.document.Merge(spec)
}

// ExposeOpenAPI registers a route that serves aggregated OpenAPI documentation.
func (r *Runtime) ExposeOpenAPI(path string) {
	r.RegisterRoutes(func(router corehttp.Router) {
		swagger.RegisterRoute(router, path, r.document.Build())
	})
}

// ExposeOpenAPIUI registers a route that serves a Swagger UI HTML page.
func (r *Runtime) ExposeOpenAPIUI(path string, specPath string, title string) {
	docsPath := resolveDocsPath(path)
	docsWildcardPath := docsPath + "/*"
	docsSpecPath := resolveDocsSpecPath(specPath)
	docsTitle := resolveDocsTitle(title)

	r.server.Register(func(app *fiber.App) {
		app.Get(docsPath, func(ctx *fiber.Ctx) error {
			return ctx.Redirect(joinDocsIndexPath(docsPath), fiber.StatusMovedPermanently)
		})
		app.Get(docsWildcardPath, fiberswagger.New(fiberswagger.Config{
			URL:   docsSpecPath,
			Title: docsTitle,
		}))
	})
}

// CoreSpec returns core-level OpenAPI specs for startup-managed endpoints.
func CoreSpec() *openapi3.T {
	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:   "Core Startup API",
			Version: "2.4.2",
		},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/status", &openapi3.PathItem{
				Get: statusOperation(),
			}),
			openapi3.WithPath("/metrics", &openapi3.PathItem{
				Get: metricsOperation(),
			}),
			openapi3.WithPath("/openapi.json", &openapi3.PathItem{
				Get: openapiOperation(),
			}),
			openapi3.WithPath("/docs", &openapi3.PathItem{
				Get: docsOperation(),
			}),
		),
		Tags: openapi3.Tags{
			&openapi3.Tag{Name: "Status"},
		},
	}
}

// statusOperation defines the OpenAPI operation for status probing.
func statusOperation() *openapi3.Operation {
	statusSchema := openapi3.NewStringSchema()
	statusSchema.Example = "ok"

	return &openapi3.Operation{
		Summary:     "Get application status",
		OperationID: "StatusController_getStatus",
		Tags:        []string{"Status"},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, &openapi3.ResponseRef{
				Value: openapi3.NewResponse().
					WithDescription("The application is running successfully.").
					WithContent(openapi3.Content{
						"application/json": &openapi3.MediaType{
							Schema: &openapi3.SchemaRef{
								Value: openapi3.NewObjectSchema().
									WithProperty("status", statusSchema),
							},
						},
					}),
			}),
		),
	}
}

// metricsOperation defines the OpenAPI operation for Prometheus metrics exposure.
func metricsOperation() *openapi3.Operation {
	return &openapi3.Operation{
		Summary:     "Get Prometheus metrics",
		OperationID: "StatusController_getMetrics",
		Tags:        []string{"Status"},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, &openapi3.ResponseRef{
				Value: openapi3.NewResponse().WithDescription("Return Prometheus metrics payload."),
			}),
		),
	}
}

// openapiOperation defines the OpenAPI operation for aggregated spec exposure.
func openapiOperation() *openapi3.Operation {
	return &openapi3.Operation{
		Summary:     "Get aggregated OpenAPI specification",
		OperationID: "StatusController_getOpenAPI",
		Tags:        []string{"Status"},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, &openapi3.ResponseRef{
				Value: openapi3.NewResponse().WithDescription("Return aggregated API specification."),
			}),
		),
	}
}

// docsOperation defines the OpenAPI operation for docs UI exposure.
func docsOperation() *openapi3.Operation {
	return &openapi3.Operation{
		Summary:     "Get Swagger UI documentation page",
		OperationID: "StatusController_getDocs",
		Tags:        []string{"Status"},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, &openapi3.ResponseRef{
				Value: openapi3.NewResponse().WithDescription("Return Swagger UI page."),
			}),
		),
	}
}

// resolveDocsPath normalizes docs base paths and applies defaults when needed.
func resolveDocsPath(path string) string {
	resolved := strings.TrimSpace(path)
	if resolved == "" {
		return "/docs"
	}
	if !strings.HasPrefix(resolved, "/") {
		resolved = "/" + resolved
	}
	if len(resolved) > 1 {
		resolved = strings.TrimRight(resolved, "/")
	}

	return resolved
}

// resolveDocsSpecPath normalizes docs spec source paths and applies defaults when needed.
func resolveDocsSpecPath(specPath string) string {
	resolved := strings.TrimSpace(specPath)
	if resolved == "" {
		return "/openapi.json"
	}
	if strings.HasPrefix(resolved, "/") || isAbsoluteURL(resolved) {
		return resolved
	}

	return "/" + resolved
}

// resolveDocsTitle normalizes docs page titles and applies defaults when needed.
func resolveDocsTitle(title string) string {
	resolved := strings.TrimSpace(title)
	if resolved == "" {
		return "API Docs"
	}

	return resolved
}

// joinDocsIndexPath resolves the docs index page path under the docs base path.
func joinDocsIndexPath(docsPath string) string {
	if docsPath == "/" {
		return "/index.html"
	}

	base := strings.TrimRight(docsPath, "/")
	if base == "" {
		base = "/docs"
	}

	return base + "/index.html"
}

// isAbsoluteURL reports whether docs spec sources are absolute URLs.
func isAbsoluteURL(value string) bool {
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}
