package runtime

import "github.com/getkin/kin-openapi/openapi3"

const (
	// membershipTag defines membership OpenAPI tag values.
	membershipTag = "membership"
	// bearerSecurityScheme defines security-scheme key values.
	bearerSecurityScheme = "membership_bearer"
)

// OpenAPISpec returns membership-module OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	components := openapi3.NewComponents()
	components.SecuritySchemes = openapi3.SecuritySchemes{
		bearerSecurityScheme: &openapi3.SecuritySchemeRef{Value: openapi3.NewJWTSecurityScheme()},
	}
	components.Schemas = openapi3.Schemas{
		"MembershipActionRequest": {Value: membershipActionRequestSchema()},
		"MembershipStampRequest":  {Value: membershipStampRequestSchema()},
		"MembershipStatus":        {Value: membershipStatusSchema()},
		"MembershipStatusList":    {Value: membershipStatusListSchema()},
		"MembershipStamp":         {Value: membershipStampSchema()},
		"MembershipStamps":        {Value: membershipStampsSchema()},
	}

	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:   "Membership API",
			Version: "2.0.5",
		},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/membership/optin", &openapi3.PathItem{Post: optInOperation()}),
			openapi3.WithPath("/membership/optout", &openapi3.PathItem{Post: optOutOperation()}),
			openapi3.WithPath("/membership/stamp", &openapi3.PathItem{Post: stampOperation()}),
			openapi3.WithPath("/membership/status/{contactId}", &openapi3.PathItem{Get: statusOperation()}),
			openapi3.WithPath("/membership/status/{contactId}/{channel}", &openapi3.PathItem{Get: statusByChannelOperation()}),
			openapi3.WithPath("/membership/status/{contactId}/stamps", &openapi3.PathItem{Get: stampsOperation()}),
			openapi3.WithPath("/membership/stamps/{contactId}/{channel}", &openapi3.PathItem{Get: stampsByChannelOperation()}),
		),
		Components: &components,
		Tags: openapi3.Tags{
			&openapi3.Tag{Name: membershipTag},
		},
	}
}

// optInOperation defines the OpenAPI operation for opt-in requests.
func optInOperation() *openapi3.Operation {
	operation := baseOperation("MembershipController_optIn", "Opt-in membership consent")
	operation.RequestBody = &openapi3.RequestBodyRef{Value: openapi3.NewRequestBody().WithRequired(false).WithContent(openapi3.Content{
		"application/json": &openapi3.MediaType{Schema: &openapi3.SchemaRef{Ref: "#/components/schemas/MembershipActionRequest"}},
	})}
	operation.Responses = openapi3.NewResponses(
		openapi3.WithStatus(200, jsonResponse("Membership status updated.", "#/components/schemas/MembershipStatus")),
		openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
		openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
		openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		openapi3.WithStatus(404, responseWithDescription("Membership contact not found.")),
	)

	return operation
}

// optOutOperation defines the OpenAPI operation for opt-out requests.
func optOutOperation() *openapi3.Operation {
	operation := baseOperation("MembershipController_optOut", "Opt-out membership consent")
	operation.RequestBody = &openapi3.RequestBodyRef{Value: openapi3.NewRequestBody().WithRequired(false).WithContent(openapi3.Content{
		"application/json": &openapi3.MediaType{Schema: &openapi3.SchemaRef{Ref: "#/components/schemas/MembershipActionRequest"}},
	})}
	operation.Responses = openapi3.NewResponses(
		openapi3.WithStatus(200, jsonResponse("Membership status updated.", "#/components/schemas/MembershipStatus")),
		openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
		openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
		openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		openapi3.WithStatus(404, responseWithDescription("Membership contact not found.")),
	)

	return operation
}

// stampOperation defines the OpenAPI operation for stamp requests.
func stampOperation() *openapi3.Operation {
	operation := baseOperation("MembershipController_stamp", "Stamp membership consent")
	operation.RequestBody = &openapi3.RequestBodyRef{Value: openapi3.NewRequestBody().WithRequired(true).WithContent(openapi3.Content{
		"application/json": &openapi3.MediaType{Schema: &openapi3.SchemaRef{Ref: "#/components/schemas/MembershipStampRequest"}},
	})}
	operation.Responses = openapi3.NewResponses(
		openapi3.WithStatus(200, jsonResponse("Membership status updated.", "#/components/schemas/MembershipStatus")),
		openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
		openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
		openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		openapi3.WithStatus(404, responseWithDescription("Membership contact not found.")),
	)

	return operation
}

// statusOperation defines the OpenAPI operation for status requests.
func statusOperation() *openapi3.Operation {
	operation := baseOperation("MembershipController_status", "Get current membership statuses")
	operation.Parameters = openapi3.Parameters{
		pathParam("contactId", "Contact ID."),
		queryParam("channel", "Optional channel override. When provided, returns one effective status."),
	}
	operation.Responses = openapi3.NewResponses(
		openapi3.WithStatus(200, jsonResponse("Membership status list.", "#/components/schemas/MembershipStatusList")),
		openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
		openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
		openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		openapi3.WithStatus(404, responseWithDescription("Membership status not found.")),
	)

	return operation
}

// statusByChannelOperation defines the OpenAPI operation for status-by-channel requests.
func statusByChannelOperation() *openapi3.Operation {
	operation := baseOperation("MembershipController_statusByChannel", "Get current membership status by channel")
	operation.Parameters = openapi3.Parameters{
		pathParam("contactId", "Contact ID."),
		pathParam("channel", "Membership channel."),
	}
	operation.Responses = openapi3.NewResponses(
		openapi3.WithStatus(200, jsonResponse("Membership status.", "#/components/schemas/MembershipStatus")),
		openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
		openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
		openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		openapi3.WithStatus(404, responseWithDescription("Membership status not found.")),
	)

	return operation
}

