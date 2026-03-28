# Shipping

The shipping module manages freight quotations, shipping mark generation, dispatch batch
orchestration, and carrier tracking. It provides a carrier-agnostic HTTP API backed by a
pluggable `CarrierProvider` registry. Currently two providers are registered:

- **TCC** — Transportes y Comunicaciones de Colombia (full API integration)
- **Manual** — a no-op provider for cases where the label is handled outside the system

---

## Architecture Overview

```
HTTP Layer (Fiber)
        │
        ├── Quotation Service
        │     └── CarrierProvider.Quote()  →  TCC REST API
        │
        ├── Mark Service
        │     └── CarrierProvider.GenerateMark()  →  TCC REST API
        │                                          →  Manual (no-op)
        │
        ├── Dispatch Service
        │     ├── Phase 1: DraftMark  →  QUOTED marks enter batch
        │     └── Phase 2: Close      →  Materializer → carrier per mark
        │
        └── Tracking Service
              └── TrackingProvider.GetTrackingHistory()  →  TCC REST API
```

---

## Table of Contents

| File | Contents |
|------|---------|
| [01-tcc/01-TCC.md](01-tcc/01-TCC.md) | Detailed TCC carrier integration (API calls, city codes, COD, error mapping) |
| [02-DOMAIN.md](02-DOMAIN.md) | All domain types, enums, and domain errors |
| [03-BATCH-DISPATCH.md](03-BATCH-DISPATCH.md) | Two-phase draft→close batch dispatch flow with examples |
| [04-API.md](04-API.md) | All HTTP endpoints, request/response schemas |
| [05-EVENTS.md](05-EVENTS.md) | Integration events, payload schemas |

---

## Carrier Summary

| Carrier ID | Type | Quote | Generate | Track | Balance Check |
|-----------|------|-------|----------|-------|--------------|
| `tcc` | `API` | ✅ | ✅ | ✅ | ✅ |
| `manual` | `MANUAL` | ❌ | ✅ (no-op) | ✅ (synthetic) | ❌ |

---

## Quick-Start Example

### Get a freight quote and generate a mark

```bash
# 1. Request a quote
curl -X POST https://api.example.com/shipping/quotations \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "carrierId": "tcc",
    "originCityCode": "11001",
    "destCityCode": "05001",
    "shipmentMode": "parcel",
    "declaredValue": 150000,
    "units": [{
      "description": "Camiseta roja M",
      "dimensions": { "heightCM": 5, "widthCM": 25, "depthCM": 30, "realWeightKG": 0.3 },
      "packageType": "1"
    }]
  }'
# Response: { "id": "q-uuid", "freightCost": 12500, "estimatedDays": 2, ... }

# 2. Generate a mark (direct, non-batch)
curl -X POST https://api.example.com/shipping/marks \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "orderId": "order-uuid",
    "carrierId": "tcc",
    "quotationId": "q-uuid",
    "shipmentMode": "parcel",
    "sender": { "name": "Bodega Central", "addressLine": "Cra 15 #80-30", "cityCode": "11001", "phone": "3001234567" },
    "recipient": { "name": "Juan García", "addressLine": "Av El Dorado 92-34", "cityCode": "05001", "phone": "3109876543" },
    "units": [{ "description": "Camiseta roja M", "dimensions": { "realWeightKG": 0.3 } }]
  }'
# Response: { "id": "m-uuid", "status": "GENERATED", "trackingNumber": "TCC-98765", "documentRef": "https://..." }
```
