package swagger

import (
	errorspkg "errors"
	"io"
	stdhttp "net/http"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	corehttp "mannaiah/module/core/http"
)

// TestNewDocumentBuild verifies base document creation behavior.
func TestNewDocumentBuild(t *testing.T) {
	document := NewDocument(Info{Title: "Mannaiah", Version: "0.0.1", Description: "API"}).Build()
	if document.OpenAPI != "3.0.3" {
		t.Fatalf("openapi = %v, want %q", document.OpenAPI, "3.0.3")
	}
}

// TestMergeRejectsNilSpec verifies nil-spec validation behavior.
func TestMergeRejectsNilSpec(t *testing.T) {
	err := NewDocument(Info{}).Merge(nil)
	if !errorspkg.Is(err, ErrNilSpec) {
		t.Fatalf("Merge() error = %v, want ErrNilSpec", err)
	}
}

// TestMergeAggregatesPathsComponentsAndTags verifies spec aggregation behavior.
func TestMergeAggregatesPathsComponentsAndTags(t *testing.T) {
	doc := NewDocument(Info{Title: "Mannaiah", Version: "0.0.1"})
	spec := &openapi3.T{
		OpenAPI: "3.0.3",
		Info:    &openapi3.Info{Title: "Contacts", Version: "0.0.1"},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/contacts", &openapi3.PathItem{
				Get: &openapi3.Operation{
					Summary: "List contacts",
					Responses: openapi3.NewResponses(
						openapi3.WithStatus(200, &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription("ok")}),
					),
				},
			}),
		),
		Components: &openapi3.Components{
			Schemas: openapi3.Schemas{
				"Contact": {Value: openapi3.NewObjectSchema()},
			},
		},
		Tags: openapi3.Tags{
			&openapi3.Tag{Name: "contacts"},
		},
	}

	if err := doc.Merge(spec); err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	result := doc.Build()
	if result.Paths.Value("/contacts") == nil {
		t.Fatalf("expected contacts path")
	}
	if result.Components == nil || result.Components.Schemas["Contact"] == nil {
		t.Fatalf("expected Contact schema")
	}
	if len(result.Tags) != 1 {
		t.Fatalf("tags len = %d, want %d", len(result.Tags), 1)
	}
}

// TestMergeRejectsDuplicateOperation verifies duplicate path/method detection.
func TestMergeRejectsDuplicateOperation(t *testing.T) {
	doc := NewDocument(Info{})
	first := &openapi3.T{
		OpenAPI: "3.0.3",
		Info:    &openapi3.Info{Title: "A", Version: "0.0.1"},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/contacts", &openapi3.PathItem{
				Get: &openapi3.Operation{
					Summary:   "A",
					Responses: openapi3.NewResponses(openapi3.WithStatus(200, responseRefWithDescription("ok"))),
				},
			}),
		),
	}
	second := &openapi3.T{
		OpenAPI: "3.0.3",
		Info:    &openapi3.Info{Title: "B", Version: "0.0.1"},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/contacts", &openapi3.PathItem{
				Get: &openapi3.Operation{
					Summary:   "B",
					Responses: openapi3.NewResponses(openapi3.WithStatus(200, responseRefWithDescription("ok"))),
				},
			}),
		),
	}

	if err := doc.Merge(first); err != nil {
		t.Fatalf("Merge(first) error = %v", err)
	}
	if err := doc.Merge(second); err == nil {
		t.Fatalf("expected duplicate operation error")
	}
}

// TestMergeRejectsDuplicateComponent verifies duplicate component detection.
func TestMergeRejectsDuplicateComponent(t *testing.T) {
	doc := NewDocument(Info{})
	first := &openapi3.T{
		OpenAPI: "3.0.3",
		Info:    &openapi3.Info{Title: "A", Version: "0.0.1"},
		Components: &openapi3.Components{
			Schemas: openapi3.Schemas{
				"Contact": {Value: openapi3.NewObjectSchema()},
			},
		},
	}
	second := &openapi3.T{
		OpenAPI: "3.0.3",
		Info:    &openapi3.Info{Title: "B", Version: "0.0.1"},
		Components: &openapi3.Components{
			Schemas: openapi3.Schemas{
				"Contact": {Value: openapi3.NewObjectSchema()},
			},
		},
	}

	if err := doc.Merge(first); err != nil {
		t.Fatalf("Merge(first) error = %v", err)
	}
	if err := doc.Merge(second); err == nil {
		t.Fatalf("expected duplicate component error")
	}
}

