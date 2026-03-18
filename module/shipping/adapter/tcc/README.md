# Shipping TCC Adapter Package

`adapter/tcc` implements the shipping quote gateway for the TCC API.

## Key Methods / Endpoints / Events
- Methods:
  - `tcc.NewClient(cfg)`
  - `(*tcc.Client).Quote(ctx, request)`
- Endpoints:
  - outbound `POST /api/clientes/tarifas/v5/consultarliquidacion` (TCC)
- Events: none in this package.
