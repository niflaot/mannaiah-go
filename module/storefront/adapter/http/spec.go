package http

import "github.com/getkin/kin-openapi/openapi3"

// Paths returns the OpenAPI path definitions for storefront endpoints.
func Paths() *openapi3.Paths {
	return openapi3.NewPaths(
		openapi3.WithPath("/storefront/renderable", &openapi3.PathItem{
			Post: &openapi3.Operation{
				Tags:        []string{"storefront"},
				Summary:     "Create renderable",
				Description: "Creates a new storefront renderable draft.",
				OperationID: "createStorefrontRenderable",
				Security:    storefrontSecurityRequirements(),
				RequestBody: jsonRequestBodyRef("#/components/schemas/CreateRenderableRequest", "Renderable creation payload"),
				Responses: openapi3.NewResponses(
					openapi3.WithStatus(201, schemaResponse("Renderable created", "#/components/schemas/Renderable")),
					openapi3.WithStatus(400, errorResponse("Validation error")),
					openapi3.WithStatus(401, errorResponse("Unauthorized")),
					openapi3.WithStatus(403, errorResponse("Forbidden")),
				),
			},
			Get: &openapi3.Operation{
				Tags:        []string{"storefront"},
				Summary:     "List renderables",
				Description: "Returns paginated storefront renderables filtered by kind or draft state.",
				OperationID: "listStorefrontRenderables",
				Security:    storefrontSecurityRequirements(),
				Parameters: openapi3.Parameters{
					{Value: openapi3.NewQueryParameter("kind").WithSchema(openapi3.NewStringSchema())},
					{Value: openapi3.NewQueryParameter("draft").WithSchema(openapi3.NewBoolSchema())},
					{Value: openapi3.NewQueryParameter("page").WithSchema(openapi3.NewIntegerSchema())},
					{Value: openapi3.NewQueryParameter("pageSize").WithSchema(openapi3.NewIntegerSchema())},
				},
				Responses: openapi3.NewResponses(
					openapi3.WithStatus(200, schemaResponse("Renderable list", "#/components/schemas/RenderableListResponse")),
					openapi3.WithStatus(401, errorResponse("Unauthorized")),
					openapi3.WithStatus(403, errorResponse("Forbidden")),
				),
			},
		}),
		openapi3.WithPath("/storefront/renderable/{id}", &openapi3.PathItem{
			Get: &openapi3.Operation{
				Tags:        []string{"storefront"},
				Summary:     "Get renderable",
				Description: "Retrieves one storefront renderable by identifier.",
				OperationID: "getStorefrontRenderable",
				Security:    storefrontSecurityRequirements(),
				Parameters:  openapi3.Parameters{{Value: openapi3.NewPathParameter("id").WithSchema(openapi3.NewStringSchema())}},
				Responses: openapi3.NewResponses(
					openapi3.WithStatus(200, schemaResponse("Renderable", "#/components/schemas/Renderable")),
					openapi3.WithStatus(404, errorResponse("Not found")),
					openapi3.WithStatus(401, errorResponse("Unauthorized")),
					openapi3.WithStatus(403, errorResponse("Forbidden")),
				),
			},
			Patch: &openapi3.Operation{
				Tags:        []string{"storefront"},
				Summary:     "Update renderable draft",
				Description: "Updates the current renderable working draft.",
				OperationID: "updateStorefrontRenderable",
				Security:    storefrontSecurityRequirements(),
				Parameters:  openapi3.Parameters{{Value: openapi3.NewPathParameter("id").WithSchema(openapi3.NewStringSchema())}},
				RequestBody: jsonRequestBodyRef("#/components/schemas/UpdateRenderableRequest", "Renderable update payload"),
				Responses: openapi3.NewResponses(
					openapi3.WithStatus(200, schemaResponse("Renderable updated", "#/components/schemas/Renderable")),
					openapi3.WithStatus(400, errorResponse("Validation error")),
					openapi3.WithStatus(404, errorResponse("Not found")),
					openapi3.WithStatus(401, errorResponse("Unauthorized")),
					openapi3.WithStatus(403, errorResponse("Forbidden")),
				),
			},
			Delete: &openapi3.Operation{
				Tags:        []string{"storefront"},
				Summary:     "Delete renderable",
				Description: "Deletes one renderable and its dependent published versions and bound page.",
				OperationID: "deleteStorefrontRenderable",
				Security:    storefrontSecurityRequirements(),
				Parameters:  openapi3.Parameters{{Value: openapi3.NewPathParameter("id").WithSchema(openapi3.NewStringSchema())}},
				Responses: openapi3.NewResponses(
					openapi3.WithStatus(204, &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription("Deleted")}),
					openapi3.WithStatus(404, errorResponse("Not found")),
					openapi3.WithStatus(401, errorResponse("Unauthorized")),
					openapi3.WithStatus(403, errorResponse("Forbidden")),
				),
			},
		}),
		openapi3.WithPath("/storefront/renderable/{id}/publish", &openapi3.PathItem{
			Post: &openapi3.Operation{
				Tags:        []string{"storefront"},
				Summary:     "Publish renderable",
				Description: "Creates a new immutable published snapshot from the current renderable draft.",
				OperationID: "publishStorefrontRenderable",
				Security:    storefrontSecurityRequirements(),
				Parameters:  openapi3.Parameters{{Value: openapi3.NewPathParameter("id").WithSchema(openapi3.NewStringSchema())}},
				Responses: openapi3.NewResponses(
					openapi3.WithStatus(201, schemaResponse("Published version", "#/components/schemas/RenderableVersion")),
					openapi3.WithStatus(404, errorResponse("Not found")),
					openapi3.WithStatus(401, errorResponse("Unauthorized")),
					openapi3.WithStatus(403, errorResponse("Forbidden")),
				),
			},
		}),
		openapi3.WithPath("/storefront/renderable/{id}/versions", &openapi3.PathItem{
			Get: &openapi3.Operation{
				Tags:        []string{"storefront"},
				Summary:     "List renderable versions",
				Description: "Returns paginated published versions for one renderable.",
				OperationID: "listStorefrontRenderableVersions",
				Security:    storefrontSecurityRequirements(),
				Parameters: openapi3.Parameters{
					{Value: openapi3.NewPathParameter("id").WithSchema(openapi3.NewStringSchema())},
					{Value: openapi3.NewQueryParameter("page").WithSchema(openapi3.NewIntegerSchema())},
					{Value: openapi3.NewQueryParameter("pageSize").WithSchema(openapi3.NewIntegerSchema())},
				},
				Responses: openapi3.NewResponses(
					openapi3.WithStatus(200, schemaResponse("Renderable version list", "#/components/schemas/RenderableVersionListResponse")),
					openapi3.WithStatus(404, errorResponse("Not found")),
					openapi3.WithStatus(401, errorResponse("Unauthorized")),
					openapi3.WithStatus(403, errorResponse("Forbidden")),
				),
			},
		}),
		openapi3.WithPath("/storefront/renderable/{id}/versions/{versionId}", &openapi3.PathItem{
			Get: &openapi3.Operation{
				Tags:        []string{"storefront"},
				Summary:     "Get renderable version",
				Description: "Retrieves one published renderable version.",
				OperationID: "getStorefrontRenderableVersion",
				Security:    storefrontSecurityRequirements(),
				Parameters: openapi3.Parameters{
					{Value: openapi3.NewPathParameter("id").WithSchema(openapi3.NewStringSchema())},
					{Value: openapi3.NewPathParameter("versionId").WithSchema(openapi3.NewStringSchema())},
				},
				Responses: openapi3.NewResponses(
					openapi3.WithStatus(200, schemaResponse("Renderable version", "#/components/schemas/RenderableVersion")),
					openapi3.WithStatus(404, errorResponse("Not found")),
					openapi3.WithStatus(401, errorResponse("Unauthorized")),
					openapi3.WithStatus(403, errorResponse("Forbidden")),
				),
			},
		}),
		openapi3.WithPath("/storefront/renderable/{id}/versions/{versionId}/rollback", &openapi3.PathItem{
			Post: &openapi3.Operation{
				Tags:        []string{"storefront"},
				Summary:     "Rollback renderable version",
				Description: "Creates a fresh published snapshot from one historical renderable version.",
				OperationID: "rollbackStorefrontRenderableVersion",
				Security:    storefrontSecurityRequirements(),
				Parameters: openapi3.Parameters{
					{Value: openapi3.NewPathParameter("id").WithSchema(openapi3.NewStringSchema())},
					{Value: openapi3.NewPathParameter("versionId").WithSchema(openapi3.NewStringSchema())},
				},
				Responses: openapi3.NewResponses(
					openapi3.WithStatus(201, schemaResponse("Rollback version", "#/components/schemas/RenderableVersion")),
					openapi3.WithStatus(404, errorResponse("Not found")),
					openapi3.WithStatus(401, errorResponse("Unauthorized")),
					openapi3.WithStatus(403, errorResponse("Forbidden")),
				),
			},
		}),
		openapi3.WithPath("/storefront/page", &openapi3.PathItem{
			Post: &openapi3.Operation{
				Tags:        []string{"storefront"},
				Summary:     "Create static page",
				Description: "Creates a storefront static page bound to an existing static-page renderable.",
				OperationID: "createStorefrontPage",
				Security:    storefrontSecurityRequirements(),
				RequestBody: jsonRequestBodyRef("#/components/schemas/CreateStaticPageRequest", "Static-page creation payload"),
				Responses: openapi3.NewResponses(
					openapi3.WithStatus(201, schemaResponse("Static page created", "#/components/schemas/StaticPage")),
					openapi3.WithStatus(400, errorResponse("Validation error")),
					openapi3.WithStatus(409, errorResponse("Conflict")),
					openapi3.WithStatus(401, errorResponse("Unauthorized")),
					openapi3.WithStatus(403, errorResponse("Forbidden")),
				),
			},
			Get: &openapi3.Operation{
				Tags:        []string{"storefront"},
				Summary:     "List static pages",
				Description: "Returns paginated active static pages by default, optionally filtered by term, bound renderable, or archived state.",
				OperationID: "listStorefrontPages",
				Security:    storefrontSecurityRequirements(),
				Parameters: openapi3.Parameters{
					{Value: openapi3.NewQueryParameter("term").WithSchema(openapi3.NewStringSchema())},
					{Value: openapi3.NewQueryParameter("renderableId").WithSchema(openapi3.NewStringSchema())},
					{Value: openapi3.NewQueryParameter("archived").WithSchema(openapi3.NewBoolSchema())},
					{Value: openapi3.NewQueryParameter("page").WithSchema(openapi3.NewIntegerSchema())},
					{Value: openapi3.NewQueryParameter("pageSize").WithSchema(openapi3.NewIntegerSchema())},
				},
				Responses: openapi3.NewResponses(
					openapi3.WithStatus(200, schemaResponse("Static page list", "#/components/schemas/StaticPageListResponse")),
					openapi3.WithStatus(401, errorResponse("Unauthorized")),
					openapi3.WithStatus(403, errorResponse("Forbidden")),
				),
			},
		}),
		openapi3.WithPath("/storefront/page/{id}", &openapi3.PathItem{
			Get: &openapi3.Operation{
				Tags:        []string{"storefront"},
				Summary:     "Get static page",
				Description: "Retrieves one storefront static page by identifier.",
				OperationID: "getStorefrontPage",
				Security:    storefrontSecurityRequirements(),
				Parameters:  openapi3.Parameters{{Value: openapi3.NewPathParameter("id").WithSchema(openapi3.NewStringSchema())}},
				Responses: openapi3.NewResponses(
					openapi3.WithStatus(200, schemaResponse("Static page", "#/components/schemas/StaticPage")),
					openapi3.WithStatus(404, errorResponse("Not found")),
					openapi3.WithStatus(401, errorResponse("Unauthorized")),
					openapi3.WithStatus(403, errorResponse("Forbidden")),
				),
			},
			Patch: &openapi3.Operation{
				Tags:        []string{"storefront"},
				Summary:     "Update static page",
				Description: "Updates storefront static-page metadata and renderable binding.",
				OperationID: "updateStorefrontPage",
				Security:    storefrontSecurityRequirements(),
				Parameters:  openapi3.Parameters{{Value: openapi3.NewPathParameter("id").WithSchema(openapi3.NewStringSchema())}},
				RequestBody: jsonRequestBodyRef("#/components/schemas/UpdateStaticPageRequest", "Static-page update payload"),
				Responses: openapi3.NewResponses(
					openapi3.WithStatus(200, schemaResponse("Static page updated", "#/components/schemas/StaticPage")),
					openapi3.WithStatus(400, errorResponse("Validation error")),
					openapi3.WithStatus(404, errorResponse("Not found")),
					openapi3.WithStatus(409, errorResponse("Conflict")),
					openapi3.WithStatus(401, errorResponse("Unauthorized")),
					openapi3.WithStatus(403, errorResponse("Forbidden")),
				),
			},
			Delete: &openapi3.Operation{
				Tags:        []string{"storefront"},
				Summary:     "Delete static page",
				Description: "Deletes one storefront static page.",
				OperationID: "deleteStorefrontPage",
				Security:    storefrontSecurityRequirements(),
				Parameters:  openapi3.Parameters{{Value: openapi3.NewPathParameter("id").WithSchema(openapi3.NewStringSchema())}},
				Responses: openapi3.NewResponses(
					openapi3.WithStatus(204, &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription("Deleted")}),
					openapi3.WithStatus(404, errorResponse("Not found")),
					openapi3.WithStatus(401, errorResponse("Unauthorized")),
					openapi3.WithStatus(403, errorResponse("Forbidden")),
				),
			},
		}),
		openapi3.WithPath("/storefront/page/{id}/archive", &openapi3.PathItem{
			Post: &openapi3.Operation{
				Tags:        []string{"storefront"},
				Summary:     "Archive static page",
				Description: "Archives one storefront static page without deleting bound renderable history.",
				OperationID: "archiveStorefrontPage",
				Security:    storefrontSecurityRequirements(),
				Parameters:  openapi3.Parameters{{Value: openapi3.NewPathParameter("id").WithSchema(openapi3.NewStringSchema())}},
				Responses: openapi3.NewResponses(
					openapi3.WithStatus(200, schemaResponse("Static page archived", "#/components/schemas/StaticPage")),
					openapi3.WithStatus(404, errorResponse("Not found")),
					openapi3.WithStatus(401, errorResponse("Unauthorized")),
					openapi3.WithStatus(403, errorResponse("Forbidden")),
				),
			},
		}),
	)
}