// TestMergeRejectsInvalidStructures verifies malformed spec branches.
func TestMergeRejectsInvalidStructures(t *testing.T) {
	doc := NewDocument(Info{})
	invalidPaths := openapi3.NewPaths()
	invalidPaths.Set("/contacts", nil)

	if err := doc.Merge(&openapi3.T{
		OpenAPI: "3.0.3",
		Info:    &openapi3.Info{Title: "Bad Paths", Version: "0.0.1"},
		Paths:   invalidPaths,
	}); err == nil {
		t.Fatalf("expected invalid operations error")
	}
	if err := doc.Merge(&openapi3.T{
		OpenAPI: "3.0.3",
		Info:    &openapi3.Info{Title: "Bad Tags", Version: "0.0.1"},
		Tags: openapi3.Tags{
			nil,
		},
	}); err == nil {
		t.Fatalf("expected invalid tag error")
	}
}

// TestMergeDeduplicatesTags verifies duplicate tag suppression behavior.
func TestMergeDeduplicatesTags(t *testing.T) {
	doc := NewDocument(Info{})
	spec := &openapi3.T{
		OpenAPI: "3.0.3",
		Info:    &openapi3.Info{Title: "Tag Test", Version: "0.0.1"},
		Tags: openapi3.Tags{
			&openapi3.Tag{Name: "contacts"},
			&openapi3.Tag{Name: "contacts"},
		},
	}
	if err := doc.Merge(spec); err != nil {
		t.Fatalf("Merge() error = %v", err)
	}
	if len(doc.Build().Tags) != 1 {
		t.Fatalf("expected unique tags")
	}
}

// TestRegisterRoute serves aggregated documents through the HTTP router.
func TestRegisterRoute(t *testing.T) {
	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8110}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}

	doc := &openapi3.T{OpenAPI: "3.0.3", Info: &openapi3.Info{Title: "Mannaiah", Version: "0.0.1"}, Paths: openapi3.NewPaths()}
	server.RegisterRoutes(func(router corehttp.Router) {
		RegisterRoute(router, "/openapi.json", doc)
	})

	req, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/openapi.json", nil)
	resp, testErr := server.App().Test(req)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if resp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, stdhttp.StatusOK)
	}
}

// TestRegisterUIRoute serves Swagger UI HTML through the HTTP router.
func TestRegisterUIRoute(t *testing.T) {
	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8113}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}

	server.RegisterRoutes(func(router corehttp.Router) {
		RegisterUIRoute(router, "/docs", "/openapi.json", "Mannaiah Docs")
	})

	req, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/docs", nil)
	resp, testErr := server.App().Test(req)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if resp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, stdhttp.StatusOK)
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		t.Fatalf("content-type = %q, want text/html", resp.Header.Get("Content-Type"))
	}

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		t.Fatalf("ReadAll() error = %v", readErr)
	}
	page := string(body)
	if !strings.Contains(page, "SwaggerUIBundle") {
		t.Fatalf("expected SwaggerUIBundle script in docs page")
	}
	if !strings.Contains(page, "/openapi.json") {
		t.Fatalf("expected spec path in docs page")
	}
	if !strings.Contains(page, "Mannaiah Docs") {
		t.Fatalf("expected title in docs page")
	}
}

// TestRegisterUIRouteDefaults verifies docs-page defaults when inputs are empty.
func TestRegisterUIRouteDefaults(t *testing.T) {
	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8114}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}

	server.RegisterRoutes(func(router corehttp.Router) {
		RegisterUIRoute(router, "/docs", "", "")
	})

	req, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/docs", nil)
	resp, testErr := server.App().Test(req)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if resp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, stdhttp.StatusOK)
	}

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		t.Fatalf("ReadAll() error = %v", readErr)
	}
	page := string(body)
	if !strings.Contains(page, "/openapi.json") {
		t.Fatalf("expected default openapi path in docs page")
	}
	if !strings.Contains(page, "API Docs") {
		t.Fatalf("expected default title in docs page")
	}
}

