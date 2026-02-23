package http

import (
	"bytes"
	"io"
	stdhttp "net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// TestRegisterAndMount verifies route registration for app-level and grouped handlers.
func TestRegisterAndMount(t *testing.T) {
	server, err := New(Config{Host: "127.0.0.1", Port: 8081}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	server.Register(func(app *fiber.App) {
		app.Get("/health", func(ctx *fiber.Ctx) error {
			return ctx.Status(fiber.StatusOK).SendString("ok")
		})
	})
	server.Mount("/v1", func(router fiber.Router) {
		router.Get("/status", func(ctx *fiber.Ctx) error {
			return ctx.Status(fiber.StatusOK).SendString("ready")
		})
	})

	healthReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/health", nil)
	healthResp, testErr := server.App().Test(healthReq)
	if testErr != nil {
		t.Fatalf("App().Test(/health) error = %v", testErr)
	}
	if healthResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("/health status = %d, want %d", healthResp.StatusCode, stdhttp.StatusOK)
	}

	statusReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/v1/status", nil)
	statusResp, statusErr := server.App().Test(statusReq)
	if statusErr != nil {
		t.Fatalf("App().Test(/v1/status) error = %v", statusErr)
	}
	if statusResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("/v1/status status = %d, want %d", statusResp.StatusCode, stdhttp.StatusOK)
	}
}

// TestRegisterRoutesAndMountRoutes verifies abstract router registration for all HTTP methods.
func TestRegisterRoutesAndMountRoutes(t *testing.T) {
	server, err := New(Config{Host: "127.0.0.1", Port: 8087}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	server.RegisterRoutes(func(router Router) {
		router.Get("/abstract/get/:name", func(ctx Context) error {
			ctx.Locals("name", ctx.Params("name"))
			name, _ := ctx.Locals("name").(string)
			return ctx.Status(stdhttp.StatusOK).JSON(fiber.Map{
				"name":  name,
				"query": ctx.Query("q", "none"),
				"auth":  ctx.GetHeader("Authorization", ""),
			})
		})
		router.Post("/abstract/post", func(ctx Context) error {
			var payload map[string]string
			if err := ctx.BodyParser(&payload); err != nil {
				return ctx.Status(stdhttp.StatusBadRequest).SendString("invalid")
			}

			return ctx.Status(stdhttp.StatusCreated).SendString(payload["name"])
		})
		router.Put("/abstract/put", func(ctx Context) error {
			return ctx.SendStatus(stdhttp.StatusNoContent)
		})
		router.Patch("/abstract/patch", nil)
		router.Delete("/abstract/delete", func(ctx Context) error {
			return ctx.SendStatus(stdhttp.StatusNoContent)
		})
	})

	server.MountRoutes("/v2", func(router Router) {
		router.Get("/status", func(ctx Context) error {
			return ctx.SendString("mounted")
		})
	})

	getReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/abstract/get/john?q=x", nil)
	getReq.Header.Set("Authorization", "Bearer test")
	getResp, getErr := server.App().Test(getReq)
	if getErr != nil {
		t.Fatalf("GET /abstract/get error = %v", getErr)
	}
	if getResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("GET /abstract/get status = %d, want %d", getResp.StatusCode, stdhttp.StatusOK)
	}
	getBody, getBodyErr := io.ReadAll(getResp.Body)
	if getBodyErr != nil {
		t.Fatalf("ReadAll() error = %v", getBodyErr)
	}
	if !strings.Contains(string(getBody), "Bearer test") {
		t.Fatalf("GET /abstract/get body = %q, want authorization header echoed", string(getBody))
	}

	postReq, _ := stdhttp.NewRequest(stdhttp.MethodPost, "/abstract/post", strings.NewReader(`{"name":"doe"}`))
	postReq.Header.Set("Content-Type", "application/json")
	postResp, postErr := server.App().Test(postReq)
	if postErr != nil {
		t.Fatalf("POST /abstract/post error = %v", postErr)
	}
	if postResp.StatusCode != stdhttp.StatusCreated {
		t.Fatalf("POST /abstract/post status = %d, want %d", postResp.StatusCode, stdhttp.StatusCreated)
	}
	postBody, bodyErr := io.ReadAll(postResp.Body)
	if bodyErr != nil {
		t.Fatalf("ReadAll() error = %v", bodyErr)
	}
	if strings.TrimSpace(string(postBody)) != "doe" {
		t.Fatalf("POST /abstract/post body = %q, want %q", strings.TrimSpace(string(postBody)), "doe")
	}

	putReq, _ := stdhttp.NewRequest(stdhttp.MethodPut, "/abstract/put", nil)
	putResp, putErr := server.App().Test(putReq)
	if putErr != nil {
		t.Fatalf("PUT /abstract/put error = %v", putErr)
	}
	if putResp.StatusCode != stdhttp.StatusNoContent {
		t.Fatalf("PUT /abstract/put status = %d, want %d", putResp.StatusCode, stdhttp.StatusNoContent)
	}

	patchReq, _ := stdhttp.NewRequest(stdhttp.MethodPatch, "/abstract/patch", nil)
	patchResp, patchErr := server.App().Test(patchReq)
	if patchErr != nil {
		t.Fatalf("PATCH /abstract/patch error = %v", patchErr)
	}
	if patchResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("PATCH /abstract/patch status = %d, want %d", patchResp.StatusCode, stdhttp.StatusOK)
	}

	deleteReq, _ := stdhttp.NewRequest(stdhttp.MethodDelete, "/abstract/delete", nil)
	deleteResp, deleteErr := server.App().Test(deleteReq)
	if deleteErr != nil {
		t.Fatalf("DELETE /abstract/delete error = %v", deleteErr)
	}
	if deleteResp.StatusCode != stdhttp.StatusNoContent {
		t.Fatalf("DELETE /abstract/delete status = %d, want %d", deleteResp.StatusCode, stdhttp.StatusNoContent)
	}

	mountedReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/v2/status", nil)
	mountedResp, mountedErr := server.App().Test(mountedReq)
	if mountedErr != nil {
		t.Fatalf("GET /v2/status error = %v", mountedErr)
	}
	if mountedResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("GET /v2/status status = %d, want %d", mountedResp.StatusCode, stdhttp.StatusOK)
	}
}

