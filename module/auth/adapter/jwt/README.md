# Auth JWT Adapter Package

`adapter/jwt` provides JWKS-backed JWT verification for auth ports.

## Key Methods / Endpoints / Events
- Methods:
  - `jwt.NewVerifier(cfg)`
  - `(*jwt.Verifier).Verify(ctx, token)`
  - `jwt.EncodeBase64URLInteger(value)`
  - `jwt.EncodeBase64URLInt(value)`
- Endpoints: none in this package.
- Events: none in this package.
