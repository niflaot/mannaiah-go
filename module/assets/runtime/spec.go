package runtime

import "github.com/getkin/kin-openapi/openapi3"

const (
	// assetsTag defines the OpenAPI tag used by asset endpoints.
	assetsTag = "assets"
	// bearerSecurityScheme defines the OpenAPI security scheme key used for bearer auth.
	bearerSecurityScheme = "assets_bearer"
)

// OpenAPISpec returns assets-module OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	components := openapi3.NewComponents()
	components.Schemas = openapi3.Schemas{
		"Asset":                  &openapi3.SchemaRef{Value: assetSchema()},
		"AssetTag":               &openapi3.SchemaRef{Value: tagSchema()},
		"AssetFolder":            &openapi3.SchemaRef{Value: folderSchema()},
		"UpdateAssetDto":         &openapi3.SchemaRef{Value: updateAssetSchema()},
		"CreateAssetFolderDto":   &openapi3.SchemaRef{Value: createFolderSchema()},
		"UpdateAssetFolderDto":   &openapi3.SchemaRef{Value: updateFolderSchema()},
		"PaginatedAssetResponse": &openapi3.SchemaRef{Value: paginatedAssetResponseSchema()},
		"PaginatedFolderResponse": &openapi3.SchemaRef{
			Value: paginatedFolderResponseSchema(),
		},
		"AssetPaginationMeta": &openapi3.SchemaRef{Value: assetPaginationMetaSchema()},
	}
	components.SecuritySchemes = openapi3.SecuritySchemes{
		bearerSecurityScheme: &openapi3.SecuritySchemeRef{Value: openapi3.NewJWTSecurityScheme()},
	}

	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:   "Assets API",
			Version: "1.0.0",
		},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/assets", assetsPathItem()),
			openapi3.WithPath("/assets/{id}", assetByIDPathItem()),
			openapi3.WithPath("/assets/folders", foldersPathItem()),
			openapi3.WithPath("/assets/folders/{id}", folderByIDPathItem()),
		),
		Components: &components,
		Tags: openapi3.Tags{
			&openapi3.Tag{Name: assetsTag},
		},
	}
}

// assetsPathItem returns OpenAPI path operations for collection endpoints.
func assetsPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{
		Post: createAssetOperation(),
		Get:  listAssetsOperation(),
	}
}

// assetByIDPathItem returns OpenAPI path operations for ID-scoped endpoints.
func assetByIDPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{
		Get:    getAssetOperation(),
		Patch:  updateAssetOperation(),
		Delete: deleteAssetOperation(),
	}
}

// foldersPathItem returns OpenAPI path operations for folder collection endpoints.
func foldersPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{
		Post: createFolderOperation(),
		Get:  listFoldersOperation(),
	}
}

// folderByIDPathItem returns OpenAPI path operations for folder ID-scoped endpoints.
func folderByIDPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{
		Get:    getFolderOperation(),
		Patch:  updateFolderOperation(),
		Delete: deleteFolderOperation(),
	}
}

// createAssetOperation defines the OpenAPI operation for asset uploads.
func createAssetOperation() *openapi3.Operation {
	formSchema := openapi3.NewObjectSchema().
		WithProperty("file", openapi3.NewStringSchema().WithFormat("binary")).
		WithProperty("name", openapi3.NewStringSchema()).
		WithProperty("folderId", openapi3.NewStringSchema()).
		WithProperty("tags", openapi3.NewStringSchema()).
		WithProperty("metadata", openapi3.NewStringSchema())
	formSchema.Required = []string{"file"}

	requestBody := openapi3.NewRequestBody().
		WithRequired(true).
		WithContent(openapi3.Content{
			"multipart/form-data": &openapi3.MediaType{Schema: &openapi3.SchemaRef{Value: formSchema}},
		})

	return &openapi3.Operation{
		OperationID: "AssetsController_uploadFile",
		Summary:     "Upload a file asset",
		Tags:        []string{assetsTag},
		Security:    bearerSecurityRequirements(),
		RequestBody: &openapi3.RequestBodyRef{Value: requestBody},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(201, responseWithDescription("The asset has been successfully uploaded and created.")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request - File validation failed.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("Folder not found.")),
			openapi3.WithStatus(503, responseWithDescription("Storage integration unavailable.")),
		),
	}
}

