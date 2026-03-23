# Shipping TCC Carrier Adapter

TCC API client, request/response mappings, and provider implementation for quotation, mark generation, and tracking.

- Quotation endpoint: `/api/clientes/tarifas/v5/consultarliquidacion`
- Guide generation endpoint: `/api/clientes/remesas/grabardespacho7`
- Tracking endpoint: `/api/clientes/remesas/consultarestatusremesasv3`
- Base URLs are hardcoded by mode:
  - Sandbox: `https://testsomos.tcc.com.co`
  - Production: `https://somos.tcc.com.co`
