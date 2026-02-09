# Watermill Correlation Internal Package

`watermill/internal/correlation` stores internal context helpers for correlation id propagation.

## Responsibilities
- Store correlation ids in `context.Context`.
- Retrieve correlation ids from `context.Context`.

## Key Methods / Endpoints / Events
- Methods:
  - `correlation.WithContext(ctx, correlationID)`
  - `correlation.FromContext(ctx)`
- Endpoints: none in this package.
- Events: none emitted directly by this package.
