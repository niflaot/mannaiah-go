package telemetry

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

var (
	// activeProvider stores the process-wide telemetry provider.
	activeProvider atomic.Pointer[Provider]
)

// Provider defines telemetry runtime dependencies and instruments.
type Provider struct {
	// cfg defines normalized telemetry configuration.
	cfg Config
	// logger receives telemetry initialization and fallback logs.
	logger *zap.Logger
	// registry defines Prometheus registry values used by this process.
	registry *prometheus.Registry
	// metricsHandler defines Prometheus exposition HTTP handler.
	metricsHandler http.Handler
	// tracerProvider defines traces provider behavior.
	tracerProvider *sdktrace.TracerProvider
	// shutdownTracer closes trace exporters and flushes remaining spans.
	shutdownTracer func(ctx context.Context) error
	// propagator defines trace context propagation behavior.
	propagator propagation.TextMapPropagator

	// httpRequestsTotal defines HTTP request count metrics.
	httpRequestsTotal *prometheus.CounterVec
	// httpRequestDuration defines HTTP request latency distribution metrics.
	httpRequestDuration *prometheus.HistogramVec
	// httpInFlight defines current HTTP in-flight request metrics.
	httpInFlight prometheus.Gauge

	// dependencyRequestsTotal defines dependency request count metrics.
	dependencyRequestsTotal *prometheus.CounterVec
	// dependencyRequestDuration defines dependency request duration metrics.
	dependencyRequestDuration *prometheus.HistogramVec

	// messagingPublishTotal defines messaging publish event metrics.
	messagingPublishTotal *prometheus.CounterVec
	// messagingConsumeTotal defines messaging consume event metrics.
	messagingConsumeTotal *prometheus.CounterVec
	// messagingHandlerDuration defines messaging handler latency distribution metrics.
	messagingHandlerDuration *prometheus.HistogramVec
	// messagingDLQTotal defines dead-letter publish count metrics.
	messagingDLQTotal *prometheus.CounterVec
	// messagingRetryTotal defines retry-attempt count metrics.
	messagingRetryTotal *prometheus.CounterVec

	// dbOpenConnections defines SQL pool open connection metrics.
	dbOpenConnections prometheus.Gauge
	// dbInUseConnections defines SQL pool in-use connection metrics.
	dbInUseConnections prometheus.Gauge
	// dbIdleConnections defines SQL pool idle connection metrics.
	dbIdleConnections prometheus.Gauge
	// dbWaitCount defines SQL pool wait-count metrics.
	dbWaitCount prometheus.Gauge
	// dbWaitDuration defines SQL pool cumulative wait-duration metrics.
	dbWaitDuration prometheus.Gauge
	// dbMaxOpenConnections defines SQL pool max-open metrics.
	dbMaxOpenConnections prometheus.Gauge

	// dbStatsMu guards SQL stats collector lifecycle fields.
	dbStatsMu sync.Mutex
	// dbStatsStop requests SQL stats collector shutdown.
	dbStatsStop chan struct{}
	// dbStatsDone signals SQL stats collector shutdown completion.
	dbStatsDone chan struct{}
}

// Init initializes telemetry providers, metrics instruments, and global propagators.
func Init(ctx context.Context, cfg Config, providedLogger *zap.Logger) (*Provider, error) {
	resolvedCfg := cfg.Normalized()
	provider := &Provider{
		cfg:        resolvedCfg,
		logger:     resolveLogger(providedLogger),
		registry:   prometheus.NewRegistry(),
		propagator: propagation.TraceContext{},
	}

	provider.configureMetrics()
	provider.configureTracing(ctx)

	SetActive(provider)
	return provider, nil
}

// Shutdown gracefully flushes telemetry pipelines and background collectors.
func (p *Provider) Shutdown(ctx context.Context) error {
	if p == nil {
		return nil
	}

	p.stopSQLStatsCollector()

	if p.shutdownTracer != nil {
		if err := p.shutdownTracer(ctx); err != nil {
			return err
		}
	}

	return nil
}

