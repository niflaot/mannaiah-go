package http

import (
	stdhttp "net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// TestShouldSkipAccessLogCoreRoutes verifies access-log suppression for core infrastructure routes.
func TestShouldSkipAccessLogCoreRoutes(t *testing.T) {
	server, err := New(Config{Host: "127.0.0.1", Port: 8120}, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	server.Register(func(app *fiber.App) {
		app.Get("/metrics", func(ctx *fiber.Ctx) error { return ctx.SendStatus(stdhttp.StatusOK) })
		app.Get("/status", func(ctx *fiber.Ctx) error { return ctx.SendStatus(stdhttp.StatusOK) })
		app.Get("/openapi.json", func(ctx *fiber.Ctx) error { return ctx.SendStatus(stdhttp.StatusOK) })
		app.Get("/docs", func(ctx *fiber.Ctx) error { return ctx.SendStatus(stdhttp.StatusOK) })
		app.Get("/orders", func(ctx *fiber.Ctx) error { return ctx.SendStatus(stdhttp.StatusOK) })
	})

	cases := []string{"/metrics", "/status", "/openapi.json", "/docs"}
	for _, path := range cases {
		request, _ := stdhttp.NewRequest(stdhttp.MethodGet, path, nil)
		response, testErr := server.App().Test(request)
		if testErr != nil {
			t.Fatalf("App().Test(%s) error = %v", path, testErr)
		}
		if response.StatusCode != stdhttp.StatusOK {
			t.Fatalf("%s status = %d, want %d", path, response.StatusCode, stdhttp.StatusOK)
		}
	}
}
