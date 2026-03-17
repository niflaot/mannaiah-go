package domain

import "time"

// RFMDimension identifies recency, frequency, or monetary scoring dimensions.
type RFMDimension string

const (
	// DimensionRecency identifies the recency RFM dimension.
	DimensionRecency RFMDimension = "recency"
	// DimensionFrequency identifies the frequency RFM dimension.
	DimensionFrequency RFMDimension = "frequency"
	// DimensionMonetary identifies the monetary RFM dimension.
	DimensionMonetary RFMDimension = "monetary"
)

// RFMBandConfig defines threshold configuration values for one RFM scoring dimension.
// Band5Min through Band2Min represent the minimum value (inclusive) to achieve each score.
// For ascending dimensions (frequency/monetary), higher values earn higher scores.
// For descending dimensions (recency), lower values earn higher scores.
type RFMBandConfig struct {
	// ID defines persistence identifier values.
	ID int64
	// Dimension identifies the scoring dimension.
	Dimension RFMDimension
	// Ascending indicates whether higher raw values produce higher band scores.
	// Set true for frequency and monetary; false for recency.
	Ascending bool
	// Band5Min defines the threshold for band-5 (best) qualification.
	Band5Min float64
	// Band4Min defines the threshold for band-4 qualification.
	Band4Min float64
	// Band3Min defines the threshold for band-3 qualification.
	Band3Min float64
	// Band2Min defines the threshold for band-2 qualification.
	Band2Min float64
	// UpdatedAt defines the last configuration update timestamp.
	UpdatedAt time.Time
}

// ScoreValue scores a raw measurement value against this band configuration.
// It returns an integer score from 1 (worst) to 5 (best).
func (b RFMBandConfig) ScoreValue(value float64) int {
	if b.Ascending {
		return scoreAscending(value, b.Band5Min, b.Band4Min, b.Band3Min, b.Band2Min)
	}

	return scoreDescending(value, b.Band5Min, b.Band4Min, b.Band3Min, b.Band2Min)
}

// scoreAscending scores a value where higher raw values earn higher bands.
func scoreAscending(value, band5, band4, band3, band2 float64) int {
	switch {
	case value >= band5 && band5 > 0:
		return 5
	case value >= band4 && band4 > 0:
		return 4
	case value >= band3 && band3 > 0:
		return 3
	case value >= band2 && band2 > 0:
		return 2
	default:
		return 1
	}
}

// scoreDescending scores a value where lower raw values earn higher bands (recency).
func scoreDescending(value, band5, band4, band3, band2 float64) int {
	switch {
	case value <= band5:
		return 5
	case value <= band4:
		return 4
	case value <= band3:
		return 3
	case value <= band2:
		return 2
	default:
		return 1
	}
}

// RFMScore defines computed RFM score values for one contact.
type RFMScore struct {
	// ContactID identifies the scored contact.
	ContactID string
	// RecencyDays is the number of days since the contact's last order.
	RecencyDays uint32
	// Frequency is the total number of distinct orders placed by the contact.
	Frequency uint32
	// Monetary is the total monetary spend by the contact.
	Monetary float64
	// RScore is the recency band score (1–5).
	RScore int
	// FScore is the frequency band score (1–5).
	FScore int
	// MScore is the monetary band score (1–5).
	MScore int
	// RFMTotal is the sum of R, F, and M band scores.
	RFMTotal int
}

// RFMGroupConditions defines optional RFM score range constraints for group membership.
// Nil pointer fields indicate that the corresponding constraint is unconstrained.
type RFMGroupConditions struct {
	// RMin defines the optional minimum R-score constraint.
	RMin *int
	// RMax defines the optional maximum R-score constraint.
	RMax *int
	// FMin defines the optional minimum F-score constraint.
	FMin *int
	// FMax defines the optional maximum F-score constraint.
	FMax *int
	// MMin defines the optional minimum M-score constraint.
	MMin *int
	// MMax defines the optional maximum M-score constraint.
	MMax *int
}

// Matches reports whether the given RFMScore satisfies all non-nil group conditions.
func (c RFMGroupConditions) Matches(score RFMScore) bool {
	if c.RMin != nil && score.RScore < *c.RMin {
		return false
	}
	if c.RMax != nil && score.RScore > *c.RMax {
		return false
	}
	if c.FMin != nil && score.FScore < *c.FMin {
		return false
	}
	if c.FMax != nil && score.FScore > *c.FMax {
		return false
	}
	if c.MMin != nil && score.MScore < *c.MMin {
		return false
	}
	if c.MMax != nil && score.MScore > *c.MMax {
		return false
	}

	return true
}

// RFMGroup defines a named RFM cohort driven by band-score range conditions.
type RFMGroup struct {
	// ID defines persistence identifier values.
	ID string
	// Name defines human-readable group names.
	Name string
	// Slug defines URL-safe group slug values.
	Slug string
	// Description defines optional group description values.
	Description string
	// Conditions defines RFM score range membership conditions.
	Conditions RFMGroupConditions
	// CreatedAt defines group creation timestamp values.
	CreatedAt time.Time
	// UpdatedAt defines group update timestamp values.
	UpdatedAt time.Time
}