// MetricsHandler returns the Prometheus exposition HTTP handler.
func (p *Provider) MetricsHandler() http.Handler {
	if p == nil || p.metricsHandler == nil {
		return http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusNoContent)
		})
	}

	return p.metricsHandler
}

// MetricsPath returns the configured metrics route path.
func (p *Provider) MetricsPath() string {
	if p == nil {
		return defaultMetricsPath
	}

	return p.cfg.MetricsPath
}

// StartSQLStatsCollector starts periodic SQL pool metrics collection.
func (p *Provider) StartSQLStatsCollector(db *sql.DB) {
	if p == nil || db == nil || p.dbOpenConnections == nil {
		return
	}

	p.dbStatsMu.Lock()
	if p.dbStatsStop != nil {
		p.dbStatsMu.Unlock()
		return
	}

	p.dbStatsStop = make(chan struct{})
	p.dbStatsDone = make(chan struct{})
	stop := p.dbStatsStop
	done := p.dbStatsDone
	interval := time.Duration(p.cfg.DBStatsIntervalMS) * time.Millisecond
	if interval <= 0 {
		interval = 15 * time.Second
	}
	p.dbStatsMu.Unlock()

	go func() {
		defer close(done)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		p.recordSQLStats(db)

		for {
			select {
			case <-ticker.C:
				p.recordSQLStats(db)
			case <-stop:
				return
			}
		}
	}()
}

// Active returns the process-wide telemetry provider.
func Active() *Provider {
	return activeProvider.Load()
}

// SetActive sets the process-wide telemetry provider.
func SetActive(provider *Provider) {
	activeProvider.Store(provider)
}

// StartSpan starts one span from the global tracer provider.
func StartSpan(ctx context.Context, tracerName string, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	resolvedCtx := ctx
	if resolvedCtx == nil {
		resolvedCtx = context.Background()
	}

	name := strings.TrimSpace(tracerName)
	if name == "" {
		name = "mannaiah"
	}

	return otel.Tracer(name).Start(resolvedCtx, spanName, opts...)
}

