package http

import "github.com/getkin/kin-openapi/openapi3"

// Paths returns the OpenAPI path definitions for coupon endpoints.
func Paths() *openapi3.Paths {
	return openapi3.NewPaths(
		openapi3.WithPath("/coupons", &openapi3.PathItem{
			Post: &openapi3.Operation{
				Tags:        []string{"Coupons"},
				Summary:     "Create coupon",
				Description: "Creates a new coupon. Generates a random code when none is provided.",
				OperationID: "createCoupon",
				Security:    openapi3.NewSecurityRequirements().With(openapi3.NewSecurityRequirement().Authenticate("coupons_bearer")),
				RequestBody: &openapi3.RequestBodyRef{
					Value: openapi3.NewRequestBody().
						WithDescription("Coupon creation payload").
						WithRequired(true).
						WithJSONSchemaRef(openapi3.NewSchemaRef("#/components/schemas/CreateCouponRequest", nil)),
				},
				Responses: openapi3.NewResponses(
					openapi3.WithStatus(201, &openapi3.ResponseRef{
						Value: openapi3.NewResponse().WithDescription("Coupon created").
							WithJSONSchemaRef(openapi3.NewSchemaRef("#/components/schemas/Coupon", nil)),
					}),
					openapi3.WithStatus(400, errorResponse("Validation error")),
					openapi3.WithStatus(409, errorResponse("Code conflict")),
					openapi3.WithStatus(401, errorResponse("Unauthorized")),
				),
			},
			Get: &openapi3.Operation{
				Tags:        []string{"Coupons"},
				Summary:     "List coupons",
				Description: "Returns a paginated list of coupons, optionally filtered by origin, active state, or code.",
				OperationID: "listCoupons",
				Security:    openapi3.NewSecurityRequirements().With(openapi3.NewSecurityRequirement().Authenticate("coupons_bearer")),
				Parameters: openapi3.Parameters{
					{Value: openapi3.NewQueryParameter("origin").WithSchema(openapi3.NewStringSchema())},
					{Value: openapi3.NewQueryParameter("active").WithSchema(openapi3.NewStringSchema())},
					{Value: openapi3.NewQueryParameter("code").WithSchema(openapi3.NewStringSchema())},
					{Value: openapi3.NewQueryParameter("limit").WithSchema(openapi3.NewIntegerSchema())},
					{Value: openapi3.NewQueryParameter("offset").WithSchema(openapi3.NewIntegerSchema())},
				},
				Responses: openapi3.NewResponses(
					openapi3.WithStatus(200, &openapi3.ResponseRef{
						Value: openapi3.NewResponse().WithDescription("Coupon list").
							WithJSONSchemaRef(openapi3.NewSchemaRef("#/components/schemas/CouponListResponse", nil)),
					}),
					openapi3.WithStatus(401, errorResponse("Unauthorized")),
				),
			},
		}),
		openapi3.WithPath("/coupons/code/{code}", &openapi3.PathItem{
			Get: &openapi3.Operation{
				Tags:        []string{"Coupons"},
				Summary:     "Get coupon by code",
				Description: "Retrieves a coupon by its unique code.",
				OperationID: "getCouponByCode",
				Security:    openapi3.NewSecurityRequirements().With(openapi3.NewSecurityRequirement().Authenticate("coupons_bearer")),
				Parameters: openapi3.Parameters{
					{Value: openapi3.NewPathParameter("code").WithSchema(openapi3.NewStringSchema())},
				},
				Responses: openapi3.NewResponses(
					openapi3.WithStatus(200, &openapi3.ResponseRef{
						Value: openapi3.NewResponse().WithDescription("Coupon").
							WithJSONSchemaRef(openapi3.NewSchemaRef("#/components/schemas/Coupon", nil)),
					}),
					openapi3.WithStatus(404, errorResponse("Not found")),
					openapi3.WithStatus(401, errorResponse("Unauthorized")),
				),
			},
		}),
		openapi3.WithPath("/coupons/{id}", &openapi3.PathItem{
			Get: &openapi3.Operation{
				Tags:        []string{"Coupons"},
				Summary:     "Get coupon by ID",
				Description: "Retrieves a coupon by its unique identifier.",
				OperationID: "getCouponByID",
				Security:    openapi3.NewSecurityRequirements().With(openapi3.NewSecurityRequirement().Authenticate("coupons_bearer")),
				Parameters: openapi3.Parameters{
					{Value: openapi3.NewPathParameter("id").WithSchema(openapi3.NewStringSchema())},
				},
				Responses: openapi3.NewResponses(
					openapi3.WithStatus(200, &openapi3.ResponseRef{
						Value: openapi3.NewResponse().WithDescription("Coupon").
							WithJSONSchemaRef(openapi3.NewSchemaRef("#/components/schemas/Coupon", nil)),
					}),
					openapi3.WithStatus(404, errorResponse("Not found")),
					openapi3.WithStatus(401, errorResponse("Unauthorized")),
				),
			},
			Put: &openapi3.Operation{
				Tags:        []string{"Coupons"},
				Summary:     "Update coupon",
				Description: "Replaces all mutable coupon fields. Assignment and scope lists are replaced wholesale.",
				OperationID: "updateCoupon",
				Security:    openapi3.NewSecurityRequirements().With(openapi3.NewSecurityRequirement().Authenticate("coupons_bearer")),
				Parameters: openapi3.Parameters{
					{Value: openapi3.NewPathParameter("id").WithSchema(openapi3.NewStringSchema())},
				},
				RequestBody: &openapi3.RequestBodyRef{
					Value: openapi3.NewRequestBody().
						WithRequired(true).
						WithJSONSchemaRef(openapi3.NewSchemaRef("#/components/schemas/UpdateCouponRequest", nil)),
				},
				Responses: openapi3.NewResponses(
					openapi3.WithStatus(200, &openapi3.ResponseRef{
						Value: openapi3.NewResponse().WithDescription("Updated coupon").
							WithJSONSchemaRef(openapi3.NewSchemaRef("#/components/schemas/Coupon", nil)),
					}),
					openapi3.WithStatus(400, errorResponse("Validation error")),
					openapi3.WithStatus(404, errorResponse("Not found")),
					openapi3.WithStatus(401, errorResponse("Unauthorized")),
				),
			},
			Delete: &openapi3.Operation{
				Tags:        []string{"Coupons"},
				Summary:     "Delete coupon",
				Description: "Soft-deletes a coupon by its identifier.",
				OperationID: "deleteCoupon",
				Security:    openapi3.NewSecurityRequirements().With(openapi3.NewSecurityRequirement().Authenticate("coupons_bearer")),
				Parameters: openapi3.Parameters{
					{Value: openapi3.NewPathParameter("id").WithSchema(openapi3.NewStringSchema())},
				},
				Responses: openapi3.NewResponses(
					openapi3.WithStatus(204, &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription("Deleted")}),
					openapi3.WithStatus(404, errorResponse("Not found")),
					openapi3.WithStatus(401, errorResponse("Unauthorized")),
				),
			},
		}),
		openapi3.WithPath("/coupons/{id}/usage", &openapi3.PathItem{
			Post: &openapi3.Operation{
				Tags:        []string{"Coupons"},
				Summary:     "Record coupon usage",
				Description: "Validates and records a coupon redemption event for an order. Enforces global and per-email usage limits, expiry, and active state.",
				OperationID: "recordCouponUsage",
				Security:    openapi3.NewSecurityRequirements().With(openapi3.NewSecurityRequirement().Authenticate("coupons_bearer")),
				Parameters: openapi3.Parameters{
					{Value: openapi3.NewPathParameter("id").WithSchema(openapi3.NewStringSchema())},
				},
				RequestBody: &openapi3.RequestBodyRef{
					Value: openapi3.NewRequestBody().
						WithRequired(true).
						WithJSONSchemaRef(openapi3.NewSchemaRef("#/components/schemas/RecordCouponUsageRequest", nil)),
				},
				Responses: openapi3.NewResponses(
					openapi3.WithStatus(204, &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription("Usage recorded")}),
					openapi3.WithStatus(404, errorResponse("Coupon not found")),
					openapi3.WithStatus(409, errorResponse("Already applied to this order")),
					openapi3.WithStatus(422, errorResponse("Usage limit / expired / inactive")),
					openapi3.WithStatus(401, errorResponse("Unauthorized")),
				),
			},
		}),
	)
}

