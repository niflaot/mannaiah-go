package search

import (
	"context"
	"sort"
	"sync"
	"time"
)

// SpotlightHit represents a single search hit in the spotlight response.
type SpotlightHit struct {
	// Type identifies the resource kind (e.g. "contact", "order").
	Type string `json:"type"`
	// ID is the resource identifier.
	ID string `json:"id"`
	// Title is the display title.
	Title string `json:"title"`
	// Subtitle is the display subtitle.
	Subtitle string `json:"subtitle"`
	// MatchedField is the field that best matched the query.
	MatchedField string `json:"matchedField"`
	// Score is the relevance score.
	Score float64 `json:"score"`
}

// SpotlightCounts maps resource types to their hit counts.
type SpotlightCounts map[string]int

// SpotlightResult is the unified spotlight response.
type SpotlightResult struct {
	// Results holds the scored, merged, and sorted hits.
	Results []SpotlightHit `json:"results"`
	// Meta contains query metadata.
	Meta SpotlightMeta `json:"meta"`
}

// SpotlightMeta contains spotlight response metadata.
type SpotlightMeta struct {
	// Term is the original search term.
	Term string `json:"term"`
	// TookMs is the query execution time in milliseconds.
	TookMs int64 `json:"took_ms"`
	// Counts maps resource types to their result count.
	Counts SpotlightCounts `json:"counts"`
}

// SpotlightProvider is implemented by each module to contribute spotlight results.
type SpotlightProvider interface {
	// SpotlightSearch returns scored hits for the given term.
	SpotlightSearch(ctx context.Context, term string, limit int) ([]SpotlightHit, error)
	// SpotlightType returns the resource type identifier.
	SpotlightType() string
}

// SpotlightService orchestrates cross-module spotlight searching.
type SpotlightService struct {
	providers []SpotlightProvider
	timeout   time.Duration
}

// NewSpotlightService creates a spotlight service with the given providers and query timeout.
func NewSpotlightService(timeout time.Duration, providers ...SpotlightProvider) *SpotlightService {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return &SpotlightService{providers: providers, timeout: timeout}
}

// Search fans out to all providers concurrently and merges results by score.
func (s *SpotlightService) Search(ctx context.Context, term string, types []string, limit int) *SpotlightResult {
	start := time.Now()
	if limit <= 0 {
		limit = 5
	}
	if limit > 10 {
		limit = 10
	}

	typeSet := buildTypeSet(types)
	active := s.activeProviders(typeSet)

	var mu sync.Mutex
	var allHits []SpotlightHit
	counts := make(SpotlightCounts)

	searchCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	var wg sync.WaitGroup
	for _, provider := range active {
		wg.Add(1)
		go func(p SpotlightProvider) {
			defer wg.Done()
			hits, err := p.SpotlightSearch(searchCtx, term, limit)
			if err != nil {
				return
			}
			mu.Lock()
			defer mu.Unlock()
			allHits = append(allHits, hits...)
			counts[p.SpotlightType()] = len(hits)
		}(provider)
	}
	wg.Wait()

	sort.Slice(allHits, func(i, j int) bool {
		return allHits[i].Score > allHits[j].Score
	})

	took := time.Since(start).Milliseconds()
	return &SpotlightResult{
		Results: allHits,
		Meta: SpotlightMeta{
			Term:   term,
			TookMs: took,
			Counts: counts,
		},
	}
}

// AddProvider registers a spotlight provider at runtime.
func (s *SpotlightService) AddProvider(p SpotlightProvider) {
	if p == nil {
		return
	}
	s.providers = append(s.providers, p)
}

// activeProviders filters providers by the requested type set.
func (s *SpotlightService) activeProviders(typeSet map[string]bool) []SpotlightProvider {
	if len(typeSet) == 0 {
		return s.providers
	}
	active := make([]SpotlightProvider, 0, len(s.providers))
	for _, p := range s.providers {
		if typeSet[p.SpotlightType()] {
			active = append(active, p)
		}
	}
	return active
}

// buildTypeSet converts a type filter slice into a lookup map.
func buildTypeSet(types []string) map[string]bool {
	if len(types) == 0 {
		return nil
	}
	m := make(map[string]bool, len(types))
	for _, t := range types {
		m[t] = true
	}
	return m
}
