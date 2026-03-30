package search

import (
	"strings"
)

const (
	scoreExactMatch   = 1.0
	scorePrefixMatch  = 0.7
	scoreContains     = 0.4
	boostPrimaryField = 0.3
	boostOtherField   = 0.1
)

// ScoredHit wraps a search result with a relevance score and match metadata.
type ScoredHit[T any] struct {
	// Entity is the matched domain object.
	Entity T `json:"entity"`
	// Score is the computed relevance score.
	Score float64 `json:"score"`
	// MatchedField is the field that best matched the term.
	MatchedField string `json:"matchedField"`
}

// FieldExtractor returns the string value of a named field from an entity.
type FieldExtractor[T any] func(entity T, field string) string

// ScoreResults scores a slice of entities against a search term using field extractors.
// primaryFields receive a higher boost than secondaryFields.
func ScoreResults[T any](entities []T, term string, primaryFields []string, secondaryFields []string, extract FieldExtractor[T]) []ScoredHit[T] {
	if len(entities) == 0 || strings.TrimSpace(term) == "" {
		hits := make([]ScoredHit[T], len(entities))
		for i, e := range entities {
			hits[i] = ScoredHit[T]{Entity: e, Score: 0}
		}
		return hits
	}

	lower := strings.ToLower(strings.TrimSpace(term))
	hits := make([]ScoredHit[T], 0, len(entities))

	for _, entity := range entities {
		bestScore := 0.0
		bestField := ""

		for _, field := range primaryFields {
			val := strings.ToLower(extract(entity, field))
			if val == "" {
				continue
			}
			s := computeFieldScore(lower, val) + boostPrimaryField
			if s > bestScore {
				bestScore = s
				bestField = field
			}
		}

		for _, field := range secondaryFields {
			val := strings.ToLower(extract(entity, field))
			if val == "" {
				continue
			}
			s := computeFieldScore(lower, val) + boostOtherField
			if s > bestScore {
				bestScore = s
				bestField = field
			}
		}

		hits = append(hits, ScoredHit[T]{
			Entity:       entity,
			Score:        bestScore,
			MatchedField: bestField,
		})
	}

	return hits
}

// computeFieldScore computes a raw relevance score for a term against a field value.
func computeFieldScore(term string, value string) float64 {
	if value == term {
		return scoreExactMatch
	}
	if strings.HasPrefix(value, term) {
		return scorePrefixMatch
	}
	if strings.Contains(value, term) {
		return scoreContains
	}
	return 0
}
