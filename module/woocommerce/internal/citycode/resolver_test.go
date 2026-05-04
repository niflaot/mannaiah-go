package citycode_test

import (
	"testing"

	"mannaiah/module/woocommerce/internal/citycode"
)

func TestResolve_ExactMatch(t *testing.T) {
	tests := []struct {
		input    string
		wantCode string
	}{
		{"Bogota", "11001"},
		{"bogota", "11001"},
		{"BOGOTA", "11001"},
		{"Medellín", "05001"},
		{"Medellin", "05001"},
		{"Cali", "76001"},
		{"Barranquilla", "08001"},
		{"Cartagena", "13001"},
		{"Leticia", "91001"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := citycode.Resolve(tc.input)
			if got != tc.wantCode {
				t.Errorf("Resolve(%q) = %q, want %q", tc.input, got, tc.wantCode)
			}
		})
	}
}

func TestResolve_AccentVariants(t *testing.T) {
	tests := []struct {
		input    string
		wantCode string
	}{
		{"Bogotá", "11001"},
		{"Medellín", "05001"},
		{"Tunja", "15001"},
		{"Manizales", "17001"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := citycode.Resolve(tc.input)
			if got != tc.wantCode {
				t.Errorf("Resolve(%q) = %q, want %q", tc.input, got, tc.wantCode)
			}
		})
	}
}

func TestResolve_FuzzyMatch(t *testing.T) {
	tests := []struct {
		input    string
		wantCode string
	}{
		{"Bogotá D.C", "11001"},
		{"Medellin ", "05001"},
		{"  Cali  ", "76001"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := citycode.Resolve(tc.input)
			if got != tc.wantCode {
				t.Errorf("Resolve(%q) = %q, want %q", tc.input, got, tc.wantCode)
			}
		})
	}
}

func TestResolve_UnknownCity(t *testing.T) {
	tests := []string{
		"",
		"   ",
		"XyzUnknownCity12345",
		"aaaaaabbbbbbcccccc",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			got := citycode.Resolve(input)
			if got != "-1" {
				t.Errorf("Resolve(%q) = %q, want \"-1\"", input, got)
			}
		})
	}
}

func TestIsNumericCode(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"11001", true},
		{"05001", true},
		{"-1", false},
		{"Bogota", false},
		{"", false},
		{"  ", false},
		{"0", false},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := citycode.IsNumericCode(tc.input)
			if got != tc.want {
				t.Errorf("IsNumericCode(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func BenchmarkResolveExact(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		citycode.Resolve("Bogota")
	}
}

func BenchmarkResolveFuzzy(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		citycode.Resolve("Bogotá D.C")
	}
}

func BenchmarkResolveUnknown(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		citycode.Resolve("XyzUnknownCity12345")
	}
}
