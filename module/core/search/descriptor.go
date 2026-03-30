package search

// Descriptor defines how a resource type is searchable.
type Descriptor struct {
	// TextFields are columns matched by the free-text Term via LIKE.
	TextFields []string
	// FilterableFields lists columns that accept typed filter operators.
	FilterableFields map[string][]Operator
	// SortableFields lists columns that accept ORDER BY.
	SortableFields []string
	// DefaultSort is the fallback sort when the caller provides none.
	DefaultSort SortField
	// Joins defines SQL JOIN clauses required for cross-table text search.
	Joins []JoinClause
}

// JoinClause describes a single SQL JOIN required for search.
type JoinClause struct {
	// Type is the join keyword (e.g. "LEFT JOIN", "JOIN").
	Type string
	// Table is the target table expression.
	Table string
	// On is the join condition.
	On string
}