// listAssetsOperation defines the OpenAPI operation for asset listing.
func listAssetsOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "AssetsController_findAll",
		Summary:     "Get all assets",
		Tags:        []string{assetsTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			queryParameter("page", false, "Page number", openapi3.NewIntegerSchema()),
			queryParameter("limit", false, "Items per page", openapi3.NewIntegerSchema()),
			queryParameter("filters", false, "Filter criteria", openapi3.NewStringSchema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Return paginated assets.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		),
	}
}

// getAssetOperation defines the OpenAPI operation for asset retrieval by ID.
func getAssetOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "AssetsController_findOne",
		Summary:     "Get an asset by id",
		Tags:        []string{assetsTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathParameter("id", "Asset ID", openapi3.NewStringSchema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Return the asset.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(404, responseWithDescription("Asset not found.")),
		),
	}
}

// updateAssetOperation defines the OpenAPI operation for asset updates.
func updateAssetOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "AssetsController_update",
		Summary:     "Update an asset",
		Tags:        []string{assetsTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathParameter("id", "Asset ID", openapi3.NewStringSchema()),
		},
		RequestBody: jsonRequestBodyRef("#/components/schemas/UpdateAssetDto"),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("The asset has been successfully updated.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("Asset or folder not found.")),
		),
	}
}

// deleteAssetOperation defines the OpenAPI operation for asset deletion.
func deleteAssetOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "AssetsController_remove",
		Summary:     "Delete an asset",
		Tags:        []string{assetsTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathParameter("id", "Asset ID", openapi3.NewStringSchema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("The asset has been successfully deleted.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("Asset not found.")),
		),
	}
}

// createFolderOperation defines the OpenAPI operation for folder creation.
func createFolderOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "AssetsFoldersController_create",
		Summary:     "Create an asset folder",
		Tags:        []string{assetsTag},
		Security:    bearerSecurityRequirements(),
		RequestBody: jsonRequestBodyRef("#/components/schemas/CreateAssetFolderDto"),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(201, responseWithDescription("The folder has been successfully created.")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		),
	}
}

// listFoldersOperation defines the OpenAPI operation for folder listing.
func listFoldersOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "AssetsFoldersController_findAll",
		Summary:     "Get asset folders",
		Tags:        []string{assetsTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			queryParameter("page", false, "Page number", openapi3.NewIntegerSchema()),
			queryParameter("limit", false, "Items per page", openapi3.NewIntegerSchema()),
			queryParameter("filters", false, "Filter criteria", openapi3.NewStringSchema()),
			queryParameter("parentFolderId", false, "Parent folder ID", openapi3.NewStringSchema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Return paginated folders.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		),
	}
}

// getFolderOperation defines the OpenAPI operation for folder retrieval by ID.
func getFolderOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "AssetsFoldersController_findOne",
		Summary:     "Get an asset folder by id",
		Tags:        []string{assetsTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathParameter("id", "Folder ID", openapi3.NewStringSchema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Return the folder.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(404, responseWithDescription("Folder not found.")),
		),
	}
}

// updateFolderOperation defines the OpenAPI operation for folder updates.
func updateFolderOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "AssetsFoldersController_update",
		Summary:     "Update an asset folder",
		Tags:        []string{assetsTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathParameter("id", "Folder ID", openapi3.NewStringSchema()),
		},
		RequestBody: jsonRequestBodyRef("#/components/schemas/UpdateAssetFolderDto"),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("The folder has been successfully updated.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("Folder not found.")),
		),
	}
}

