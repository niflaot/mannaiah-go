package telemetry

import "testing"

// TestConfigNormalizedDefaults verifies telemetry default normalization behavior.
func TestConfigNormalizedDefaults(t *testing.T) {
	cfg := Config{}
	normalized := cfg.Normalized()

	if normalized.ServiceName != defaultServiceName {
		t.Fatalf("ServiceName = %q, want %q", normalized.ServiceName, defaultServiceName)
	}
	if normalized.ServiceVersion != defaultServiceVersion {
		t.Fatalf("ServiceVersion = %q, want %q", normalized.ServiceVersion, defaultServiceVersion)
	}
	if normalized.MetricsPath != defaultMetricsPath {
		t.Fatalf("MetricsPath = %q, want %q", normalized.MetricsPath, defaultMetricsPath)
	}
	if normalized.TracesSamplerRatio != defaultTracesSamplerRatio {
		t.Fatalf("TracesSamplerRatio = %f, want %f", normalized.TracesSamplerRatio, defaultTracesSamplerRatio)
	}
	if normalized.DBStatsIntervalMS != defaultDBStatsIntervalMS {
		t.Fatalf("DBStatsIntervalMS = %d, want %d", normalized.DBStatsIntervalMS, defaultDBStatsIntervalMS)
	}
}

// TestConfigNormalizedMetricsPath verifies metrics path normalization behavior.
func TestConfigNormalizedMetricsPath(t *testing.T) {
	normalized := Config{MetricsPath: "metrics"}.Normalized()
	if normalized.MetricsPath != "/metrics" {
		t.Fatalf("MetricsPath = %q, want %q", normalized.MetricsPath, "/metrics")
	}
}
