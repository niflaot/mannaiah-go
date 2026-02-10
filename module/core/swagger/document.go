package swagger

import (
	"errors"
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	corehttp "mannaiah/module/core/http"
)

var (
	// ErrNilSpec is returned when a nil OpenAPI spec is merged.
	ErrNilSpec = errors.New("swagger spec must not be nil")
)

// Info defines OpenAPI document metadata.
type Info struct {
	// Title defines API title values.
	Title string
	// Version defines API version values.
	Version string
	// Description defines API description values.
	Description string
}

// Document defines an aggregated OpenAPI document.
type Document struct {
	// document defines the typed OpenAPI document root.
	document *openapi3.T
}

// NewDocument creates an empty OpenAPI aggregation document.
func NewDocument(info Info) *Document {
	components := openapi3.NewComponents()
	components.Schemas = openapi3.Schemas{}
	components.Parameters = openapi3.ParametersMap{}
	components.Headers = openapi3.Headers{}
	components.RequestBodies = openapi3.RequestBodies{}
	components.Responses = openapi3.ResponseBodies{}
	components.SecuritySchemes = openapi3.SecuritySchemes{}
	components.Examples = openapi3.Examples{}
	components.Links = openapi3.Links{}
	components.Callbacks = openapi3.Callbacks{}

	return &Document{
		document: &openapi3.T{
			OpenAPI: "3.0.3",
			Info: &openapi3.Info{
				Title:       info.Title,
				Version:     info.Version,
				Description: info.Description,
			},
			Paths:      openapi3.NewPaths(),
			Components: &components,
			Tags:       openapi3.Tags{},
		},
	}
}

// Merge adds a module OpenAPI spec into the aggregated document.
func (d *Document) Merge(spec *openapi3.T) error {
	if spec == nil {
		return ErrNilSpec
	}

	if err := d.mergePaths(spec.Paths); err != nil {
		return err
	}
	if err := d.mergeComponents(spec.Components); err != nil {
		return err
	}
	if err := d.mergeTags(spec.Tags); err != nil {
		return err
	}

	return nil
}

// Build returns the aggregated OpenAPI document.
func (d *Document) Build() *openapi3.T {
	return d.document
}

// RegisterRoute registers an endpoint that serves the aggregated OpenAPI document.
func RegisterRoute(router corehttp.Router, path string, document *openapi3.T) {
	router.Get(path, func(ctx corehttp.Context) error {
		return ctx.Status(200).JSON(document)
	})
}

// RegisterUIRoute registers an endpoint that serves a Swagger UI HTML page.
func RegisterUIRoute(router corehttp.Router, path string, specPath string, title string) {
	resolvedSpecPath := strings.TrimSpace(specPath)
	if resolvedSpecPath == "" {
		resolvedSpecPath = "/openapi.json"
	}

	resolvedTitle := strings.TrimSpace(title)
	if resolvedTitle == "" {
		resolvedTitle = "API Docs"
	}

	page := buildSwaggerUIPage(resolvedSpecPath, resolvedTitle)
	router.Get(path, func(ctx corehttp.Context) error {
		return ctx.Status(200).
			Set("Content-Type", "text/html; charset=utf-8").
			SendString(page)
	})
}

// mergePaths merges path/method documentation objects.
func (d *Document) mergePaths(paths *openapi3.Paths) error {
	if paths == nil {
		return nil
	}

	for path, sourcePathItem := range paths.Map() {
		if sourcePathItem == nil {
			return fmt.Errorf("invalid operations object for path %q", path)
		}

		targetPathItem := d.document.Paths.Value(path)
		if targetPathItem == nil {
			d.document.Paths.Set(path, clonePathItem(sourcePathItem))
			continue
		}

		sourceOperations := sourcePathItem.Operations()
		for method, operation := range sourceOperations {
			httpMethod := strings.ToUpper(strings.TrimSpace(method))
			if targetPathItem.GetOperation(httpMethod) != nil {
				return fmt.Errorf("duplicate swagger operation %s %s", strings.ToLower(httpMethod), path)
			}
			targetPathItem.SetOperation(httpMethod, operation)
		}
	}

	return nil
}

// mergeComponents merges OpenAPI component sets.
func (d *Document) mergeComponents(source *openapi3.Components) error {
	if source == nil {
		return nil
	}

	if err := mergeSchemas(d.document.Components.Schemas, source.Schemas); err != nil {
		return err
	}
	if err := mergeParameters(d.document.Components.Parameters, source.Parameters); err != nil {
		return err
	}
	if err := mergeHeaders(d.document.Components.Headers, source.Headers); err != nil {
		return err
	}
	if err := mergeRequestBodies(d.document.Components.RequestBodies, source.RequestBodies); err != nil {
		return err
	}
	if err := mergeResponses(d.document.Components.Responses, source.Responses); err != nil {
		return err
	}
	if err := mergeSecuritySchemes(d.document.Components.SecuritySchemes, source.SecuritySchemes); err != nil {
		return err
	}
	if err := mergeExamples(d.document.Components.Examples, source.Examples); err != nil {
		return err
	}
	if err := mergeLinks(d.document.Components.Links, source.Links); err != nil {
		return err
	}
	if err := mergeCallbacks(d.document.Components.Callbacks, source.Callbacks); err != nil {
		return err
	}

	return nil
}

