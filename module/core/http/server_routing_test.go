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
