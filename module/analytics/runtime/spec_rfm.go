package runtime

import "github.com/getkin/kin-openapi/openapi3"

// rfmBandConfigSchema defines the response schema for one RFM band configuration.
func rfmBandConfigSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("id", openapi3.NewInt64Schema()).
		WithProperty("dimension", openapi3.NewStringSchema()).
		WithProperty("ascending", openapi3.NewBoolSchema()).
		WithProperty("band5Min", openapi3.NewFloat64Schema()).
		WithProperty("band4Min", openapi3.NewFloat64Schema()).
		WithProperty("band3Min", openapi3.NewFloat64Schema()).
		WithProperty("band2Min", openapi3.NewFloat64Schema()).
		WithProperty("updatedAt", openapi3.NewDateTimeSchema())
}

// rfmBandUpdateRequestSchema defines the request schema for updating one RFM band.
func rfmBandUpdateRequestSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("band5Min", openapi3.NewFloat64Schema()).
		WithProperty("band4Min", openapi3.NewFloat64Schema()).
		WithProperty("band3Min", openapi3.NewFloat64Schema()).
		WithProperty("band2Min", openapi3.NewFloat64Schema()).
		WithProperty("ascending", openapi3.NewBoolSchema())
}

// rfmGroupConditionsSchema defines the conditions sub-schema for RFM groups.
func rfmGroupConditionsSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("rMin", openapi3.NewInt32Schema()).
		WithProperty("rMax", openapi3.NewInt32Schema()).
		WithProperty("fMin", openapi3.NewInt32Schema()).
		WithProperty("fMax", openapi3.NewInt32Schema()).
		WithProperty("mMin", openapi3.NewInt32Schema()).
		WithProperty("mMax", openapi3.NewInt32Schema())
}

// rfmGroupSchema defines the response schema for an RFM group.
func rfmGroupSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("id", openapi3.NewStringSchema()).
		WithProperty("name", openapi3.NewStringSchema()).
		WithProperty("slug", openapi3.NewStringSchema()).
		WithProperty("description", openapi3.NewStringSchema()).
		WithProperty("conditions", rfmGroupConditionsSchema()).
		WithProperty("createdAt", openapi3.NewDateTimeSchema()).
		WithProperty("updatedAt", openapi3.NewDateTimeSchema())
}

// rfmGroupRequestSchema defines the request schema for creating or updating an RFM group.
func rfmGroupRequestSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("name", openapi3.NewStringSchema()).
		WithProperty("slug", openapi3.NewStringSchema()).
		WithProperty("description", openapi3.NewStringSchema()).
		WithProperty("conditions", rfmGroupConditionsSchema()).
		WithRequired([]string{"name", "slug"})
}

// rfmScoreSchema defines the response schema for an RFM contact score.
func rfmScoreSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("contactId", openapi3.NewStringSchema()).
		WithProperty("recencyDays", openapi3.NewInt32Schema()).
		WithProperty("frequency", openapi3.NewInt32Schema()).
		WithProperty("monetary", openapi3.NewFloat64Schema()).
		WithProperty("rScore", openapi3.NewInt32Schema()).
		WithProperty("fScore", openapi3.NewInt32Schema()).
		WithProperty("mScore", openapi3.NewInt32Schema()).
		WithProperty("rfmTotal", openapi3.NewInt32Schema())
}

// rfmScoreBatchRequestSchema defines the request schema for batch contact scoring.
func rfmScoreBatchRequestSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("contactIds", openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema())).
		WithRequired([]string{"contactIds"})
}

// rfmBandsPathItem returns the path item for GET/no-collection band endpoints.
func rfmBandsPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{Get: getBandsOperation()}
}

// rfmBandByDimensionPathItem returns the path item for PUT band-by-dimension endpoints.
func rfmBandByDimensionPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{Put: updateBandOperation()}
}

// rfmGroupsPathItem returns the path item for RFM group collection endpoints.
func rfmGroupsPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{
		Post: createGroupOperation(),
		Get:  listGroupsOperation(),
	}
}

// rfmGroupByIDPathItem returns the path item for RFM group ID-scoped endpoints.
func rfmGroupByIDPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{
		Get:    getGroupOperation(),
		Put:    updateGroupOperation(),
		Delete: deleteGroupOperation(),
	}
}

// rfmContactScorePathItem returns the path item for contact score endpoints.
func rfmContactScorePathItem() *openapi3.PathItem {
	return &openapi3.PathItem{Get: scoreContactOperation()}
}

// rfmScoreBatchPathItem returns the path item for batch score endpoints.
func rfmScoreBatchPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{Post: scoreBatchOperation()}
}

// rfmRefreshPathItem returns the path item for RFM MV refresh endpoints.
func rfmRefreshPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{Post: rfmRefreshOperation()}
}