// SecuritySchemes returns the OpenAPI security scheme definitions for storefront endpoints.
func SecuritySchemes() openapi3.SecuritySchemes {
	return openapi3.SecuritySchemes{
		"storefront_bearer": &openapi3.SecuritySchemeRef{Value: openapi3.NewJWTSecurityScheme()},
	}
}

// Tags returns the OpenAPI tag definitions for storefront endpoints.
func Tags() openapi3.Tags {
	return openapi3.Tags{
		{Name: "storefront", Description: "Storefront renderable and static-page management"},
	}
}

// Schemas returns the OpenAPI component schemas for storefront types.
func Schemas() openapi3.Schemas {
	jsonObject := openapi3.NewObjectSchema()
	allowsAdditionalProperties := true
	jsonObject.AdditionalProperties = openapi3.AdditionalProperties{Has: &allowsAdditionalProperties}

	renderable := openapi3.NewObjectSchema().WithProperties(map[string]*openapi3.Schema{
		"id":                       openapi3.NewStringSchema(),
		"kind":                     openapi3.NewStringSchema(),
		"metadata":                 jsonObject,
		"content":                  openapi3.NewSchema(),
		"draft":                    openapi3.NewBoolSchema(),
		"snapshotHash":             openapi3.NewStringSchema(),
		"latestPublishedVersionId": openapi3.NewStringSchema(),
		"latestPublishedAt":        openapi3.NewStringSchema().WithFormat("date-time"),
		"createdAt":                openapi3.NewStringSchema().WithFormat("date-time"),
		"updatedAt":                openapi3.NewStringSchema().WithFormat("date-time"),
	})

	version := openapi3.NewObjectSchema().WithProperties(map[string]*openapi3.Schema{
		"id":              openapi3.NewStringSchema(),
		"renderableId":    openapi3.NewStringSchema(),
		"sourceVersionId": openapi3.NewStringSchema(),
		"metadata":        jsonObject,
		"content":         openapi3.NewSchema(),
		"snapshotHash":    openapi3.NewStringSchema(),
		"publishedAt":     openapi3.NewStringSchema().WithFormat("date-time"),
	})

	staticPage := openapi3.NewObjectSchema().WithProperties(map[string]*openapi3.Schema{
		"id":           openapi3.NewStringSchema(),
		"renderableId": openapi3.NewStringSchema(),
		"title":        openapi3.NewStringSchema(),
		"url":          openapi3.NewStringSchema(),
		"seoTags":      jsonObject,
		"archivedAt":   openapi3.NewStringSchema().WithFormat("date-time"),
		"createdAt":    openapi3.NewStringSchema().WithFormat("date-time"),
		"updatedAt":    openapi3.NewStringSchema().WithFormat("date-time"),
	})

	renderableReq := openapi3.NewObjectSchema().WithRequired([]string{"kind", "metadata", "content"}).WithProperties(map[string]*openapi3.Schema{
		"kind":     openapi3.NewStringSchema(),
		"metadata": jsonObject,
		"content":  openapi3.NewSchema(),
	})

	renderableUpdateReq := openapi3.NewObjectSchema().WithRequired([]string{"metadata", "content"}).WithProperties(map[string]*openapi3.Schema{
		"metadata": jsonObject,
		"content":  openapi3.NewSchema(),
	})

	pageReq := openapi3.NewObjectSchema().WithRequired([]string{"renderableId", "title", "url", "seoTags"}).WithProperties(map[string]*openapi3.Schema{
		"renderableId": openapi3.NewStringSchema(),
		"title":        openapi3.NewStringSchema(),
		"url":          openapi3.NewStringSchema(),
		"seoTags":      jsonObject,
	})

	renderableList := listSchema(renderable)
	versionList := listSchema(version)
	pageList := listSchema(staticPage)

	return openapi3.Schemas{
		"Renderable":                    openapi3.NewSchemaRef("", renderable),
		"RenderableVersion":             openapi3.NewSchemaRef("", version),
		"StaticPage":                    openapi3.NewSchemaRef("", staticPage),
		"CreateRenderableRequest":       openapi3.NewSchemaRef("", renderableReq),
		"UpdateRenderableRequest":       openapi3.NewSchemaRef("", renderableUpdateReq),
		"CreateStaticPageRequest":       openapi3.NewSchemaRef("", pageReq),
		"UpdateStaticPageRequest":       openapi3.NewSchemaRef("", pageReq),
		"RenderableListResponse":        openapi3.NewSchemaRef("", renderableList),
		"RenderableVersionListResponse": openapi3.NewSchemaRef("", versionList),
		"StaticPageListResponse":        openapi3.NewSchemaRef("", pageList),
		"StorefrontError":               openapi3.NewSchemaRef("", errorSchema()),
	}
}