// SecuritySchemes returns the OpenAPI security scheme definitions for coupon endpoints.
func SecuritySchemes() openapi3.SecuritySchemes {
	return openapi3.SecuritySchemes{
		"coupons_bearer": &openapi3.SecuritySchemeRef{Value: openapi3.NewJWTSecurityScheme()},
	}
}

// Tags returns the OpenAPI tag definitions for coupon endpoints.
func Tags() openapi3.Tags {
	return openapi3.Tags{
		{Name: "Coupons", Description: "Coupon management and usage tracking"},
	}
}

// Schemas returns the OpenAPI component schemas for coupon types.
func Schemas() openapi3.Schemas {
	stringArr := openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema())
	optInt := openapi3.NewIntegerSchema()
	optInt.Nullable = true

	coupon := openapi3.NewObjectSchema().WithProperties(map[string]*openapi3.Schema{
		"id":                  openapi3.NewStringSchema(),
		"code":                openapi3.NewStringSchema(),
		"origin":              openapi3.NewStringSchema(),
		"discountType":        openapi3.NewStringSchema().WithEnum("fixed", "percentage"),
		"discountAmount":      openapi3.NewFloat64Schema(),
		"maxUsagesGlobal":     optInt,
		"maxUsagesPerEmail":   optInt,
		"active":              openapi3.NewBoolSchema(),
		"expiresAt":           openapi3.NewStringSchema().WithFormat("date-time"),
		"assignedEmails":      stringArr,
		"assignedContactIds":  stringArr,
		"includedProductIds":  stringArr,
		"includedCategoryIds": stringArr,
		"includedTagIds":      stringArr,
		"wooCommerceId":       optInt,
		"createdAt":           openapi3.NewStringSchema().WithFormat("date-time"),
		"updatedAt":           openapi3.NewStringSchema().WithFormat("date-time"),
	})

	createReq := openapi3.NewObjectSchema().
		WithRequired([]string{"discountType", "discountAmount"}).
		WithProperties(map[string]*openapi3.Schema{
			"code":                openapi3.NewStringSchema(),
			"origin":              openapi3.NewStringSchema(),
			"discountType":        openapi3.NewStringSchema().WithEnum("fixed", "percentage"),
			"discountAmount":      openapi3.NewFloat64Schema(),
			"maxUsagesGlobal":     optInt,
			"maxUsagesPerEmail":   optInt,
			"active":              openapi3.NewBoolSchema(),
			"expiresAt":           openapi3.NewStringSchema().WithFormat("date-time"),
			"assignedEmails":      stringArr,
			"assignedContactIds":  stringArr,
			"includedProductIds":  stringArr,
			"includedCategoryIds": stringArr,
			"includedTagIds":      stringArr,
		})

	updateReq := openapi3.NewObjectSchema().
		WithRequired([]string{"discountType", "discountAmount"}).
		WithProperties(map[string]*openapi3.Schema{
			"origin":              openapi3.NewStringSchema(),
			"discountType":        openapi3.NewStringSchema().WithEnum("fixed", "percentage"),
			"discountAmount":      openapi3.NewFloat64Schema(),
			"maxUsagesGlobal":     optInt,
			"maxUsagesPerEmail":   optInt,
			"active":              openapi3.NewBoolSchema(),
			"expiresAt":           openapi3.NewStringSchema().WithFormat("date-time"),
			"assignedEmails":      stringArr,
			"assignedContactIds":  stringArr,
			"includedProductIds":  stringArr,
			"includedCategoryIds": stringArr,
			"includedTagIds":      stringArr,
		})

	usageReq := openapi3.NewObjectSchema().
		WithRequired([]string{"orderId"}).
		WithProperties(map[string]*openapi3.Schema{
			"orderId": openapi3.NewStringSchema(),
			"email":   openapi3.NewStringSchema(),
		})

	listResp := openapi3.NewObjectSchema().WithProperties(map[string]*openapi3.Schema{
		"items": openapi3.NewArraySchema().WithItems(coupon),
		"total": openapi3.NewInt64Schema(),
	})

	return openapi3.Schemas{
		"Coupon":                  openapi3.NewSchemaRef("", coupon),
		"CreateCouponRequest":     openapi3.NewSchemaRef("", createReq),
		"UpdateCouponRequest":     openapi3.NewSchemaRef("", updateReq),
		"RecordCouponUsageRequest": openapi3.NewSchemaRef("", usageReq),
		"CouponListResponse":      openapi3.NewSchemaRef("", listResp),
	}
}

// errorResponse creates a minimal error response schema ref.
func errorResponse(description string) *openapi3.ResponseRef {
	return &openapi3.ResponseRef{
		Value: openapi3.NewResponse().WithDescription(description).
			WithJSONSchemaRef(openapi3.NewSchemaRef("", openapi3.NewObjectSchema().WithProperties(map[string]*openapi3.Schema{
				"error": openapi3.NewStringSchema(),
			}))),
	}
}