// stampsOperation defines the OpenAPI operation for stamps requests.
func stampsOperation() *openapi3.Operation {
	operation := baseOperation("MembershipController_stamps", "List membership stamps")
	operation.Parameters = openapi3.Parameters{
		pathParam("contactId", "Contact ID."),
		queryParam("channel", "Membership channel (defaults to email)."),
	}
	operation.Responses = openapi3.NewResponses(
		openapi3.WithStatus(200, jsonResponse("Membership stamps.", "#/components/schemas/MembershipStamps")),
		openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
		openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
		openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		openapi3.WithStatus(404, responseWithDescription("Membership status not found.")),
	)

	return operation
}

// stampsByChannelOperation defines the OpenAPI operation for stamps-by-channel requests.
func stampsByChannelOperation() *openapi3.Operation {
	operation := baseOperation("MembershipController_stampsByChannel", "List membership stamps by channel")
	operation.Parameters = openapi3.Parameters{
		pathParam("contactId", "Contact ID."),
		pathParam("channel", "Membership channel."),
	}
	operation.Responses = openapi3.NewResponses(
		openapi3.WithStatus(200, jsonResponse("Membership stamps.", "#/components/schemas/MembershipStamps")),
		openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
		openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
		openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		openapi3.WithStatus(404, responseWithDescription("Membership status not found.")),
	)

	return operation
}

// baseOperation builds one membership operation with common security and tags.
func baseOperation(operationID string, summary string) *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: operationID,
		Summary:     summary,
		Tags:        []string{membershipTag},
		Security:    bearerSecurityRequirements(),
	}
}

// membershipActionRequestSchema defines opt-in and opt-out request payload schema values.
func membershipActionRequestSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("contactId", openapi3.NewStringSchema()).
		WithProperty("email", openapi3.NewStringSchema()).
		WithProperty("channel", openapi3.NewStringSchema().WithDefault("all")).
		WithProperty("source", openapi3.NewStringSchema().WithDefault("api")).
		WithProperty("occurredAt", openapi3.NewDateTimeSchema())
}

// membershipStampRequestSchema defines generic stamp request payload schema values.
func membershipStampRequestSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("contactId", openapi3.NewStringSchema()).
		WithProperty("email", openapi3.NewStringSchema()).
		WithProperty("channel", openapi3.NewStringSchema()).
		WithProperty("action", openapi3.NewStringSchema().WithEnum("opt_in", "opt_out")).
		WithProperty("source", openapi3.NewStringSchema().WithDefault("api")).
		WithProperty("occurredAt", openapi3.NewDateTimeSchema()).
		WithRequired([]string{"channel", "action"})
}

// membershipStatusSchema defines membership status response schema values.
func membershipStatusSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("contactId", openapi3.NewStringSchema()).
		WithProperty("channel", openapi3.NewStringSchema()).
		WithProperty("action", openapi3.NewStringSchema().WithEnum("opt_in", "opt_out")).
		WithProperty("source", openapi3.NewStringSchema()).
		WithProperty("occurredAt", openapi3.NewDateTimeSchema()).
		WithProperty("updatedAt", openapi3.NewDateTimeSchema())
}

// membershipStatusListSchema defines membership status-list response schema values.
func membershipStatusListSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("contactId", openapi3.NewStringSchema()).
		WithProperty("statuses", openapi3.NewArraySchema().WithItems(membershipStatusSchema()))
}

// membershipStampSchema defines immutable membership stamp response schema values.
func membershipStampSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("id", openapi3.NewStringSchema()).
		WithProperty("contactId", openapi3.NewStringSchema()).
		WithProperty("channel", openapi3.NewStringSchema()).
		WithProperty("action", openapi3.NewStringSchema().WithEnum("opt_in", "opt_out")).
		WithProperty("source", openapi3.NewStringSchema()).
		WithProperty("occurredAt", openapi3.NewDateTimeSchema()).
		WithProperty("createdAt", openapi3.NewDateTimeSchema())
}

// membershipStampsSchema defines immutable membership stamp-list response schema values.
func membershipStampsSchema() *openapi3.Schema {
	return openapi3.NewArraySchema().WithItems(membershipStampSchema())
}

// pathParam builds one required string path parameter.
func pathParam(name string, description string) *openapi3.ParameterRef {
	return &openapi3.ParameterRef{Value: openapi3.NewPathParameter(name).WithDescription(description)}
}

// queryParam builds one optional string query parameter.
func queryParam(name string, description string) *openapi3.ParameterRef {
	param := openapi3.NewQueryParameter(name)
	param.Required = false
	param.Description = description
	param.Schema = &openapi3.SchemaRef{Value: openapi3.NewStringSchema()}
	return &openapi3.ParameterRef{Value: param}
}

// bearerSecurityRequirements builds bearer-auth operation security requirements.
func bearerSecurityRequirements() *openapi3.SecurityRequirements {
	return openapi3.NewSecurityRequirements().With(openapi3.NewSecurityRequirement().Authenticate(bearerSecurityScheme))
}

// jsonResponse builds a JSON response with a schema reference.
func jsonResponse(description string, schemaRef string) *openapi3.ResponseRef {
	return &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription(description).WithContent(openapi3.Content{
		"application/json": &openapi3.MediaType{Schema: &openapi3.SchemaRef{Ref: schemaRef}},
	})}
}

// responseWithDescription builds an OpenAPI response from a plain description.
func responseWithDescription(description string) *openapi3.ResponseRef {
	return &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription(description)}
}
