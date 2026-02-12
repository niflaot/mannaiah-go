# Auth Module

`module/auth` provides authentication and authorization services for module adapters.

## Packages
- `runtime`: module composition root wiring (`auth.New` facade target).
- `domain`: technology-neutral auth claims.
- `port`: verifier port for token validation.
- `application`: auth/permission use cases.
- `adapter/jwt`: JWKS-backed JWT verifier.
- `adapter/http`: auth HTTP endpoint adapter (`/check-auth`).

## Key Methods / Endpoints / Events
- Methods:
  - `auth.New(cfg, coreEnvironment, logger)`
  - `auth.OpenAPISpec()`
  - `(*auth.Module).Require(ctx, authorizationHeader, requiredPermissions...)`
  - `(*auth.Module).IsUnauthorized(err)`
  - `(*auth.Module).IsForbidden(err)`
  - `(*auth.Module).RegisterRoutes(router)`
  - `(*auth.Module).Load(loader)`
- Endpoints:
  - `GET /check-auth`
- Events: none in this module.
