package search

import "github.com/getkin/kin-openapi/openapi3"

const searchTag = "search"

// OpenAPISpec returns the search-module OpenAPI specification covering all
// resource search endpoints and the spotlight endpoint.
func OpenAPISpec() *openapi3.T {
	components := openapi3.NewComponents()
	components.Schemas = openapi3.Schemas{
		"SearchResult":    &openapi3.SchemaRef{Value: searchResultSchema()},
		"SpotlightResult": &openapi3.SchemaRef{Value: spotlightResultSchema()},
		"SpotlightHit":    &openapi3.SchemaRef{Value: spotlightHitSchema()},
	}

	paths := openapi3.NewPaths()
	resources := []struct {
		path            string
		tag             string
		desc            string
		extraParameters openapi3.Parameters
	}{
		{path: "/search/contacts", tag: "contacts", desc: "Search contacts"},
		{path: "/search/orders", tag: "orders", desc: "Search orders"},
		{path: "/search/products", tag: "products", desc: "Search products"},
		{path: "/search/categories", tag: "categories", desc: "Search categories"},
		{path: "/search/variations", tag: "variations", desc: "Search product variations"},
		{path: "/search/tags", tag: "tags", desc: "Search product tags"},
		{path: "/search/shipping", tag: "shipping", desc: "Search shipping marks"},
		{path: "/search/campaigns", tag: "campaigns", desc: "Search campaigns"},
		{path: "/search/coupons", tag: "coupons", desc: "Search coupons", extraParameters: couponSearchParameters()},
		{path: "/search/segments", tag: "segments", desc: "Search segments"},
	}

	for _, r := range resources {
		paths.Set(r.path, resourceSearchPathItem(r.tag, r.desc, r.extraParameters))
	}
	paths.Set("/search", spotlightPathItem())

	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:   "Search API",
			Version: "1.1.0",
		},
		Paths:      paths,
		Components: &components,
		Tags: openapi3.Tags{
			&openapi3.Tag{Name: searchTag, Description: "Unified search and spotlight endpoints"},
		},
	}
}

func resourceSearchPathItem(tag string, summary string, extraParameters openapi3.Parameters) *openapi3.PathItem {
	parameters := searchQueryParameters()
	parameters = append(parameters, extraParameters...)

	return &openapi3.PathItem{
		Get: &openapi3.Operation{
			OperationID: "Search_" + tag,
			Summary:     summary,
			Tags:        []string{tag, searchTag},
			Parameters:  parameters,
			Responses: openapi3.NewResponses(
				openapi3.WithStatus(200, &openapi3.ResponseRef{
					Value: openapi3.NewResponse().WithDescription("Paginated search results").WithJSONSchemaRef(&openapi3.SchemaRef{Ref: "#/components/schemas/SearchResult"}),
				}),
				openapi3.WithStatus(400, &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription("Invalid query parameters")}),
			),
		},
	}
}

// couponSearchParameters returns coupon-specific unified-search parameters.
func couponSearchParameters() openapi3.Parameters {
	return openapi3.Parameters{
		{Value: &openapi3.Parameter{
			Name:        "filter[discountType]",
			In:          "query",
			Schema:      &openapi3.SchemaRef{Value: openapi3.NewStringSchema().WithEnum("fixed", "percentage")},
			Description: "Exact discount type filter. Example: filter[discountType]=fixed",
		}},
	}
}

func spotlightPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{
		Get: &openapi3.Operation{
			OperationID: "Search_spotlight",
			Summary:     "Cross-resource spotlight search",
			Tags:        []string{searchTag},
			Parameters: openapi3.Parameters{
				{Value: &openapi3.Parameter{
					Name: "term", In: "query", Required: true,
					Schema:      &openapi3.SchemaRef{Value: openapi3.NewStringSchema()},
					Description: "Free-text search term",
				}},
				{Value: &openapi3.Parameter{
					Name: "types", In: "query",
					Schema:      &openapi3.SchemaRef{Value: openapi3.NewStringSchema()},
					Description: "Comma-separated resource types to include (e.g. contact,order,product)",
				}},
				{Value: &openapi3.Parameter{
					Name: "limit", In: "query",
					Schema:      &openapi3.SchemaRef{Value: openapi3.NewInt64Schema()},
					Description: "Maximum results per provider (default: 10, max: 50)",
				}},
			},
			Responses: openapi3.NewResponses(
				openapi3.WithStatus(200, &openapi3.ResponseRef{
					Value: openapi3.NewResponse().WithDescription("Spotlight search results").WithJSONSchemaRef(&openapi3.SchemaRef{Ref: "#/components/schemas/SpotlightResult"}),
				}),
				openapi3.WithStatus(400, &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription("Missing required term parameter")}),
			),
		},
	}
}

