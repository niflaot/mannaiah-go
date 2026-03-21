package port

import "context"

// TagCorrelation defines one source → target tag correlation record.
type TagCorrelation struct {
	// TargetTag is the correlated product tag.
	TargetTag string
	// Probability is the configured cross-sell probability (0–100).
	Probability float64
}

// TagCorrelationStore defines read behavior over the tag_correlations table.
type TagCorrelationStore interface {
	// GetCorrelations returns all target tags correlated to any of the given source tags.
	// Returns an empty slice when sourceTags is empty or no correlations exist.
	GetCorrelations(ctx context.Context, sourceTags []string) ([]TagCorrelation, error)
}