// deleteFolderOperation defines the OpenAPI operation for folder deletion.
func deleteFolderOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "AssetsFoldersController_remove",
		Summary:     "Delete an asset folder",
		Tags:        []string{assetsTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathParameter("id", "Folder ID", openapi3.NewStringSchema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("The folder has been successfully deleted.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("Folder not found.")),
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

// pathParameter builds a required path-parameter OpenAPI definition.
func pathParameter(name, description string, schema *openapi3.Schema) *openapi3.ParameterRef {
	return &openapi3.ParameterRef{Value: openapi3.NewPathParameter(name).WithDescription(description).WithSchema(schema)}
}

// queryParameter builds an optional query-parameter OpenAPI definition.
func queryParameter(name string, required bool, description string, schema *openapi3.Schema) *openapi3.ParameterRef {
	return &openapi3.ParameterRef{Value: openapi3.NewQueryParameter(name).WithRequired(required).WithDescription(description).WithSchema(schema)}
}

// tagSchema returns the schema for asset/folder tags.
func tagSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("name", openapi3.NewStringSchema()).
		WithProperty("color", openapi3.NewStringSchema())
}

// assetSchema returns the response schema for asset payloads.
func assetSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("_id", openapi3.NewStringSchema()).
		WithProperty("key", openapi3.NewStringSchema()).
		WithProperty("name", openapi3.NewStringSchema()).
		WithProperty("originalName", openapi3.NewStringSchema()).
		WithProperty("folderId", openapi3.NewStringSchema()).
		WithProperty("mimeType", openapi3.NewStringSchema()).
		WithProperty("size", openapi3.NewIntegerSchema()).
		WithProperty("tags", openapi3.NewArraySchema().WithItems(tagSchema())).
		WithProperty("metadata", openapi3.NewObjectSchema().WithAdditionalProperties(openapi3.NewStringSchema())).
		WithProperty("createdAt", openapi3.NewDateTimeSchema()).
		WithProperty("updatedAt", openapi3.NewDateTimeSchema()).
		WithProperty("deletedAt", openapi3.NewDateTimeSchema()).
		WithProperty("isDeleted", openapi3.NewBoolSchema())
}

// folderSchema returns the response schema for folder payloads.
func folderSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("_id", openapi3.NewStringSchema()).
		WithProperty("name", openapi3.NewStringSchema()).
		WithProperty("slug", openapi3.NewStringSchema()).
		WithProperty("parentFolderId", openapi3.NewStringSchema()).
		WithProperty("tags", openapi3.NewArraySchema().WithItems(tagSchema())).
		WithProperty("createdAt", openapi3.NewDateTimeSchema()).
		WithProperty("updatedAt", openapi3.NewDateTimeSchema()).
		WithProperty("deletedAt", openapi3.NewDateTimeSchema()).
		WithProperty("isDeleted", openapi3.NewBoolSchema())
}

// updateAssetSchema returns the request schema for asset update payloads.
func updateAssetSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("name", openapi3.NewStringSchema()).
		WithProperty("folderId", openapi3.NewStringSchema()).
		WithProperty("tags", openapi3.NewArraySchema().WithItems(tagSchema())).
		WithProperty("metadata", openapi3.NewObjectSchema().WithAdditionalProperties(openapi3.NewStringSchema()))
}

// createFolderSchema returns the request schema for folder creation payloads.
func createFolderSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("name", openapi3.NewStringSchema()).
		WithProperty("parentFolderId", openapi3.NewStringSchema()).
		WithProperty("tags", openapi3.NewArraySchema().WithItems(tagSchema())).
		WithRequired([]string{"name"})
}

// updateFolderSchema returns the request schema for folder update payloads.
func updateFolderSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("name", openapi3.NewStringSchema()).
		WithProperty("parentFolderId", openapi3.NewStringSchema()).
		WithProperty("tags", openapi3.NewArraySchema().WithItems(tagSchema()))
}

// assetPaginationMetaSchema returns the response schema for pagination metadata.
func assetPaginationMetaSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("page", openapi3.NewIntegerSchema()).
		WithProperty("total", openapi3.NewIntegerSchema()).
		WithProperty("limit", openapi3.NewIntegerSchema())
}

// paginatedAssetResponseSchema returns the response schema for paginated asset responses.
func paginatedAssetResponseSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("data", openapi3.NewArraySchema().WithItems(assetSchema())).
		WithProperty("meta", assetPaginationMetaSchema())
}

// paginatedFolderResponseSchema returns the response schema for paginated folder responses.
func paginatedFolderResponseSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("data", openapi3.NewArraySchema().WithItems(folderSchema())).
		WithProperty("meta", assetPaginationMetaSchema())
}