// storefrontSecurityRequirements returns bearer auth requirements for storefront endpoints.
func storefrontSecurityRequirements() *openapi3.SecurityRequirements {
	return openapi3.NewSecurityRequirements().With(openapi3.NewSecurityRequirement().Authenticate("storefront_bearer"))
}

// jsonRequestBodyRef creates a JSON request body reference.
func jsonRequestBodyRef(schemaRef string, description string) *openapi3.RequestBodyRef {
	return &openapi3.RequestBodyRef{Value: openapi3.NewRequestBody().WithDescription(description).WithRequired(true).WithJSONSchemaRef(openapi3.NewSchemaRef(schemaRef, nil))}
}

// schemaResponse creates a schema-backed response reference.
func schemaResponse(description string, schemaRef string) *openapi3.ResponseRef {
	return &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription(description).WithJSONSchemaRef(openapi3.NewSchemaRef(schemaRef, nil))}
}

// errorResponse creates an error response reference.
func errorResponse(description string) *openapi3.ResponseRef {
	return &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription(description).WithJSONSchemaRef(openapi3.NewSchemaRef("#/components/schemas/StorefrontError", nil))}
}

// listSchema creates a paginated list schema for one item schema.
func listSchema(itemSchema *openapi3.Schema) *openapi3.Schema {
	return openapi3.NewObjectSchema().WithProperties(map[string]*openapi3.Schema{
		"data":       openapi3.NewArraySchema().WithItems(itemSchema),
		"total":      openapi3.NewInt64Schema(),
		"page":       openapi3.NewInt64Schema(),
		"pageSize":   openapi3.NewInt64Schema(),
		"totalPages": openapi3.NewInt64Schema(),
	})
}

// errorSchema defines the minimal storefront error payload schema.
func errorSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().WithProperties(map[string]*openapi3.Schema{
		"error":   openapi3.NewStringSchema(),
		"message": openapi3.NewStringSchema(),
	})
}