// getBandsOperation defines the OpenAPI operation for GET /analytics/rfm/bands.
func getBandsOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "AnalyticsRFMController_getBands",
		Summary:     "List RFM band threshold configurations",
		Tags:        []string{analyticsTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("RFM band configurations.", "#/components/schemas/RFMBandConfig")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		),
	}
}

// updateBandOperation defines the OpenAPI operation for PUT /analytics/rfm/bands/:dimension.
func updateBandOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "AnalyticsRFMController_updateBand",
		Summary:     "Update RFM band thresholds for one dimension",
		Tags:        []string{analyticsTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			rfmPathParameter("dimension", "RFM dimension (recency, frequency, monetary)"),
		},
		RequestBody: jsonRequestBodyRef("#/components/schemas/RFMBandUpdateRequest"),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Updated band configuration.", "#/components/schemas/RFMBandConfig")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		),
	}
}

// createGroupOperation defines the OpenAPI operation for POST /analytics/rfm/groups.
func createGroupOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "AnalyticsRFMController_createGroup",
		Summary:     "Create a new RFM group",
		Tags:        []string{analyticsTag},
		Security:    bearerSecurityRequirements(),
		RequestBody: jsonRequestBodyRef("#/components/schemas/RFMGroupRequest"),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(201, jsonResponse("RFM group created.", "#/components/schemas/RFMGroup")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(409, responseWithDescription("Conflict - slug already exists.")),
		),
	}
}

// listGroupsOperation defines the OpenAPI operation for GET /analytics/rfm/groups.
func listGroupsOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "AnalyticsRFMController_listGroups",
		Summary:     "List all RFM groups",
		Tags:        []string{analyticsTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("RFM group list.", "#/components/schemas/RFMGroup")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		),
	}
}

// getGroupOperation defines the OpenAPI operation for GET /analytics/rfm/groups/:id.
func getGroupOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "AnalyticsRFMController_getGroup",
		Summary:     "Get one RFM group by ID",
		Tags:        []string{analyticsTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			rfmPathParameter("id", "RFM group ID"),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("RFM group.", "#/components/schemas/RFMGroup")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("RFM group not found.")),
		),
	}
}

// updateGroupOperation defines the OpenAPI operation for PUT /analytics/rfm/groups/:id.
func updateGroupOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "AnalyticsRFMController_updateGroup",
		Summary:     "Update one RFM group",
		Tags:        []string{analyticsTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			rfmPathParameter("id", "RFM group ID"),
		},
		RequestBody: jsonRequestBodyRef("#/components/schemas/RFMGroupRequest"),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Updated RFM group.", "#/components/schemas/RFMGroup")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("RFM group not found.")),
		),
	}
}

// deleteGroupOperation defines the OpenAPI operation for DELETE /analytics/rfm/groups/:id.
func deleteGroupOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "AnalyticsRFMController_deleteGroup",
		Summary:     "Delete one RFM group",
		Tags:        []string{analyticsTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			rfmPathParameter("id", "RFM group ID"),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(204, responseWithDescription("RFM group deleted.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("RFM group not found.")),
		),
	}
}

// scoreContactOperation defines the OpenAPI operation for GET /analytics/rfm/contacts/:contactId/score.
func scoreContactOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "AnalyticsRFMController_scoreContact",
		Summary:     "Get RFM score for one contact",
		Tags:        []string{analyticsTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			rfmPathParameter("contactId", "Contact ID"),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("RFM score for the contact.", "#/components/schemas/RFMScore")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		),
	}
}

// scoreBatchOperation defines the OpenAPI operation for POST /analytics/rfm/contacts/score-batch.
func scoreBatchOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "AnalyticsRFMController_scoreBatch",
		Summary:     "Batch RFM score for up to 1000 contacts",
		Tags:        []string{analyticsTag},
		Security:    bearerSecurityRequirements(),
		RequestBody: jsonRequestBodyRef("#/components/schemas/RFMScoreBatchRequest"),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("RFM scores.", "#/components/schemas/RFMScore")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		),
	}
}

// rfmRefreshOperation defines the OpenAPI operation for POST /analytics/rfm/refresh.
func rfmRefreshOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "AnalyticsRFMController_refresh",
		Summary:     "Refresh the RFM materialized view",
		Tags:        []string{analyticsTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Refresh completed.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		),
	}
}

// rfmPathParameter builds a required path parameter for RFM route definitions.
func rfmPathParameter(name, description string) *openapi3.ParameterRef {
	return &openapi3.ParameterRef{Value: openapi3.NewPathParameter(name).WithDescription(description).WithSchema(openapi3.NewStringSchema())}
}

// jsonRequestBodyRef builds a required JSON request body referencing a component schema.
func jsonRequestBodyRef(schemaRef string) *openapi3.RequestBodyRef {
	return &openapi3.RequestBodyRef{
		Value: openapi3.NewRequestBody().
			WithRequired(true).
			WithContent(openapi3.Content{
				"application/json": &openapi3.MediaType{Schema: &openapi3.SchemaRef{Ref: schemaRef}},
			}),
	}
}
