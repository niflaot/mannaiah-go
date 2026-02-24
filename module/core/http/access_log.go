package http

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
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
	if path == "/metrics" || path == "/status" || path == "/openapi.json" || strings.HasPrefix(path, "/docs") {
		return true
	}
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

// accessLogFields builds per-request correlation fields appended to access logs.
func accessLogFields(ctx *fiber.Ctx) []zap.Field {
	rayID := resolveAccessRayID(ctx)
	traceID := resolveAccessTraceID(ctx, rayID)

	return []zap.Field{
		zap.String("ray_id", rayID),
		zap.String("trace_id", traceID),
	}
}

// resolveAccessRayID resolves the final response/request tracing identifier for access logs.
func resolveAccessRayID(ctx *fiber.Ctx) string {
	if ctx == nil {
		return ""
	}

	responseRayID := strings.TrimSpace(ctx.GetRespHeader(HeaderRayID))
	if responseRayID != "" {
		return responseRayID
	}

	return readOrCreateRayID(ctx)
}

// resolveAccessTraceID resolves the OpenTelemetry trace identifier for access logs.
func resolveAccessTraceID(ctx *fiber.Ctx, fallback string) string {
	if ctx == nil {
		return strings.TrimSpace(fallback)
	}

	spanContext := trace.SpanContextFromContext(ctx.UserContext())
	if spanContext.IsValid() {
		return spanContext.TraceID().String()
	}

	return strings.TrimSpace(fallback)
}
