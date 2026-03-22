package telemetry

import "strings"

const (
	// defaultServiceName defines fallback telemetry service-name values.
	defaultServiceName = "mannaiah-api"
	// defaultServiceVersion defines fallback telemetry service-version values.
	defaultServiceVersion = "v2.9.8"
	// defaultMetricsPath defines fallback Prometheus metrics endpoint paths.
	defaultMetricsPath = "/metrics"
	// defaultTracesExporter defines fallback traces exporter values.
	defaultTracesExporter = "otlp"
	// defaultTracesSampler defines fallback traces sampler values.
	defaultTracesSampler = "parentbased_traceidratio"
	// defaultTracesSamplerRatio defines fallback trace sampling ratio values.
	defaultTracesSamplerRatio = 0.10
	// defaultDBStatsIntervalMS defines fallback DB stats collection intervals.
	defaultDBStatsIntervalMS = 15000
)

// Config defines telemetry runtime settings.
type Config struct {
	// Enabled enables telemetry initialization and instrumentation.
	Enabled bool `mapstructure:"TELEMETRY_ENABLED" default:"true"`
	// ServiceName defines telemetry service.name resource values.
	ServiceName string `mapstructure:"TELEMETRY_SERVICE_NAME" default:"mannaiah-api"`
	// ServiceVersion defines telemetry service.version resource values.
	ServiceVersion string `mapstructure:"TELEMETRY_SERVICE_VERSION" default:"v2.9.8"`
	// TracesEnabled enables OpenTelemetry tracing pipelines.
	TracesEnabled bool `mapstructure:"TELEMETRY_TRACES_ENABLED" default:"true"`
	// TracesExporter defines traces exporter type (for example, otlp).
	TracesExporter string `mapstructure:"TELEMETRY_TRACES_EXPORTER" default:"otlp"`
	// OTLPEndpoint defines OTLP exporter endpoint host:port values.
	OTLPEndpoint string `mapstructure:"TELEMETRY_OTLP_ENDPOINT" default:"otel-collector:4317"`
	// OTLPInsecure enables insecure OTLP transport mode.
	OTLPInsecure bool `mapstructure:"TELEMETRY_OTLP_INSECURE" default:"false"`
	// TracesSampler defines traces sampler strategy values.
	TracesSampler string `mapstructure:"TELEMETRY_TRACES_SAMPLER" default:"parentbased_traceidratio"`
	// TracesSamplerRatio defines trace-id ratio sampling values.
	TracesSamplerRatio float64 `mapstructure:"TELEMETRY_TRACES_SAMPLER_RATIO" default:"0.10"`
	// MetricsEnabled enables Prometheus metrics collection and exposure.
	MetricsEnabled bool `mapstructure:"TELEMETRY_METRICS_ENABLED" default:"true"`
	// MetricsPath defines HTTP route path for Prometheus scraping.
	MetricsPath string `mapstructure:"TELEMETRY_METRICS_PATH" default:"/metrics"`
	// DBStatsIntervalMS defines SQL pool stats scrape interval in milliseconds.
	DBStatsIntervalMS int `mapstructure:"TELEMETRY_DB_STATS_INTERVAL_MS" default:"15000"`
}

// Normalized returns telemetry config values with safe defaults.
func (c Config) Normalized() Config {
	result := c
	if strings.TrimSpace(result.ServiceName) == "" {
		result.ServiceName = defaultServiceName
	}
	if strings.TrimSpace(result.ServiceVersion) == "" {
		result.ServiceVersion = defaultServiceVersion
	}
	if strings.TrimSpace(result.TracesExporter) == "" {
		result.TracesExporter = defaultTracesExporter
	}
	if strings.TrimSpace(result.TracesSampler) == "" {
		result.TracesSampler = defaultTracesSampler
	}
	if result.TracesSamplerRatio <= 0 || result.TracesSamplerRatio > 1 {
		result.TracesSamplerRatio = defaultTracesSamplerRatio
	}

	trimmedMetricsPath := strings.TrimSpace(result.MetricsPath)
	if trimmedMetricsPath == "" {
		trimmedMetricsPath = defaultMetricsPath
	}
	if !strings.HasPrefix(trimmedMetricsPath, "/") {
		trimmedMetricsPath = "/" + trimmedMetricsPath
	}
	result.MetricsPath = trimmedMetricsPath

	if result.DBStatsIntervalMS <= 0 {
		result.DBStatsIntervalMS = defaultDBStatsIntervalMS
	}

	return result
}