// TestMergeAggregatesAllComponentTypes verifies component merge support for all OpenAPI component maps.
func TestMergeAggregatesAllComponentTypes(t *testing.T) {
	doc := NewDocument(Info{})
	if err := doc.Merge(specWithAllComponentTypes("A")); err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	components := doc.Build().Components
	if components == nil {
		t.Fatalf("expected components")
	}
	if components.Schemas["ASchema"] == nil {
		t.Fatalf("expected schema component")
	}
	if components.Parameters["AParameter"] == nil {
		t.Fatalf("expected parameter component")
	}
	if components.Headers["AHeader"] == nil {
		t.Fatalf("expected header component")
	}
	if components.RequestBodies["ARequestBody"] == nil {
		t.Fatalf("expected request body component")
	}
	if components.Responses["AResponse"] == nil {
		t.Fatalf("expected response component")
	}
	if components.SecuritySchemes["ASecurityScheme"] == nil {
		t.Fatalf("expected security scheme component")
	}
	if components.Examples["AExample"] == nil {
		t.Fatalf("expected example component")
	}
	if components.Links["ALink"] == nil {
		t.Fatalf("expected link component")
	}
	if components.Callbacks["ACallback"] == nil {
		t.Fatalf("expected callback component")
	}
}

// TestMergeRejectsDuplicateComponentTypes verifies duplicate detection for every supported component map.
func TestMergeRejectsDuplicateComponentTypes(t *testing.T) {
	tests := []struct {
		name      string
		component string
		spec      *openapi3.T
	}{
		{
			name:      "schemas",
			component: "schemas.Contact",
			spec: &openapi3.T{
				OpenAPI: "3.0.3",
				Info:    &openapi3.Info{Title: "Schema", Version: "0.0.1"},
				Components: &openapi3.Components{
					Schemas: openapi3.Schemas{"Contact": {Value: openapi3.NewObjectSchema()}},
				},
			},
		},
		{
			name:      "parameters",
			component: "parameters.ContactID",
			spec: &openapi3.T{
				OpenAPI: "3.0.3",
				Info:    &openapi3.Info{Title: "Parameter", Version: "0.0.1"},
				Components: &openapi3.Components{
					Parameters: openapi3.ParametersMap{"ContactID": {Value: openapi3.NewPathParameter("id")}},
				},
			},
		},
		{
			name:      "headers",
			component: "headers.Trace",
			spec: &openapi3.T{
				OpenAPI: "3.0.3",
				Info:    &openapi3.Info{Title: "Header", Version: "0.0.1"},
				Components: &openapi3.Components{
					Headers: openapi3.Headers{"Trace": {Value: &openapi3.Header{}}},
				},
			},
		},
		{
			name:      "requestBodies",
			component: "requestBodies.ContactCreate",
			spec: &openapi3.T{
				OpenAPI: "3.0.3",
				Info:    &openapi3.Info{Title: "RequestBody", Version: "0.0.1"},
				Components: &openapi3.Components{
					RequestBodies: openapi3.RequestBodies{"ContactCreate": {Value: openapi3.NewRequestBody()}},
				},
			},
		},
		{
			name:      "responses",
			component: "responses.ContactOk",
			spec: &openapi3.T{
				OpenAPI: "3.0.3",
				Info:    &openapi3.Info{Title: "Response", Version: "0.0.1"},
				Components: &openapi3.Components{
					Responses: openapi3.ResponseBodies{"ContactOk": responseRefWithDescription("ok")},
				},
			},
		},
		{
			name:      "securitySchemes",
			component: "securitySchemes.Bearer",
			spec: &openapi3.T{
				OpenAPI: "3.0.3",
				Info:    &openapi3.Info{Title: "Security", Version: "0.0.1"},
				Components: &openapi3.Components{
					SecuritySchemes: openapi3.SecuritySchemes{"Bearer": {Value: &openapi3.SecurityScheme{Type: "http"}}},
				},
			},
		},
		{
			name:      "examples",
			component: "examples.Contact",
			spec: &openapi3.T{
				OpenAPI: "3.0.3",
				Info:    &openapi3.Info{Title: "Example", Version: "0.0.1"},
				Components: &openapi3.Components{
					Examples: openapi3.Examples{"Contact": {Value: &openapi3.Example{Value: "example"}}},
				},
			},
		},
		{
			name:      "links",
			component: "links.Contact",
			spec: &openapi3.T{
				OpenAPI: "3.0.3",
				Info:    &openapi3.Info{Title: "Link", Version: "0.0.1"},
				Components: &openapi3.Components{
					Links: openapi3.Links{"Contact": {Value: &openapi3.Link{}}},
				},
			},
		},
		{
			name:      "callbacks",
			component: "callbacks.Contact",
			spec: &openapi3.T{
				OpenAPI: "3.0.3",
				Info:    &openapi3.Info{Title: "Callback", Version: "0.0.1"},
				Components: &openapi3.Components{
					Callbacks: openapi3.Callbacks{"Contact": {Value: openapi3.NewCallback(openapi3.WithCallback("{$request.body#/callbackUrl}", &openapi3.PathItem{}))}},
				},
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			doc := NewDocument(Info{})
			if err := doc.Merge(testCase.spec); err != nil {
				t.Fatalf("Merge(first) error = %v", err)
			}

			err := doc.Merge(testCase.spec)
			if err == nil {
				t.Fatalf("expected duplicate component error")
			}
			if !strings.Contains(err.Error(), testCase.component) {
				t.Fatalf("error = %v, want component %q", err, testCase.component)
			}
		})
	}
}

