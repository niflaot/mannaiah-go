# TCC Carrier Integration

TCC (**Transportes y Comunicaciones de Colombia**) is the primary API-backed carrier
integrated in the shipping module. This document covers the carrier's REST API surface, the
city-code normalisation rules, the COD charge formula, the full request/response mapping, and
error handling.

> **Disambiguation:** "TCC" here is the carrier name — not the distributed
> transactions protocol "Try-Confirm-Cancel". The batch draft→close flow that resembles
> two-phase commit is documented in [../03-BATCH-DISPATCH.md](../03-BATCH-DISPATCH.md).

---

## Connectivity

| Environment | Base URL |
|-------------|---------|
| Sandbox | `https://testsomos.tcc.com.co` |
| Production | `https://somos.tcc.com.co` |

**Auth:** `Authorization: Bearer <token>` on every request.

**Timeout:** configurable via `SHIPPING_TCC_REQUEST_TIMEOUT_MS` (default `10000` ms).

---

## City Code Normalisation

TCC expects **8-digit** city codes. Colombian DANE codes are 5 digits. The adapter
automatically appends `000` when a 5-digit code is supplied.

| Input | Sent to TCC |
|-------|------------|
| `"11001"` | `"11001000"` |
| `"05001"` | `"05001000"` |
| `"11001000"` | `"11001000"` _(unchanged)_ |

Always provide the 5-digit DANE code in the Mannaiah API — the adapter handles normalisation
transparently.

---

## Shipment Mode → Business Unit Mapping

TCC segments shipments into two business units. The `shipmentMode` field selects:

| `shipmentMode` | TCC `tipoenvio` | TCC `unidadNegocio` | Account number used |
|----------------|----------------|---------------------|---------------------|
| `"parcel"` | `1` | `1` | `ParcelAccountNumber` (config) |
| `"express"` | `2` | `2` | `ExpressAccountNumber` (config) |

---

## API Operations

### Quote (`POST /api/clientes/tarifas/v5/consultarliquidacion`)

**Request payload (sent to TCC):**

```json
{
  "tipoenvio": 1,
  "unidadNegocio": 1,
  "cuenta": "ACCT-PARCEL",
  "origen": "11001000",
  "destino": "05001000",
  "unidades": [
    {
      "descripcion": "Camiseta roja M",
      "pesoreal": 0.3,
      "pesovolumetrico": 0.0015,
      "alto": 5,
      "ancho": 25,
      "largo": 30,
      "tipoempaque": "1"
    }
  ],
  "valordeclarado": 150000
}
```

Volumetric weight is computed by the adapter as:
$$W_v = H \times W \times D \times 0.0004$$

For `H=5, W=25, D=30`: $W_v = 5 \times 25 \times 30 \times 0.0004 = 1.5$ kg.

**TCC response (success):**
```json
{
  "totaldespacho": 12500,
  "plazoentrega": 2,
  "moneda": "COP"
}
```

**Mapped to `QuotationResult`:**

| TCC Field | Mannaiah Field |
|-----------|---------------|
| `totaldespacho` | `FreightCost` |
| `plazoentrega` | `EstimatedDays` |
| `moneda` | `CurrencyCode` |

A `QuotationRecord` is persisted with TTL = 24 h (configurable). The raw TCC response is
stored in `quotations.raw_response` for auditability.

---

### Generate Mark (`POST /api/clientes/remesas/grabardespacho7`)

Used for both direct mark generation (`POST /shipping/marks`) and batch close materialisation.

**Request payload (sent to TCC):**

```json
{
  "tipoenvio": 1,
  "unidadNegocio": 1,
  "cuenta": "ACCT-PARCEL",
  "remitente": {
    "nombre": "Bodega Central",
    "razonsocial": "",
    "nit": "",
    "tipodocumento": "",
    "direccion": "Cra 15 #80-30",
    "ciudad": "11001000",
    "telefono": "3001234567",
    "email": "bodega@example.com"
  },
  "destinatario": {
    "nombre": "Juan García",
    "razonsocial": "",
    "nit": "",
    "tipodocumento": "",
    "direccion": "Av El Dorado 92-34",
    "ciudad": "05001000",
    "telefono": "3109876543",
    "email": ""
  },
  "unidades": [
    {
      "descripcion": "Camiseta roja M",
      "pesoreal": 0.3,
      "pesovolumetrico": 0.0015,
      "alto": 5,
      "ancho": 25,
      "largo": 30,
      "tipoempaque": "1"
    }
  ],
  "valordeclarado": 150000,
  "formadepago": "CTA",
  "valorcobro": 0,
  "porcentajemanejo": 0,
  "valorcobroreal": 0,
  "observaciones": ""
}
```

