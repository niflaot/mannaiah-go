package telemetry

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// HTTPMiddleware builds one Fiber middleware for HTTP tracing and metrics.
func (p *Provider) HTTPMiddleware() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		if p == nil || (!p.cfg.Enabled) {
			return ctx.Next()
		}

		method := strings.TrimSpace(ctx.Method())
		startedAt := time.Now()

		if p.httpInFlight != nil {
			p.httpInFlight.Inc()
			defer p.httpInFlight.Dec()
		}

		carrier := propagation.MapCarrier{}
		for key, values := range ctx.GetReqHeaders() {
			if len(values) == 0 {
				continue
			}
			carrier.Set(strings.ToLower(strings.TrimSpace(key)), values[0])
		}

		requestCtx := p.propagator.Extract(ctx.UserContext(), carrier)
		routeTemplate := resolveRouteTemplate(ctx)
		spanName := strings.TrimSpace(method + " " + routeTemplate)
		if spanName == "" {
			spanName = "http.request"
		}

		spanAttributes := []attribute.KeyValue{
			attribute.String("http.request.method", method),
			attribute.String("url.path", routeTemplate),
		}

		spanCtx, span := StartSpan(
			requestCtx,
			"mannaiah/http",
			spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(spanAttributes...),
		)
		ctx.SetUserContext(spanCtx)

		err := ctx.Next()
		statusCode := ctx.Response().StatusCode()
		if statusCode <= 0 {
			statusCode = fiber.StatusOK
		}
		statusCodeText := strconv.Itoa(statusCode)

		routeTemplate = resolveRouteTemplate(ctx)
		if p.httpRequestsTotal != nil {
			p.httpRequestsTotal.WithLabelValues(method, routeTemplate, statusCodeText).Inc()
		}
		if p.httpRequestDuration != nil {
			p.httpRequestDuration.WithLabelValues(method, routeTemplate, statusCodeText).Observe(time.Since(startedAt).Seconds())
		}

		span.SetAttributes(
			attribute.String("http.response.status_code", statusCodeText),
			attribute.String("http.route", routeTemplate),
		)

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "")
		}
		span.End()

		return err
	}
}

// resolveRouteTemplate resolves bounded route templates for metric labels.
func resolveRouteTemplate(ctx *fiber.Ctx) string {
	if ctx == nil {
		return "/"
	}
	route := ctx.Route()
	if route != nil {
		trimmedPath := strings.TrimSpace(route.Path)
		if trimmedPath != "" {
			return trimmedPath
		}
	}

	trimmedPath := strings.TrimSpace(ctx.Path())
	if trimmedPath == "" {
		return "/"
	}

	return trimmedPath
}