func searchQueryParameters() openapi3.Parameters {
	return openapi3.Parameters{
		{Value: &openapi3.Parameter{
			Name: "term", In: "query",
			Schema:      &openapi3.SchemaRef{Value: openapi3.NewStringSchema()},
			Description: "Free-text search term matched against configured text fields via LIKE",
		}},
		{Value: &openapi3.Parameter{
			Name: "page", In: "query",
			Schema:      &openapi3.SchemaRef{Value: openapi3.NewInt64Schema()},
			Description: "1-based page number (default: 1)",
		}},
		{Value: &openapi3.Parameter{
			Name: "pageSize", In: "query",
			Schema:      &openapi3.SchemaRef{Value: openapi3.NewInt64Schema()},
			Description: "Items per page (default: 20, max: 100)",
		}},
		{Value: &openapi3.Parameter{
			Name: "sort", In: "query",
			Schema:      &openapi3.SchemaRef{Value: openapi3.NewStringSchema()},
			Description: "Sort fields as field:dir pairs. Example: created_at:desc,name:asc",
		}},
		{Value: &openapi3.Parameter{
			Name: "filter[field]", In: "query",
			Schema:      &openapi3.SchemaRef{Value: openapi3.NewStringSchema()},
			Description: "Exact match filter. Example: filter[status]=ACTIVE",
		}},
		{Value: &openapi3.Parameter{
			Name: "filter[field.op]", In: "query",
			Schema:      &openapi3.SchemaRef{Value: openapi3.NewStringSchema()},
			Description: "Operator filter (gte,lte,gt,lt,between,in,like). Example: filter[created_at.gte]=2024-01-01",
		}},
	}
}

func searchResultSchema() *openapi3.Schema {
	return &openapi3.Schema{
		Type: &openapi3.Types{"object"},
		Properties: openapi3.Schemas{
			"data":       &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"array"}, Items: &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"object"}}}}},
			"total":      &openapi3.SchemaRef{Value: openapi3.NewInt64Schema()},
			"page":       &openapi3.SchemaRef{Value: openapi3.NewInt64Schema()},
			"pageSize":   &openapi3.SchemaRef{Value: openapi3.NewInt64Schema()},
			"totalPages": &openapi3.SchemaRef{Value: openapi3.NewInt64Schema()},
		},
	}
}

func spotlightResultSchema() *openapi3.Schema {
	return &openapi3.Schema{
		Type: &openapi3.Types{"object"},
		Properties: openapi3.Schemas{
			"results": &openapi3.SchemaRef{Value: &openapi3.Schema{
				Type:  &openapi3.Types{"array"},
				Items: &openapi3.SchemaRef{Ref: "#/components/schemas/SpotlightHit"},
			}},
			"meta": &openapi3.SchemaRef{Value: &openapi3.Schema{
				Type: &openapi3.Types{"object"},
				Properties: openapi3.Schemas{
					"term":   &openapi3.SchemaRef{Value: openapi3.NewStringSchema()},
					"tookMs": &openapi3.SchemaRef{Value: openapi3.NewInt64Schema()},
					"counts": &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"object"}, AdditionalProperties: openapi3.AdditionalProperties{Schema: &openapi3.SchemaRef{Value: openapi3.NewInt64Schema()}}}},
				},
			}},
		},
	}
}

func spotlightHitSchema() *openapi3.Schema {
	return &openapi3.Schema{
		Type: &openapi3.Types{"object"},
		Properties: openapi3.Schemas{
			"type":         &openapi3.SchemaRef{Value: openapi3.NewStringSchema()},
			"id":           &openapi3.SchemaRef{Value: openapi3.NewStringSchema()},
			"title":        &openapi3.SchemaRef{Value: openapi3.NewStringSchema()},
			"subtitle":     &openapi3.SchemaRef{Value: openapi3.NewStringSchema()},
			"matchedField": &openapi3.SchemaRef{Value: openapi3.NewStringSchema()},
			"score":        &openapi3.SchemaRef{Value: openapi3.NewFloat64Schema()},
		},
	}
}
