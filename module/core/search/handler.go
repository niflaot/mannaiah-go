package search

import (
	"strconv"
	"strings"

	corehttp "mannaiah/module/core/http"
)

// SearchHandlerFunc returns an HTTP handler that executes a search query against
// the provided Repository and returns a paginated JSON response.
func SearchHandlerFunc[T any](repo Repository[T]) corehttp.Handler {
	return func(ctx corehttp.Context) error {
		query := ParseQuery(ctx)
		result, err := repo.Search(ctx.Context(), query)
		if err != nil {
			return corehttp.NewAppError(500, "search_failed", err)
		}
		return ctx.Status(200).JSON(result)
	}
}

// SpotlightHandlerFunc returns an HTTP handler for the spotlight search endpoint.
func SpotlightHandlerFunc(service *SpotlightService) corehttp.Handler {
	return func(ctx corehttp.Context) error {
		term := strings.TrimSpace(ctx.Query("term", ""))
		if term == "" {
			return corehttp.NewAppError(400, "term_required", nil)
		}

		typesRaw := ctx.Query("types", "")
		var types []string
		if typesRaw != "" {
			for _, t := range strings.Split(typesRaw, ",") {
				if trimmed := strings.TrimSpace(t); trimmed != "" {
					types = append(types, trimmed)
				}
			}
		}

		limit := 10
		if v, err := strconv.Atoi(ctx.Query("limit", "10")); err == nil && v > 0 {
			limit = v
		}
		if limit > 50 {
			limit = 50
		}

		result := service.Search(ctx.Context(), term, types, limit)
		return ctx.Status(200).JSON(result)
	}
}
