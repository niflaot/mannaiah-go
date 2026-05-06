package http

import (
	"context"
	"mime/multipart"
	"strings"
	"testing"

	corehttp "mannaiah/module/core/http"
)

// launchTestContext defines the minimal request context used to exercise the app-launch landing page.
type launchTestContext struct {
	// queryValues defines query-string inputs exposed to the handler.
	queryValues map[string]string
	// headers captures response header values written by the handler.
	headers map[string]string
	// statusCode captures the final response status code.
	statusCode int
	// body captures the plain-text response payload.
	body string
}

// Context returns a background request context for the test handler invocation.
func (c *launchTestContext) Context() context.Context {
	return context.Background()
}

// GetHeader reads request header values for test requests.
func (c *launchTestContext) GetHeader(key string, defaultValue ...string) string {
	if value, ok := c.headers[key]; ok {
		return value
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

// Queries returns the full query-string map for the test request.
func (c *launchTestContext) Queries() map[string]string {
	return c.queryValues
}

// Status records the response status code for the test response.
func (c *launchTestContext) Status(code int) corehttp.Context {
	c.statusCode = code
	return c
}

// JSON satisfies the context interface for tests that do not emit JSON.
func (c *launchTestContext) JSON(body any) error {
	return nil
}

// SendString captures the text response payload.
func (c *launchTestContext) SendString(body string) error {
	c.body = body
	return nil
}

// SendStatus records a response status without a body payload.
func (c *launchTestContext) SendStatus(status int) error {
	c.statusCode = status
	return nil
}

// SetHeader records response header values.
func (c *launchTestContext) SetHeader(key string, value string) {
	if c.headers == nil {
		c.headers = map[string]string{}
	}
	c.headers[key] = value
}

// SendBytes captures binary response payloads as strings for assertions.
func (c *launchTestContext) SendBytes(body []byte) error {
	c.body = string(body)
	return nil
}

// Params returns empty path parameters because the launch page has none.
func (c *launchTestContext) Params(key string, defaultValue ...string) string {
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

// Query returns one query-string value for the test request.
func (c *launchTestContext) Query(key string, defaultValue ...string) string {
	if value, ok := c.queryValues[key]; ok {
		return value
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

// BodyParser satisfies the context interface for tests that do not decode bodies.
func (c *launchTestContext) BodyParser(out any) error {
	return nil
}

// Body returns an empty request body for the launch-page test.
func (c *launchTestContext) Body() []byte {
	return nil
}

// FormFile satisfies the context interface for tests that do not parse multipart payloads.
func (c *launchTestContext) FormFile(key string) (*multipart.FileHeader, error) {
	return nil, nil
}

// FormValue returns empty multipart form values for the launch-page test.
func (c *launchTestContext) FormValue(key string, defaultValue ...string) string {
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

// Locals satisfies the context interface for tests that do not use request locals.
func (c *launchTestContext) Locals(key string, value ...any) any {
	return nil
}

// TestAppLaunchRouteReturnsLandingPage verifies Shopify App URL launches render a deterministic landing page.
func TestAppLaunchRouteReturnsLandingPage(t *testing.T) {
	handler := &Handler{}
	requestContext := &launchTestContext{
		queryValues: map[string]string{"shop": "2axh5c-b1.myshopify.com", "installed": "1"},
		headers:     map[string]string{},
	}

	if err := handler.appLaunch(requestContext); err != nil {
		t.Fatalf("appLaunch() error = %v", err)
	}
	if requestContext.statusCode != 200 {
		t.Fatalf("appLaunch() status = %d, want %d", requestContext.statusCode, 200)
	}
	if contentType := requestContext.headers["Content-Type"]; !strings.Contains(contentType, "text/html") {
		t.Fatalf("appLaunch() content type = %q, want html", contentType)
	}
	if !strings.Contains(requestContext.body, "2axh5c-b1.myshopify.com") {
		t.Fatalf("appLaunch() body missing shop domain: %q", requestContext.body)
	}
	if !strings.Contains(requestContext.body, "Instalacion de Shopify completada") {
		t.Fatalf("appLaunch() body missing install headline: %q", requestContext.body)
	}
	if !strings.Contains(requestContext.body, "Admin UI extension") {
		t.Fatalf("appLaunch() body missing next-step guidance: %q", requestContext.body)
	}
	if !strings.Contains(requestContext.body, "/shopify/app") {
		t.Fatalf("appLaunch() body missing recommended route: %q", requestContext.body)
	}
	if !strings.Contains(requestContext.body, "/docs") {
		t.Fatalf("appLaunch() body missing docs link: %q", requestContext.body)
	}
}
