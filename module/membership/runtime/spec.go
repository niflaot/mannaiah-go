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

	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:   "Membership API",
			Version: "2.0.0",
		},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/membership/optin", &openapi3.PathItem{Post: optInOperation()}),
			openapi3.WithPath("/membership/optout", &openapi3.PathItem{Post: optOutOperation()}),
			openapi3.WithPath("/membership/stamp", &openapi3.PathItem{Post: stampOperation()}),
			openapi3.WithPath("/membership/status/{contactId}", &openapi3.PathItem{Get: statusOperation()}),
			openapi3.WithPath("/membership/status/{contactId}/{channel}", &openapi3.PathItem{Get: statusByChannelOperation()}),
			openapi3.WithPath("/membership/status/{contactId}/stamps", &openapi3.PathItem{Get: stampsOperation()}),
			openapi3.WithPath("/membership/stamps/{contactId}/{channel}", &openapi3.PathItem{Get: stampsByChannelOperation()}),
			openapi3.WithPath("/membership/migrate", &openapi3.PathItem{Post: migrateOperation()}),
		),
		Components: &components,
		Tags: openapi3.Tags{
			&openapi3.Tag{Name: membershipTag},
		},
	}
}

// optInOperation defines the OpenAPI operation for opt-in requests.
func optInOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "MembershipController_optIn",
		Summary:     "Opt-in membership consent",
		Tags:        []string{membershipTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Membership status updated.")),
		),
	}
}

// optOutOperation defines the OpenAPI operation for opt-out requests.
func optOutOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "MembershipController_optOut",
		Summary:     "Opt-out membership consent",
		Tags:        []string{membershipTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Membership status updated.")),
		),
	}
}

// stampOperation defines the OpenAPI operation for stamp requests.
func stampOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "MembershipController_stamp",
		Summary:     "Stamp membership consent",
		Tags:        []string{membershipTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Membership status updated.")),
		),
	}
}

// statusOperation defines the OpenAPI operation for status requests.
func statusOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "MembershipController_status",
		Summary:     "Get current membership status",
		Tags:        []string{membershipTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Membership status.")),
		),
	}
}

// statusByChannelOperation defines the OpenAPI operation for status-by-channel requests.
func statusByChannelOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "MembershipController_statusByChannel",
		Summary:     "Get current membership status by channel",
		Tags:        []string{membershipTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Membership status.")),
		),
	}
}

// stampsOperation defines the OpenAPI operation for stamps requests.
func stampsOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "MembershipController_stamps",
		Summary:     "List membership stamps",
		Tags:        []string{membershipTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Membership stamps.")),
		),
	}
}

// stampsByChannelOperation defines the OpenAPI operation for stamps-by-channel requests.
func stampsByChannelOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "MembershipController_stampsByChannel",
		Summary:     "List membership stamps by channel",
		Tags:        []string{membershipTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Membership stamps.")),
		),
	}
}

// migrateOperation defines the OpenAPI operation for migration requests.
func migrateOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "MembershipController_migrate",
		Summary:     "Migrate legacy contact metadata to membership stamps",
		Tags:        []string{membershipTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Membership migration summary.")),
		),
	}
}

// bearerSecurityRequirements builds bearer-auth operation security requirements.
func bearerSecurityRequirements() *openapi3.SecurityRequirements {
	return openapi3.NewSecurityRequirements().With(openapi3.NewSecurityRequirement().Authenticate(bearerSecurityScheme))
}

// responseWithDescription builds an OpenAPI response from a plain description.
func responseWithDescription(description string) *openapi3.ResponseRef {
	return &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription(description)}
}