// mergeTags merges OpenAPI tags by unique tag name.
func (d *Document) mergeTags(tags openapi3.Tags) error {
	if len(tags) == 0 {
		return nil
	}

	existingTags := map[string]struct{}{}
	for _, existingTag := range d.document.Tags {
		if existingTag == nil {
			continue
		}
		existingTags[strings.TrimSpace(existingTag.Name)] = struct{}{}
	}

	for _, sourceTag := range tags {
		if sourceTag == nil {
			return fmt.Errorf("invalid swagger tag entry")
		}

		name := strings.TrimSpace(sourceTag.Name)
		if name == "" {
			continue
		}
		if _, exists := existingTags[name]; exists {
			continue
		}

		d.document.Tags = append(d.document.Tags, cloneTag(sourceTag))
		existingTags[name] = struct{}{}
	}

	return nil
}

// clonePathItem creates a detached path item value for merged path storage.
func clonePathItem(source *openapi3.PathItem) *openapi3.PathItem {
	if source == nil {
		return nil
	}

	cloned := *source
	if len(source.Parameters) > 0 {
		cloned.Parameters = make(openapi3.Parameters, len(source.Parameters))
		copy(cloned.Parameters, source.Parameters)
	}
	if len(source.Servers) > 0 {
		cloned.Servers = append(openapi3.Servers{}, source.Servers...)
	}

	return &cloned
}

// cloneTag creates a detached tag value for merged tag storage.
func cloneTag(source *openapi3.Tag) *openapi3.Tag {
	if source == nil {
		return nil
	}

	cloned := *source
	return &cloned
}

// buildSwaggerUIPage builds a static Swagger UI HTML page for a spec endpoint.
func buildSwaggerUIPage(specPath string, title string) string {
	escapedTitle := html.EscapeString(title)
	quotedSpecPath := strconv.Quote(specPath)

	return fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>%s</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>
    html, body { margin: 0; padding: 0; background: #f5f5f5; }
    #swagger-ui { max-width: 1200px; margin: 0 auto; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
  <script>
  window.onload = function () {
    window.ui = SwaggerUIBundle({
      url: %s,
      dom_id: '#swagger-ui',
      presets: [
        SwaggerUIBundle.presets.apis,
        SwaggerUIStandalonePreset
      ],
      layout: "BaseLayout"
    });
  };
  </script>
</body>
</html>
`, escapedTitle, quotedSpecPath)
}

// mergeSchemas merges schema components while preventing duplicate keys.
func mergeSchemas(target, source openapi3.Schemas) error {
	for key, value := range source {
		if _, exists := target[key]; exists {
			return fmt.Errorf("duplicate swagger component schemas.%s", key)
		}
		target[key] = value
	}

	return nil
}

// mergeParameters merges parameter components while preventing duplicate keys.
func mergeParameters(target, source openapi3.ParametersMap) error {
	for key, value := range source {
		if _, exists := target[key]; exists {
			return fmt.Errorf("duplicate swagger component parameters.%s", key)
		}
		target[key] = value
	}

	return nil
}

// mergeHeaders merges header components while preventing duplicate keys.
func mergeHeaders(target, source openapi3.Headers) error {
	for key, value := range source {
		if _, exists := target[key]; exists {
			return fmt.Errorf("duplicate swagger component headers.%s", key)
		}
		target[key] = value
	}

	return nil
}

// mergeRequestBodies merges request body components while preventing duplicate keys.
func mergeRequestBodies(target, source openapi3.RequestBodies) error {
	for key, value := range source {
		if _, exists := target[key]; exists {
			return fmt.Errorf("duplicate swagger component requestBodies.%s", key)
		}
		target[key] = value
	}

	return nil
}

// mergeResponses merges response components while preventing duplicate keys.
func mergeResponses(target, source openapi3.ResponseBodies) error {
	for key, value := range source {
		if _, exists := target[key]; exists {
			return fmt.Errorf("duplicate swagger component responses.%s", key)
		}
		target[key] = value
	}

	return nil
}

// mergeSecuritySchemes merges security scheme components while preventing duplicate keys.
func mergeSecuritySchemes(target, source openapi3.SecuritySchemes) error {
	for key, value := range source {
		if _, exists := target[key]; exists {
			return fmt.Errorf("duplicate swagger component securitySchemes.%s", key)
		}
		target[key] = value
	}

	return nil
}

// mergeExamples merges example components while preventing duplicate keys.
func mergeExamples(target, source openapi3.Examples) error {
	for key, value := range source {
		if _, exists := target[key]; exists {
			return fmt.Errorf("duplicate swagger component examples.%s", key)
		}
		target[key] = value
	}

	return nil
}

// mergeLinks merges link components while preventing duplicate keys.
func mergeLinks(target, source openapi3.Links) error {
	for key, value := range source {
		if _, exists := target[key]; exists {
			return fmt.Errorf("duplicate swagger component links.%s", key)
		}
		target[key] = value
	}

	return nil
}

// mergeCallbacks merges callback components while preventing duplicate keys.
func mergeCallbacks(target, source openapi3.Callbacks) error {
	for key, value := range source {
		if _, exists := target[key]; exists {
			return fmt.Errorf("duplicate swagger component callbacks.%s", key)
		}
		target[key] = value
	}

	return nil
}