// TestRegisterAndMountNilCallbacks verifies nil callbacks are safely ignored.
func TestRegisterAndMountNilCallbacks(t *testing.T) {
	server, err := New(Config{Host: "127.0.0.1", Port: 8082}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	server.Register(nil)
	server.Mount("/v1", nil)
	server.RegisterRoutes(nil)
	server.MountRoutes("/v2", nil)
}

// TestZapFiberMiddlewareLogs verifies zapfiber request logs are emitted through provided logger.
func TestZapFiberMiddlewareLogs(t *testing.T) {
	var output bytes.Buffer
	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(&output), zapcore.DebugLevel)
	logger := zap.New(core)

	server, err := New(Config{Host: "127.0.0.1", Port: 8083}, logger)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	server.Register(func(app *fiber.App) {
		app.Get("/logged", func(ctx *fiber.Ctx) error {
			return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
		})
	})

	req, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/logged", nil)
	resp, testErr := server.App().Test(req)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if resp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, stdhttp.StatusOK)
	}

	payload := output.String()
	if !strings.Contains(payload, "/logged") {
		t.Fatalf("expected zapfiber log payload to include route, got %q", payload)
	}
}

// TestZapFiberMiddlewareSkipsWooLookupNoise verifies noisy WooCommerce lookup requests are excluded from access logs.
func TestZapFiberMiddlewareSkipsWooLookupNoise(t *testing.T) {
	var output bytes.Buffer
	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(&output), zapcore.DebugLevel)
	logger := zap.New(core)

	server, err := New(Config{Host: "127.0.0.1", Port: 8084}, logger)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	server.Register(func(app *fiber.App) {
		app.Get("/contacts", func(ctx *fiber.Ctx) error {
			return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
		})
		app.Get("/orders", func(ctx *fiber.Ctx) error {
			return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
		})
		app.Get("/contacts/:id", func(ctx *fiber.Ctx) error {
			return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
		})
		app.Get("/products", func(ctx *fiber.Ctx) error {
			return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
		})
		app.Get("/metrics", func(ctx *fiber.Ctx) error {
			return ctx.Status(fiber.StatusOK).SendString("metrics")
		})
		app.Get("/status", func(ctx *fiber.Ctx) error {
			return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
		})
		app.Get("/openapi.json", func(ctx *fiber.Ctx) error {
			return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"openapi": "3.0.3"})
		})
		app.Get("/docs", func(ctx *fiber.Ctx) error {
			return ctx.Status(fiber.StatusOK).SendString("docs")
		})
	})

	skippedContactsReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/contacts?email=test%40example.com&limit=1&page=1", nil)
	skippedContactsResp, skippedContactsErr := server.App().Test(skippedContactsReq)
	if skippedContactsErr != nil {
		t.Fatalf("App().Test() contacts error = %v", skippedContactsErr)
	}
	if skippedContactsResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("contacts status = %d, want %d", skippedContactsResp.StatusCode, stdhttp.StatusOK)
	}

	skippedOrdersReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/orders?identifier=1024112&realm=woocommerce&limit=1&page=1", nil)
	skippedOrdersResp, skippedOrdersErr := server.App().Test(skippedOrdersReq)
	if skippedOrdersErr != nil {
		t.Fatalf("App().Test() orders error = %v", skippedOrdersErr)
	}
	if skippedOrdersResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("orders status = %d, want %d", skippedOrdersResp.StatusCode, stdhttp.StatusOK)
	}

	skippedProductsReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/products", nil)
	skippedProductsResp, skippedProductsErr := server.App().Test(skippedProductsReq)
	if skippedProductsErr != nil {
		t.Fatalf("App().Test() products error = %v", skippedProductsErr)
	}
	if skippedProductsResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("products status = %d, want %d", skippedProductsResp.StatusCode, stdhttp.StatusOK)
	}

	skippedOrdersPageReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/orders?page=1&limit=10", nil)
	skippedOrdersPageResp, skippedOrdersPageErr := server.App().Test(skippedOrdersPageReq)
	if skippedOrdersPageErr != nil {
		t.Fatalf("App().Test() paginated orders error = %v", skippedOrdersPageErr)
	}
	if skippedOrdersPageResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("paginated orders status = %d, want %d", skippedOrdersPageResp.StatusCode, stdhttp.StatusOK)
	}

	skippedContactByIDReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/contacts/c480b5cbb751bd434e63b5f9f344f527", nil)
	skippedContactByIDResp, skippedContactByIDErr := server.App().Test(skippedContactByIDReq)
	if skippedContactByIDErr != nil {
		t.Fatalf("App().Test() contact by id error = %v", skippedContactByIDErr)
	}
	if skippedContactByIDResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("contact by id status = %d, want %d", skippedContactByIDResp.StatusCode, stdhttp.StatusOK)
	}

	skippedMetricsReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/metrics", nil)
	skippedMetricsResp, skippedMetricsErr := server.App().Test(skippedMetricsReq)
	if skippedMetricsErr != nil {
		t.Fatalf("App().Test() metrics error = %v", skippedMetricsErr)
	}
	if skippedMetricsResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("metrics status = %d, want %d", skippedMetricsResp.StatusCode, stdhttp.StatusOK)
	}

	skippedStatusReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/status", nil)
	skippedStatusResp, skippedStatusErr := server.App().Test(skippedStatusReq)
	if skippedStatusErr != nil {
		t.Fatalf("App().Test() status error = %v", skippedStatusErr)
	}
	if skippedStatusResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("status route status = %d, want %d", skippedStatusResp.StatusCode, stdhttp.StatusOK)
	}

	skippedOpenAPIReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/openapi.json", nil)
	skippedOpenAPIResp, skippedOpenAPIErr := server.App().Test(skippedOpenAPIReq)
	if skippedOpenAPIErr != nil {
		t.Fatalf("App().Test() openapi error = %v", skippedOpenAPIErr)
	}
	if skippedOpenAPIResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("openapi status = %d, want %d", skippedOpenAPIResp.StatusCode, stdhttp.StatusOK)
	}

	skippedDocsReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/docs", nil)
	skippedDocsResp, skippedDocsErr := server.App().Test(skippedDocsReq)
	if skippedDocsErr != nil {
		t.Fatalf("App().Test() docs error = %v", skippedDocsErr)
	}
	if skippedDocsResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("docs status = %d, want %d", skippedDocsResp.StatusCode, stdhttp.StatusOK)
	}

	visibleReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/contacts?email=test%40example.com&limit=2&page=1", nil)
	visibleResp, visibleErr := server.App().Test(visibleReq)
	if visibleErr != nil {
		t.Fatalf("App().Test() visible error = %v", visibleErr)
	}
	if visibleResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("visible status = %d, want %d", visibleResp.StatusCode, stdhttp.StatusOK)
	}

	payload := output.String()
	if strings.Contains(payload, "/contacts?email=test%40example.com&limit=1&page=1") {
		t.Fatalf("expected skipped contacts lookup request to be absent from logs, got %q", payload)
	}
	if strings.Contains(payload, "/orders?identifier=1024112&realm=woocommerce&limit=1&page=1") {
		t.Fatalf("expected skipped orders lookup request to be absent from logs, got %q", payload)
	}
	if strings.Contains(payload, "/products") {
		t.Fatalf("expected skipped products request to be absent from logs, got %q", payload)
	}
	if strings.Contains(payload, "/orders?page=1&limit=10") {
		t.Fatalf("expected skipped paginated orders request to be absent from logs, got %q", payload)
	}
	if strings.Contains(payload, "/contacts/c480b5cbb751bd434e63b5f9f344f527") {
		t.Fatalf("expected skipped contact-by-id request to be absent from logs, got %q", payload)
	}
	if strings.Contains(payload, "/metrics") {
		t.Fatalf("expected skipped metrics request to be absent from logs, got %q", payload)
	}
	if strings.Contains(payload, "/status") {
		t.Fatalf("expected skipped status request to be absent from logs, got %q", payload)
	}
	if strings.Contains(payload, "/openapi.json") {
		t.Fatalf("expected skipped openapi request to be absent from logs, got %q", payload)
	}
	if strings.Contains(payload, "/docs") {
		t.Fatalf("expected skipped docs request to be absent from logs, got %q", payload)
	}
	if !strings.Contains(payload, "/contacts?email=test%40example.com&limit=2&page=1") {
		t.Fatalf("expected visible contacts request to remain logged, got %q", payload)
	}
}
