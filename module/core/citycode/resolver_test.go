package citycode_test

import (
	"context"
	"testing"

	"mannaiah/module/core/citycode"
)

// TestResolve maps city names and numeric values to normalized municipality codes.
func TestResolve(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Bogota", "11001"},
		{"Bogotá D.C", "11001"},
		{"Medellín", "05001"},
		{"Armenia", "-1"},
		{"05001", "05001"},
		{"5001", "05001"},
		{"XyzUnknownCity12345", "-1"},
	}

	for _, tc := range tests {
		if got := citycode.Resolve(tc.input); got != tc.want {
			t.Fatalf("Resolve(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// TestResolveDetailed reports duplicate city candidates for operator repair.
func TestResolveDetailed(t *testing.T) {
	result, err := citycode.ResolveDetailed(context.Background(), "Armenia", "")
	if err != nil {
		t.Fatalf("ResolveDetailed() error = %v", err)
	}
	if result.Found {
		t.Fatal("ResolveDetailed() found = true, want duplicate rejection")
	}
	if result.Reason != "DUPLICATED" {
		t.Fatalf("ResolveDetailed() reason = %q, want DUPLICATED", result.Reason)
	}
	if len(result.Suggestions) < 2 {
		t.Fatalf("len(Suggestions) = %d, want at least 2", len(result.Suggestions))
	}
}

// TestResolveDetailedUsesDepartmentEvidence resolves duplicated names when department evidence is present.
func TestResolveDetailedUsesDepartmentEvidence(t *testing.T) {
	result, err := citycode.ResolveDetailed(context.Background(), "Armenia", "Quindio")
	if err != nil {
		t.Fatalf("ResolveDetailed() error = %v", err)
	}
	if !result.Found {
		t.Fatalf("ResolveDetailed() found = false, reason = %q", result.Reason)
	}
	if result.Code != "63001" {
		t.Fatalf("ResolveDetailed() code = %q, want 63001", result.Code)
	}
}

// TestName maps municipality codes back to city names for external payloads.
func TestName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"11001", "Bogotá D.C."},
		{"05001", "Medellín"},
		{"5001", "Medellín"},
		{"BOG", "BOG"},
	}

	for _, tc := range tests {
		if got := citycode.Name(tc.input); got != tc.want {
			t.Fatalf("Name(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// BenchmarkResolveCityName measures the shared integration city resolver hot path.
func BenchmarkResolveCityName(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if code := citycode.Resolve("Bogotá"); code != "11001" {
			b.Fatalf("expected 11001, got %q", code)
		}
	}
}
