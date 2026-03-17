package domain

import "testing"

// TestRFMBandConfigScoreDescending verifies recency (descending) band scoring logic.
func TestRFMBandConfigScoreDescending(t *testing.T) {
	cfg := RFMBandConfig{
		Ascending: false,
		Band5Min:  7,
		Band4Min:  30,
		Band3Min:  90,
		Band2Min:  180,
	}

	cases := []struct {
		value    float64
		expected int
	}{
		{0, 5},
		{7, 5},
		{8, 4},
		{30, 4},
		{31, 3},
		{90, 3},
		{91, 2},
		{180, 2},
		{181, 1},
		{999, 1},
	}

	for _, tc := range cases {
		got := cfg.ScoreValue(tc.value)
		if got != tc.expected {
			t.Errorf("ScoreValue(%v) = %d, want %d", tc.value, got, tc.expected)
		}
	}
}

// TestRFMBandConfigScoreAscending verifies frequency/monetary (ascending) band scoring logic.
func TestRFMBandConfigScoreAscending(t *testing.T) {
	cfg := RFMBandConfig{
		Ascending: true,
		Band5Min:  10,
		Band4Min:  6,
		Band3Min:  3,
		Band2Min:  2,
	}

	cases := []struct {
		value    float64
		expected int
	}{
		{1, 1},
		{2, 2},
		{3, 3},
		{6, 4},
		{10, 5},
		{100, 5},
	}

	for _, tc := range cases {
		got := cfg.ScoreValue(tc.value)
		if got != tc.expected {
			t.Errorf("ScoreValue(%v) = %d, want %d", tc.value, got, tc.expected)
		}
	}
}

// TestRFMGroupConditionsMatches verifies group condition matching logic.
func TestRFMGroupConditionsMatches(t *testing.T) {
	rMin, rMax := 3, 5
	fMin, fMax := 4, 5
	mMin, mMax := 3, 5

	cond := RFMGroupConditions{
		RMin: &rMin, RMax: &rMax,
		FMin: &fMin, FMax: &fMax,
		MMin: &mMin, MMax: &mMax,
	}

	match := RFMScore{RScore: 4, FScore: 4, MScore: 4}
	if !cond.Matches(match) {
		t.Errorf("Matches(champion) = false, want true")
	}

	noMatch := RFMScore{RScore: 1, FScore: 4, MScore: 4}
	if cond.Matches(noMatch) {
		t.Errorf("Matches(rScore=1) = true, want false")
	}
}

// TestRFMGroupConditionsMatchesEmpty verifies that empty conditions match any score.
func TestRFMGroupConditionsMatchesEmpty(t *testing.T) {
	cond := RFMGroupConditions{}
	score := RFMScore{RScore: 1, FScore: 1, MScore: 1}
	if !cond.Matches(score) {
		t.Errorf("Matches(empty conditions) = false, want true")
	}
}