// TestClonePathItem verifies cloning behavior for nil and populated path item values.
func TestClonePathItem(t *testing.T) {
	if value := clonePathItem(nil); value != nil {
		t.Fatalf("clonePathItem(nil) = %v, want nil", value)
	}

	source := &openapi3.PathItem{
		Get: &openapi3.Operation{
			Responses: openapi3.NewResponses(
				openapi3.WithStatus(200, responseRefWithDescription("ok")),
			),
		},
		Parameters: openapi3.Parameters{
			{Value: openapi3.NewQueryParameter("page").WithSchema(openapi3.NewIntegerSchema())},
		},
		Servers: openapi3.Servers{
			{URL: "http://localhost"},
		},
	}

	cloned := clonePathItem(source)
	if cloned == nil {
		t.Fatalf("expected cloned path item")
	}
	if cloned == source {
		t.Fatalf("expected detached path item pointer")
	}
	if &cloned.Parameters[0] == &source.Parameters[0] {
		t.Fatalf("expected detached parameters slice")
	}
	if &cloned.Servers[0] == &source.Servers[0] {
		t.Fatalf("expected detached servers slice")
	}
}

// TestCloneTag verifies cloning behavior for nil and populated tag values.
func TestCloneTag(t *testing.T) {
	if value := cloneTag(nil); value != nil {
		t.Fatalf("cloneTag(nil) = %v, want nil", value)
	}

	source := &openapi3.Tag{Name: "contacts", Description: "contacts tag"}
	cloned := cloneTag(source)
	if cloned == nil {
		t.Fatalf("expected cloned tag")
	}
	if cloned == source {
		t.Fatalf("expected detached tag pointer")
	}
}

// specWithAllComponentTypes builds a single spec carrying all supported OpenAPI component maps.
func specWithAllComponentTypes(prefix string) *openapi3.T {
	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info:    &openapi3.Info{Title: "All Components", Version: "0.0.1"},
		Components: &openapi3.Components{
			Schemas: openapi3.Schemas{
				prefix + "Schema": {Value: openapi3.NewObjectSchema()},
			},
			Parameters: openapi3.ParametersMap{
				prefix + "Parameter": {Value: openapi3.NewQueryParameter("page").WithSchema(openapi3.NewIntegerSchema())},
			},
			Headers: openapi3.Headers{
				prefix + "Header": {Value: &openapi3.Header{}},
			},
			RequestBodies: openapi3.RequestBodies{
				prefix + "RequestBody": {Value: openapi3.NewRequestBody()},
			},
			Responses: openapi3.ResponseBodies{
				prefix + "Response": responseRefWithDescription("ok"),
			},
			SecuritySchemes: openapi3.SecuritySchemes{
				prefix + "SecurityScheme": {Value: &openapi3.SecurityScheme{Type: "http"}},
			},
			Examples: openapi3.Examples{
				prefix + "Example": {Value: &openapi3.Example{Value: "example"}},
			},
			Links: openapi3.Links{
				prefix + "Link": {Value: &openapi3.Link{}},
			},
			Callbacks: openapi3.Callbacks{
				prefix + "Callback": {Value: openapi3.NewCallback(openapi3.WithCallback("{$request.body#/callbackUrl}", &openapi3.PathItem{}))},
			},
		},
	}
}

// responseRefWithDescription builds a response ref with a plain text description.
func responseRefWithDescription(description string) *openapi3.ResponseRef {
	return &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription(description)}
}
