# Auth Application Package

`application` implements authentication and permission-check use cases.

## Key Methods / Endpoints / Events
- Methods:
  - `application.NewService(environment, devAuthToken, devAuthScope, verifier, logger)`
  - `(*application.AuthService).Require(ctx, authorizationHeader, requiredPermissions...)`
  - `(*application.AuthService).Authenticate(ctx, authorizationHeader)`
  - `(*application.AuthService).Authorize(claims, requiredPermissions...)`
- Endpoints: none in this package.
- Events: none in this package.