// EndSpan records final status and closes one span.
func EndSpan(span trace.Span, err error) {
	if span == nil {
		return
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	span.End()
}

// TraceparentFromContext renders a W3C traceparent value from one context.
func TraceparentFromContext(ctx context.Context) string {
	resolvedCtx := ctx
	if resolvedCtx == nil {
		resolvedCtx = context.Background()
	}

	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(resolvedCtx, carrier)
	return strings.TrimSpace(carrier.Get("traceparent"))
}

// ContextWithTraceparent extracts W3C trace context values into one context.
func ContextWithTraceparent(ctx context.Context, traceparent string) context.Context {
	resolvedCtx := ctx
	if resolvedCtx == nil {
		resolvedCtx = context.Background()
	}
	trimmedTraceparent := strings.TrimSpace(traceparent)
	if trimmedTraceparent == "" {
		return resolvedCtx
	}

	carrier := propagation.MapCarrier{}
	carrier.Set("traceparent", trimmedTraceparent)

	return otel.GetTextMapPropagator().Extract(resolvedCtx, carrier)
}

// RecordDependency records dependency request counters and durations.
func RecordDependency(dependency string, operation string, startedAt time.Time, err error) {
	provider := Active()
	if provider == nil || provider.dependencyRequestsTotal == nil || provider.dependencyRequestDuration == nil {
		return
	}

	duration := time.Since(startedAt).Seconds()
	result := classifyResult(err)
	provider.dependencyRequestsTotal.WithLabelValues(dependency, operation, result).Inc()
	provider.dependencyRequestDuration.WithLabelValues(dependency, operation, result).Observe(duration)
}

// RecordMessaging records messaging event counters and handler latencies.
func RecordMessaging(topic string, operation string, startedAt time.Time, err error) {
	provider := Active()
	if provider == nil {
		return
	}

	result := classifyResult(err)
	trimmedOperation := strings.TrimSpace(operation)
	switch trimmedOperation {
	case "publish":
		if provider.messagingPublishTotal != nil {
			provider.messagingPublishTotal.WithLabelValues(topic, "publish", result).Inc()
		}
	case "consume":
		if provider.messagingConsumeTotal != nil {
			provider.messagingConsumeTotal.WithLabelValues(topic, "consume", result).Inc()
		}
		if provider.messagingHandlerDuration != nil {
			provider.messagingHandlerDuration.WithLabelValues(topic, "consume", result).Observe(time.Since(startedAt).Seconds())
		}
	case "retry":
		if provider.messagingRetryTotal != nil {
			provider.messagingRetryTotal.WithLabelValues(topic, "retry", result).Inc()
		}
	}
}

// IncMessagingDLQ increments dead-letter counters for one topic.
func IncMessagingDLQ(topic string) {
	provider := Active()
	if provider == nil || provider.messagingDLQTotal == nil {
		return
	}

	provider.messagingDLQTotal.WithLabelValues(topic, "dlq", "ok").Inc()
}

// configureMetrics initializes Prometheus collectors and exposition handlers.
func (p *Provider) configureMetrics() {
	p.metricsHandler = promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{})

	if !p.cfg.Enabled || !p.cfg.MetricsEnabled {
		return
	}

	p.registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	p.registry.MustRegister(prometheus.NewGoCollector())

	p.httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mannaiah_http_server_requests_total",
			Help: "Total HTTP requests served by method, route, and status code.",
		},
		[]string{"method", "route", "status_code"},
	)
	p.httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mannaiah_http_server_request_duration_seconds",
			Help:    "HTTP request latency distribution by method, route, and status code.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route", "status_code"},
	)
	p.httpInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "mannaiah_http_server_in_flight_requests",
			Help: "Current in-flight HTTP requests.",
		},
	)

	p.dependencyRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mannaiah_dependency_requests_total",
			Help: "Total dependency requests by dependency, operation, and result.",
		},
		[]string{"dependency", "operation", "result"},
	)
	p.dependencyRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mannaiah_dependency_request_duration_seconds",
			Help:    "Dependency request latency distribution by dependency, operation, and result.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"dependency", "operation", "result"},
	)

	p.messagingPublishTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mannaiah_messaging_publish_total",
			Help: "Total messaging publish events by topic and result.",
		},
		[]string{"topic", "operation", "result"},
	)
	p.messagingConsumeTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mannaiah_messaging_consume_total",
			Help: "Total messaging consume events by topic and result.",
		},
		[]string{"topic", "operation", "result"},
	)
	p.messagingHandlerDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mannaiah_messaging_handler_duration_seconds",
			Help:    "Messaging consumer handler latency distribution by topic and result.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"topic", "operation", "result"},
	)
	p.messagingDLQTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mannaiah_messaging_dlq_total",
			Help: "Total dead-letter publishes by original topic.",
		},
		[]string{"topic", "operation", "result"},
	)
	p.messagingRetryTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mannaiah_messaging_retry_total",
			Help: "Total retry attempts by topic and result.",
		},
		[]string{"topic", "operation", "result"},
	)

	p.dbOpenConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "mannaiah_db_pool_open_connections",
		Help: "Current number of open SQL connections.",
	})
	p.dbInUseConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "mannaiah_db_pool_in_use_connections",
		Help: "Current number of SQL connections in use.",
	})
	p.dbIdleConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "mannaiah_db_pool_idle_connections",
		Help: "Current number of idle SQL connections.",
	})
	p.dbWaitCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "mannaiah_db_pool_wait_count_total",
		Help: "Total number of waits for SQL connections.",
	})
	p.dbWaitDuration = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "mannaiah_db_pool_wait_duration_seconds",
		Help: "Total waiting duration for SQL connections in seconds.",
	})
	p.dbMaxOpenConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "mannaiah_db_pool_max_open_connections",
		Help: "Configured max open SQL connections.",
	})

	p.registry.MustRegister(
		p.httpRequestsTotal,
		p.httpRequestDuration,
		p.httpInFlight,
		p.dependencyRequestsTotal,
		p.dependencyRequestDuration,
		p.messagingPublishTotal,
		p.messagingConsumeTotal,
		p.messagingHandlerDuration,
		p.messagingDLQTotal,
		p.messagingRetryTotal,
		p.dbOpenConnections,
		p.dbInUseConnections,
		p.dbIdleConnections,
		p.dbWaitCount,
		p.dbWaitDuration,
		p.dbMaxOpenConnections,
	)
}

