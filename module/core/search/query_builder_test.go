package search

import (
	"testing"
)

// TestEscapeLike verifies SQL LIKE special character escaping.
func TestEscapeLike(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"50%", "50\\%"},
		{"a_b", "a\\_b"},
		{"%_%%", "\\%\\_\\%\\%"},
		{"", ""},
	}
	for _, tt := range tests {
		got := escapeLike(tt.input)
		if got != tt.want {
			t.Errorf("escapeLike(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestIsOperatorAllowed verifies operator whitelist checking.
func TestIsOperatorAllowed(t *testing.T) {
	allowed := map[string][]Operator{
		"status":     {OpEQ, OpIn},
		"created_at": {OpGTE, OpLTE, OpBetween},
	}

	tests := []struct {
		field string
		op    Operator
		want  bool
	}{
		{"status", OpEQ, true},
		{"status", OpIn, true},
		{"status", OpGTE, false},
		{"created_at", OpGTE, true},
		{"created_at", OpEQ, false},
		{"unknown", OpEQ, false},
	}
	for _, tt := range tests {
		got := isOperatorAllowed(tt.field, tt.op, allowed)
		if got != tt.want {
			t.Errorf("isOperatorAllowed(%q, %q) = %v, want %v", tt.field, tt.op, got, tt.want)
		}
	}
}

// TestIsOperatorAllowedNilMap verifies nil map returns false.
func TestIsOperatorAllowedNilMap(t *testing.T) {
	if isOperatorAllowed("field", OpEQ, nil) {
		t.Error("expected false for nil allowed map")
	}
}

// TestIsSortable verifies sort field whitelist checking.
func TestIsSortable(t *testing.T) {
	sortable := []string{"name", "created_at", "price"}

	tests := []struct {
		field string
		want  bool
	}{
		{"name", true},
		{"created_at", true},
		{"unknown", false},
		{"", false},
	}
	for _, tt := range tests {
		got := isSortable(tt.field, sortable)
		if got != tt.want {
			t.Errorf("isSortable(%q) = %v, want %v", tt.field, got, tt.want)
		}
	}
}

// TestResolveSortDefaults verifies fallback to descriptor default sort.
func TestResolveSortDefaults(t *testing.T) {
	desc := Descriptor{
		DefaultSort:    SortField{Field: "name", Direction: Asc},
		SortableFields: []string{"name", "created_at"},
	}

	result := resolveSort(nil, desc)
	if len(result) != 1 || result[0].Field != "name" || result[0].Direction != Asc {
		t.Errorf("resolveSort(nil) = %v, want [{name ASC}]", result)
	}
}

// TestResolveSortFallsBackOnInvalidFields verifies unknown fields fall back.
func TestResolveSortFallsBackOnInvalidFields(t *testing.T) {
	desc := Descriptor{
		DefaultSort:    SortField{Field: "name", Direction: Asc},
		SortableFields: []string{"name", "created_at"},
	}

	result := resolveSort([]SortField{{Field: "unknown", Direction: Desc}}, desc)
	if len(result) != 1 || result[0].Field != "name" {
		t.Errorf("resolveSort with unknown field = %v, want [{name ASC}]", result)
	}
}

// TestResolveSortValidFields verifies valid requested fields pass through.
func TestResolveSortValidFields(t *testing.T) {
	desc := Descriptor{
		SortableFields: []string{"name", "created_at"},
	}

	requested := []SortField{
		{Field: "created_at", Direction: Desc},
		{Field: "name", Direction: Asc},
	}
	result := resolveSort(requested, desc)
	if len(result) != 2 {
		t.Fatalf("resolveSort returned %d fields, want 2", len(result))
	}
	if result[0].Field != "created_at" || result[0].Direction != Desc {
		t.Errorf("sort[0] = %v, want {created_at DESC}", result[0])
	}
}

// TestResolveSortInvalidDirection normalizes unknown direction to Desc.
func TestResolveSortInvalidDirection(t *testing.T) {
	desc := Descriptor{
		SortableFields: []string{"name"},
	}

	result := resolveSort([]SortField{{Field: "name", Direction: "INVALID"}}, desc)
	if result[0].Direction != Desc {
		t.Errorf("direction = %q, want %q", result[0].Direction, Desc)
	}
}

// TestResolveSortNoDefaultFallsBackToCreatedAt tests ultimate fallback.
func TestResolveSortNoDefaultFallsBackToCreatedAt(t *testing.T) {
	desc := Descriptor{
		SortableFields: []string{"name"},
	}

	result := resolveSort(nil, desc)
	if len(result) != 1 || result[0].Field != "created_at" || result[0].Direction != Desc {
		t.Errorf("resolveSort(nil, no default) = %v, want [{created_at DESC}]", result)
	}
}
