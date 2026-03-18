# Shipping Module

`module/shipping` provides carrier-agnostic shipping quote use cases and adapters.

## Packages
- `runtime`: module composition-root wiring and module OpenAPI artifact.
- `domain`: shipping quote entities and validation invariants.
- `port`: carrier quote port contracts.
- `application/quote/service`: quote orchestration use case.
- `adapter/http`: HTTP endpoint adapter for quote requests.
- `adapter/tcc`: first carrier adapter implementation (TCC).

## Key Methods / Endpoints / Events
- Methods:
  - `shipping.New(cfg, logger)`
  - `(*shipping.Module).Load(loader)`
  - `(*shipping.Module).OpenAPISpec() *openapi3.T`
  - `(*shipping.Module).RegisterRoutes(router)`
  - `(*shipping.Module).SetAuthorizer(authorizer)`
- Endpoints:
  - `POST /shipping/quotes`
- Events: none in this module yet.
