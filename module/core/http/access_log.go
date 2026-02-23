package http

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// shouldSkipAccessLog reports whether one request should be excluded from access logging.
func shouldSkipAccessLog(ctx *fiber.Ctx) bool {
	if ctx == nil {
		return false
	}
	if ctx.Method() != fiber.MethodGet {
		return false
	}

	path := strings.TrimSpace(ctx.Path())
	if path == "/products" {
		return true
	}
	if path == "/orders" {
		if strings.TrimSpace(ctx.Query("identifier")) != "" &&
			strings.EqualFold(strings.TrimSpace(ctx.Query("realm")), "woocommerce") &&
			ctx.Query("limit") == "1" &&
			ctx.Query("page") == "1" {
			return true
		}
		return strings.TrimSpace(ctx.Query("page")) != "" && strings.TrimSpace(ctx.Query("limit")) != ""
	}
	if path == "/contacts" {
		return strings.TrimSpace(ctx.Query("email")) != "" &&
			ctx.Query("limit") == "1" &&
			ctx.Query("page") == "1"
	}
	if strings.HasPrefix(path, "/contacts/") {
		return strings.TrimSpace(strings.TrimPrefix(path, "/contacts/")) != ""
	}

	return false
}
