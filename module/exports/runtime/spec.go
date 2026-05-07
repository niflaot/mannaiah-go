package runtime

import "github.com/getkin/kin-openapi/openapi3"

const (
	exportsTag           = "exports"
	bearerSecurityScheme = "exports_bearer"
)

// OpenAPISpec returns export-module OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	components := openapi3.NewComponents()
	components.Schemas = openapi3.Schemas{
		"ExportReport":     &openapi3.SchemaRef{Value: exportReportSchema()},
		"ExportReportList": &openapi3.SchemaRef{Value: exportReportListSchema()},
	}
	components.SecuritySchemes = openapi3.SecuritySchemes{
		bearerSecurityScheme: &openapi3.SecuritySchemeRef{Value: openapi3.NewJWTSecurityScheme()},
	}

	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:   "Exports API",
			Version: "1.0.0",
		},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/exports/contacts", generateContactsPathItem()),
			openapi3.WithPath("/exports/orders", generateOrdersPathItem()),
			openapi3.WithPath("/export/orders", generateOrdersPathItem()),
			openapi3.WithPath("/exports/reports", reportsPathItem()),
			openapi3.WithPath("/exports/reports/{id}", reportByIDPathItem()),
			openapi3.WithPath("/exports/search", reportsSearchPathItem()),
		),
		Components: &components,
		Tags:       openapi3.Tags{&openapi3.Tag{Name: exportsTag}},
	}
}

func generateContactsPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{Post: generateOperation("ExportsController_generateContacts", "Generate contacts CSV export report")}
}

func generateOrdersPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{Post: generateOperation("ExportsController_generateOrders", "Generate orders CSV export report")}
}

func reportsPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{Get: listReportsOperation()}
}

func reportByIDPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{Get: getReportOperation()}
}

func reportsSearchPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{Get: searchReportsOperation()}
}

func generateOperation(operationID, summary string) *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: operationID,
		Summary:     summary,
		Tags:        []string{exportsTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(201, jsonResponse("Generated export report metadata.", "#/components/schemas/ExportReport")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(500, responseWithDescription("Generation or storage failure.")),
		),
	}
}

func listReportsOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ExportsController_listReports",
		Summary:     "List generated export reports",
		Tags:        []string{exportsTag},
		Security:    bearerSecurityRequirements(),
		Parameters:  listParameters(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Paginated export reports.", "#/components/schemas/ExportReportList")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		),
	}
}

func searchReportsOperation() *openapi3.Operation {
	operation := listReportsOperation()
	operation.OperationID = "ExportsController_searchReports"
	operation.Summary = "Search generated export reports by type"
	return operation
}

func getReportOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ExportsController_getReport",
		Summary:     "Get generated export report metadata",
		Tags:        []string{exportsTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathParameter("id", "Report ID", openapi3.NewStringSchema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Export report metadata.", "#/components/schemas/ExportReport")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("Export report not found.")),
		),
	}
}

func listParameters() openapi3.Parameters {
	return openapi3.Parameters{
		queryParameter("page", "Page number (default: 1)", openapi3.NewIntegerSchema()),
		queryParameter("limit", "Items per page (default: 50, max: 500)", openapi3.NewIntegerSchema()),
		queryParameter("type", "Filter by report type (contacts or orders)", reportTypeSchema()),
	}
}

func exportReportSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("id", openapi3.NewStringSchema()).
		WithProperty("type", reportTypeSchema()).
		WithProperty("status", openapi3.NewStringSchema().WithEnum("completed")).
		WithProperty("stamp", openapi3.NewStringSchema()).
		WithProperty("fileName", openapi3.NewStringSchema()).
		WithProperty("storageKey", openapi3.NewStringSchema()).
		WithProperty("sha256", openapi3.NewStringSchema()).
		WithProperty("contentType", openapi3.NewStringSchema()).
		WithProperty("rowCount", openapi3.NewIntegerSchema()).
		WithProperty("byteSize", openapi3.NewIntegerSchema()).
		WithProperty("generatedAt", openapi3.NewDateTimeSchema()).
		WithProperty("createdAt", openapi3.NewDateTimeSchema()).
		WithProperty("updatedAt", openapi3.NewDateTimeSchema())
}

func exportReportListSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("data", openapi3.NewArraySchema().WithItems(exportReportSchema())).
		WithProperty("page", openapi3.NewIntegerSchema()).
		WithProperty("limit", openapi3.NewIntegerSchema()).
		WithProperty("total", openapi3.NewIntegerSchema()).
		WithProperty("totalPages", openapi3.NewIntegerSchema())
}

func reportTypeSchema() *openapi3.Schema {
	return openapi3.NewStringSchema().WithEnum("contacts", "orders")
}

func bearerSecurityRequirements() *openapi3.SecurityRequirements {
	return openapi3.NewSecurityRequirements().With(openapi3.NewSecurityRequirement().Authenticate(bearerSecurityScheme))
}

func responseWithDescription(description string) *openapi3.ResponseRef {
	return &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription(description)}
}

func jsonResponse(description, schemaRef string) *openapi3.ResponseRef {
	return &openapi3.ResponseRef{
		Value: openapi3.NewResponse().WithDescription(description).WithContent(openapi3.Content{
			"application/json": &openapi3.MediaType{Schema: &openapi3.SchemaRef{Ref: schemaRef}},
		}),
	}
}

func queryParameter(name, description string, schema *openapi3.Schema) *openapi3.ParameterRef {
	return &openapi3.ParameterRef{Value: openapi3.NewQueryParameter(name).WithDescription(description).WithSchema(schema)}
}

func pathParameter(name, description string, schema *openapi3.Schema) *openapi3.ParameterRef {
	return &openapi3.ParameterRef{Value: openapi3.NewPathParameter(name).WithDescription(description).WithSchema(schema)}
}
