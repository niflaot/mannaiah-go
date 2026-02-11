# Auth Module

`module/auth` provides authentication and authorization services for module adapters.

## Packages
- `runtime`: module composition root wiring (`auth.New` facade target).
- `domain`: technology-neutral auth claims.
- `port`: verifier port for token validation.
- `application`: auth/permission use cases.
- `adapter/jwt`: JWKS-backed JWT verifier.

## Key Methods / Endpoints / Events
- Methods:
  - `auth.New(cfg, coreEnvironment, logger)`
  - `(*auth.Module).Require(ctx, authorizationHeader, requiredPermissions...)`
  - `(*auth.Module).IsUnauthorized(err)`
  - `(*auth.Module).IsForbidden(err)`
- Endpoints: none in this module.
- Events: none in this module.
