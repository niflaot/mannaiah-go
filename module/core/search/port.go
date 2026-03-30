package search

import (
	"context"
	"math"
)

// Operator enumerates supported filter operators for search fields.
type Operator string

const (
	// OpEQ matches exact equality.
	OpEQ Operator = "eq"
	// OpLike matches substring containment.
	OpLike Operator = "like"
	// OpIn matches membership in a list.
	OpIn Operator = "in"
	// OpBetween matches values within a range pair.
	OpBetween Operator = "between"
	// OpGT matches values strictly greater than threshold.
	OpGT Operator = "gt"
	// OpLT matches values strictly less than threshold.
	OpLT Operator = "lt"
	// OpGTE matches values greater than or equal to threshold.
	OpGTE Operator = "gte"
	// OpLTE matches values less than or equal to threshold.
	OpLTE Operator = "lte"
)

// Direction enumerates sort ordering directions.
type Direction string

const (
	// Asc sorts in ascending order.
	Asc Direction = "ASC"
	// Desc sorts in descending order.
	Desc Direction = "DESC"
)

// Filter represents a typed filter criterion.
type Filter struct {
	// Field is the column or field name.
	Field string
	// Operator is the comparison operator.
	Operator Operator
	// Value is the scalar, slice, or range pair used by the operator.
	Value any
}

// SortField represents a single sort instruction.
type SortField struct {
	// Field is the column or field name to order by.
	Field string
	// Direction is the ordering direction.
	Direction Direction
}

// Query is the unified search input for all resource types.
type Query struct {
	// Term is the free-text search token applied to configured text fields.
	Term string
	// Filters are typed field filters.
	Filters []Filter
	// Sort defines multi-field ordering instructions.
	Sort []SortField
	// Page is the 1-based page number.
	Page int
	// PageSize is the number of items per page (max 100, default 20).
	PageSize int
}

// Result is the unified paginated search response.
type Result[T any] struct {
	// Data holds the matched entities for the current page.
	Data []T `json:"data"`
	// Total is the overall number of matching records.
	Total int64 `json:"total"`
	// Page is the current 1-based page index.
	Page int `json:"page"`
	// PageSize is the number of items per page.
	PageSize int `json:"pageSize"`
	// TotalPages is the computed total page count.
	TotalPages int `json:"totalPages"`
}

const (
	// DefaultPage is the fallback page number.
	DefaultPage = 1
	// DefaultPageSize is the fallback page size.
	DefaultPageSize = 20
	// MaxPageSize is the hard ceiling for page size.
	MaxPageSize = 100
)

// NormalizePagination resolves page and page size defaults with upper bounds.
func NormalizePagination(page int, pageSize int) (int, int) {
	p := page
	if p <= 0 {
		p = DefaultPage
	}
	ps := pageSize
	if ps <= 0 {
		ps = DefaultPageSize
	}
	if ps > MaxPageSize {
		ps = MaxPageSize
	}
	return p, ps
}

// NewResult constructs a paginated search result from data, total count, and pagination values.
func NewResult[T any](data []T, total int64, page int, pageSize int) *Result[T] {
	p, ps := NormalizePagination(page, pageSize)
	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(ps)))
	}
	if data == nil {
		data = make([]T, 0)
	}
	return &Result[T]{
		Data:       data,
		Total:      total,
		Page:       p,
		PageSize:   ps,
		TotalPages: totalPages,
	}
}

// Repository defines the search port each module must implement.
type Repository[T any] interface {
	// Search executes a search query and returns paginated results.
	Search(ctx context.Context, query Query) (*Result[T], error)
}
