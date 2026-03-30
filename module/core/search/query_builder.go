package search

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// BuildGORMQuery translates a search Query and Descriptor into GORM chain operations.
// It returns a base query (for counting) and a paginated query (for data retrieval).
func BuildGORMQuery(tx *gorm.DB, query Query, desc Descriptor) (base *gorm.DB, paginated *gorm.DB) {
	base = applyJoins(tx, desc)
	base = applyTextSearch(base, query.Term, desc.TextFields)
	base = applyFilters(base, query.Filters, desc.FilterableFields)

	page, pageSize := NormalizePagination(query.Page, query.PageSize)
	sort := resolveSort(query.Sort, desc)

	paginated = base.Session(&gorm.Session{})
	for _, s := range sort {
		paginated = paginated.Order(fmt.Sprintf("%s %s", s.Field, s.Direction))
	}
	paginated = paginated.Offset((page - 1) * pageSize).Limit(pageSize)

	return base, paginated
}

// applyJoins appends descriptor-defined JOIN clauses.
func applyJoins(tx *gorm.DB, desc Descriptor) *gorm.DB {
	next := tx
	for _, j := range desc.Joins {
		clause := fmt.Sprintf("%s %s ON %s", j.Type, j.Table, j.On)
		next = next.Joins(clause)
	}
	return next
}

// applyTextSearch adds OR-chained LIKE conditions for the free-text term.
func applyTextSearch(tx *gorm.DB, term string, textFields []string) *gorm.DB {
	trimmed := strings.TrimSpace(term)
	if trimmed == "" || len(textFields) == 0 {
		return tx
	}

	clauses := make([]string, 0, len(textFields))
	args := make([]any, 0, len(textFields))
	pattern := "%" + escapeLike(trimmed) + "%"

	for _, field := range textFields {
		clauses = append(clauses, fmt.Sprintf("%s LIKE ?", field))
		args = append(args, pattern)
	}

	return tx.Where(strings.Join(clauses, " OR "), args...)
}

// applyFilters adds typed WHERE conditions for each filter.
func applyFilters(tx *gorm.DB, filters []Filter, allowed map[string][]Operator) *gorm.DB {
	next := tx
	for _, f := range filters {
		if !isOperatorAllowed(f.Field, f.Operator, allowed) {
			continue
		}
		next = applyFilter(next, f)
	}
	return next
}

// applyFilter applies a single filter to the query.
func applyFilter(tx *gorm.DB, f Filter) *gorm.DB {
	switch f.Operator {
	case OpEQ:
		return tx.Where(fmt.Sprintf("%s = ?", f.Field), f.Value)
	case OpLike:
		return tx.Where(fmt.Sprintf("%s LIKE ?", f.Field), f.Value)
	case OpIn:
		return tx.Where(fmt.Sprintf("%s IN ?", f.Field), f.Value)
	case OpBetween:
		pair, ok := f.Value.([2]any)
		if !ok {
			return tx
		}
		return tx.Where(fmt.Sprintf("%s BETWEEN ? AND ?", f.Field), pair[0], pair[1])
	case OpGT:
		return tx.Where(fmt.Sprintf("%s > ?", f.Field), f.Value)
	case OpLT:
		return tx.Where(fmt.Sprintf("%s < ?", f.Field), f.Value)
	case OpGTE:
		return tx.Where(fmt.Sprintf("%s >= ?", f.Field), f.Value)
	case OpLTE:
		return tx.Where(fmt.Sprintf("%s <= ?", f.Field), f.Value)
	default:
		return tx
	}
}

// isOperatorAllowed checks whether a field+operator combination is permitted.
func isOperatorAllowed(field string, op Operator, allowed map[string][]Operator) bool {
	if allowed == nil {
		return false
	}
	ops, ok := allowed[field]
	if !ok {
		return false
	}
	for _, o := range ops {
		if o == op {
			return true
		}
	}
	return false
}

// resolveSort returns the effective sort fields, falling back to the descriptor default.
func resolveSort(requested []SortField, desc Descriptor) []SortField {
	if len(requested) == 0 {
		if desc.DefaultSort.Field != "" {
			return []SortField{desc.DefaultSort}
		}
		return []SortField{{Field: "created_at", Direction: Desc}}
	}
	valid := make([]SortField, 0, len(requested))
	for _, s := range requested {
		if isSortable(s.Field, desc.SortableFields) {
			dir := s.Direction
			if dir != Asc && dir != Desc {
				dir = Desc
			}
			valid = append(valid, SortField{Field: s.Field, Direction: dir})
		}
	}
	if len(valid) == 0 {
		if desc.DefaultSort.Field != "" {
			return []SortField{desc.DefaultSort}
		}
		return []SortField{{Field: "created_at", Direction: Desc}}
	}
	return valid
}

// isSortable checks if a field is in the allowed sortable list.
func isSortable(field string, sortable []string) bool {
	for _, s := range sortable {
		if s == field {
			return true
		}
	}
	return false
}

// escapeLike escapes SQL LIKE special characters.
func escapeLike(s string) string {
	r := strings.NewReplacer("%", "\\%", "_", "\\_")
	return r.Replace(s)
}