// configureTracing initializes trace providers and global propagators.
func (p *Provider) configureTracing(ctx context.Context) {
	otel.SetTextMapPropagator(p.propagator)

	if !p.cfg.Enabled || !p.cfg.TracesEnabled {
		tp := sdktrace.NewTracerProvider()
		p.tracerProvider = tp
		p.shutdownTracer = tp.Shutdown
		otel.SetTracerProvider(tp)
		return
	}

	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceName(p.cfg.ServiceName),
			semconv.ServiceVersion(p.cfg.ServiceVersion),
		),
	)
	if err != nil {
		p.logger.Warn("telemetry resource initialization failed; using defaults", zap.Error(err))
		res = resource.Default()
	}

	sampler := resolveSampler(p.cfg)
	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	trimmedExporter := strings.ToLower(strings.TrimSpace(p.cfg.TracesExporter))
	trimmedEndpoint := strings.TrimSpace(p.cfg.OTLPEndpoint)
	if trimmedExporter == "otlp" && trimmedEndpoint != "" {
		exporterOptions := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(trimmedEndpoint),
		}
		if p.cfg.OTLPInsecure {
			exporterOptions = append(exporterOptions, otlptracegrpc.WithInsecure())
		}

		exporter, exporterErr := otlptracegrpc.New(ctx, exporterOptions...)
		if exporterErr != nil {
			p.logger.Warn("otlp exporter initialization failed; tracing export disabled", zap.Error(exporterErr))
		} else {
			traceProvider = sdktrace.NewTracerProvider(
				sdktrace.WithResource(res),
				sdktrace.WithSampler(sampler),
				sdktrace.WithBatcher(exporter),
			)
		}
	}

	p.tracerProvider = traceProvider
	p.shutdownTracer = traceProvider.Shutdown
	otel.SetTracerProvider(traceProvider)
}

// recordSQLStats updates DB pool gauges from one SQL stats snapshot.
func (p *Provider) recordSQLStats(db *sql.DB) {
	stats := db.Stats()
	p.dbOpenConnections.Set(float64(stats.OpenConnections))
	p.dbInUseConnections.Set(float64(stats.InUse))
	p.dbIdleConnections.Set(float64(stats.Idle))
	p.dbWaitCount.Set(float64(stats.WaitCount))
	p.dbWaitDuration.Set(stats.WaitDuration.Seconds())
	p.dbMaxOpenConnections.Set(float64(stats.MaxOpenConnections))
}

// stopSQLStatsCollector stops a running SQL pool collector when present.
func (p *Provider) stopSQLStatsCollector() {
	p.dbStatsMu.Lock()
	stop := p.dbStatsStop
	done := p.dbStatsDone
	p.dbStatsStop = nil
	p.dbStatsDone = nil
	p.dbStatsMu.Unlock()

	if stop != nil {
		close(stop)
	}
	if done != nil {
		<-done
	}
}

// resolveSampler maps configured sampler names to SDK samplers.
func resolveSampler(cfg Config) sdktrace.Sampler {
	switch strings.ToLower(strings.TrimSpace(cfg.TracesSampler)) {
	case "always_on":
		return sdktrace.AlwaysSample()
	case "always_off":
		return sdktrace.NeverSample()
	case "traceidratio":
		return sdktrace.TraceIDRatioBased(cfg.TracesSamplerRatio)
	default:
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.TracesSamplerRatio))
	}
}

// resolveLogger resolves nil loggers to no-op logger defaults.
func resolveLogger(logger *zap.Logger) *zap.Logger {
	if logger == nil {
		return zap.NewNop()
	}

	return logger
}

// classifyResult normalizes telemetry result values into bounded labels.
func classifyResult(err error) string {
	if err == nil {
		return "ok"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	if strings.Contains(strings.ToLower(err.Error()), "unavailable") {
		return "unavailable"
	}

	return "error"
}
