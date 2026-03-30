package search

import (
	"testing"
)

// TestScoreResultsExactMatch verifies exact match on primary field.
func TestScoreResultsExactMatch(t *testing.T) {
	entities := []string{"john", "johnny", "jane"}
	extract := func(e string, _ string) string { return e }
	hits := ScoreResults(entities, "john", []string{"name"}, nil, extract)

	if len(hits) != 3 {
		t.Fatalf("expected 3 hits, got %d", len(hits))
	}
	if hits[0].Score != scoreExactMatch+boostPrimaryField {
		t.Errorf("exact match score = %f, want %f", hits[0].Score, scoreExactMatch+boostPrimaryField)
	}
	if hits[0].MatchedField != "name" {
		t.Errorf("matched field = %q, want %q", hits[0].MatchedField, "name")
	}
}

// TestScoreResultsPrefixMatch verifies prefix match scoring.
func TestScoreResultsPrefixMatch(t *testing.T) {
	entities := []string{"johnny"}
	extract := func(e string, _ string) string { return e }
	hits := ScoreResults(entities, "john", []string{"name"}, nil, extract)

	expected := scorePrefixMatch + boostPrimaryField
	if hits[0].Score != expected {
		t.Errorf("prefix match score = %f, want %f", hits[0].Score, expected)
	}
}

// TestScoreResultsContainsMatch verifies contains match scoring.
func TestScoreResultsContainsMatch(t *testing.T) {
	entities := []string{"maryjohn"}
	extract := func(e string, _ string) string { return e }
	hits := ScoreResults(entities, "john", []string{"name"}, nil, extract)

	expected := scoreContains + boostPrimaryField
	if hits[0].Score != expected {
		t.Errorf("contains match score = %f, want %f", hits[0].Score, expected)
	}
}

// TestScoreResultsNoMatch verifies no-match entities scored with boost only.
func TestScoreResultsNoMatch(t *testing.T) {
	entities := []string{"alice"}
	extract := func(e string, _ string) string { return e }
	hits := ScoreResults(entities, "john", []string{"name"}, nil, extract)

	if hits[0].Score != boostPrimaryField {
		t.Errorf("no-match score = %f, want %f (boost only)", hits[0].Score, boostPrimaryField)
	}
}

// TestScoreResultsSecondaryField checks secondary field boost is lower.
func TestScoreResultsSecondaryField(t *testing.T) {
	entities := []string{"john"}
	extract := func(e string, _ string) string { return e }
	hits := ScoreResults(entities, "john", nil, []string{"alt"}, extract)

	expected := scoreExactMatch + boostOtherField
	if hits[0].Score != expected {
		t.Errorf("secondary exact score = %f, want %f", hits[0].Score, expected)
	}
}

// TestScoreResultsEmptyTerm returns zero-score hits.
func TestScoreResultsEmptyTerm(t *testing.T) {
	entities := []string{"john"}
	extract := func(e string, _ string) string { return e }
	hits := ScoreResults(entities, "", []string{"name"}, nil, extract)

	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(hits))
	}
	if hits[0].Score != 0 {
		t.Errorf("empty term score = %f, want 0", hits[0].Score)
	}
}

// TestScoreResultsEmptyEntities returns empty slice.
func TestScoreResultsEmptyEntities(t *testing.T) {
	extract := func(e string, _ string) string { return e }
	hits := ScoreResults([]string{}, "john", []string{"name"}, nil, extract)

	if len(hits) != 0 {
		t.Fatalf("expected 0 hits, got %d", len(hits))
	}
}

// TestComputeFieldScore verifies internal scoring logic.
func TestComputeFieldScore(t *testing.T) {
	tests := []struct {
		name  string
		term  string
		value string
		want  float64
	}{
		{"exact", "john", "john", scoreExactMatch},
		{"prefix", "john", "johnny", scorePrefixMatch},
		{"contains", "john", "xjohny", scoreContains},
		{"no match", "john", "alice", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeFieldScore(tt.term, tt.value)
			if got != tt.want {
				t.Errorf("computeFieldScore(%q, %q) = %f, want %f", tt.term, tt.value, got, tt.want)
			}
		})
	}
}