**`razonsocial`** is set from `Address.LegalName` when non-empty (B2B shipments).

**TCC response (success):**
```json
{
  "numeroremesa": "TCC-98765",
  "urlguia": "https://somos.tcc.com.co/guias/TCC-98765.pdf"
}
```

**Mapped to `ShippingMark`:**

| TCC Field | Mannaiah Field |
|-----------|---------------|
| `numeroremesa` | `TrackingNumber` |
| `urlguia` | `DocumentRef` |

Mark transitions to `GENERATED` (direct path) or `CREATED` (batch path).

---

## COD (Collect on Delivery)

When an order has a collect-on-delivery amount, TCC charges an additional handling fee.

**Formula:**

$$\text{ChargedAmount} = \text{round}\left(\text{Amount} \times \left(1 + \frac{\text{FeePercent}}{100}\right) \times 100\right) / 100$$

**Example:**

| Parameter | Value |
|-----------|-------|
| `CollectOnDeliveryAmount` | `200000` COP |
| `CollectOnDeliveryFeePercent` | `2.5%` |
| Fee absolute | `200000 × 0.025 = 5000` |
| `CollectOnDeliveryFeeAmount` | `5000` |
| `CollectOnDeliveryChargedAmount` | `205000` |

The `valorcobroreal` sent to TCC is the `ChargedAmount`. The `valorcobro` sent is the
original `CollectOnDeliveryAmount`. TCC uses both values to validate the fee.

---

### Tracking (`POST /api/clientes/remesas/consultarestatusremesasv3`)

**Request:**
```json
{ "numeroremesa": "TCC-98765" }
```

**Response (abbreviated):**
```json
{
  "estatus": [
    { "fecha": "2026-03-28 08:00", "codigo": "1001", "texto": "En origen", "ciudad": "BOGOTA" },
    { "fecha": "2026-03-28 14:00", "codigo": "3000", "texto": "Entregado", "ciudad": "MEDELLIN" }
  ]
}
```

**Status code mapping:**

| TCC `codigo` | Domain `TrackingStatus` |
|-------------|------------------------|
| `3000` | `COMPLETED` |
| `500` | `ORIGIN` |
| `4xx` (return codes) | `RETURN` |
| Incidence codes | `INCIDENCE` |
| _(anything else)_ | `PROCESSING` |

Tracking results are cached in Redis with configurable TTL to avoid hammering the TCC API on
repeated queries for the same tracking number.

---

## Error Handling

| Scenario | Behaviour |
|----------|-----------|
| HTTP timeout | Returns `ErrCarrierUnavailable`; mark transitions to `FAILED` in batch close |
| TCC API error body | Error text is parsed and stored in `ShippingMark.FailureReason` |
| Invalid city code | Rejected by TCC API; mark `FAILED` |
| Insufficient balance | `provider.CheckBalance()` fails pre-emptively before mark generation; mark never submitted |
| SSL / network | Configurable retry not implemented; single attempt per mark |

---

## Example: Full TCC Flow

```
Operator
  │
  ├─ POST /shipping/quotations
  │   carrierId: "tcc", origin: "11001", dest: "05001", units: [...]
  │
  │   Adapter:
  │     origin → "11001000", dest → "05001000"
  │     tipoenvio = 1 (parcel)
  │     POST /api/clientes/tarifas/v5/consultarliquidacion
  │       ← { totaldespacho: 12500, plazoentrega: 2, moneda: "COP" }
  │     Store QuotationRecord (TTL 24h)
  │   Response: { id: "q-001", freightCost: 12500, estimatedDays: 2 }
  │
  ├─ POST /shipping/marks
  │   carrierId: "tcc", quotationId: "q-001", sender: {...}, recipient: {...}
  │
  │   provider.CheckBalance() → OK
  │   provider.GenerateMark(&mark)
  │     POST /api/clientes/remesas/grabardespacho7
  │       ← { numeroremesa: "TCC-98765", urlguia: "https://.../TCC-98765.pdf" }
  │   mark.Status = GENERATED
  │   mark.TrackingNumber = "TCC-98765"
  │   mark.DocumentRef = "https://.../TCC-98765.pdf"
  │   Event: shipping.v1.mark.generated
  │
  └─ GET /shipping/tracking/TCC-98765?carrier=tcc
      POST /api/clientes/remesas/consultarestatusremesasv3
        ← [{ codigo: "3000", texto: "Entregado" }]
      TrackingHistory.GlobalStatus = COMPLETED
      Event: shipping.v1.tracking.updated
```
