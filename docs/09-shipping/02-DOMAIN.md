# Shipping — Domain Model

---

## Core Entities

### `ShippingMark`

A shipping mark represents a single shipment label assigned to an order. It is the central
entity in the shipping module.

| Field | Type | Notes |
|-------|------|-------|
| `ID` | `string (UUID)` | Primary key |
| `OrderID` | `string` | Associated order |
| `CarrierID` | `string` | `"tcc"` or `"manual"` |
| `TrackingNumber` | `string` | Assigned by carrier on generation; blank until then |
| `Status` | `MarkStatus` | See enum below |
| `DocumentType` | `MarkDocumentType` | `LINK` (URL) or `FILE` (stored blob) |
| `DocumentRef` | `string` | Shipping label URL or storage path |
| `ManifestType` | `MarkDocumentType` | Type of manifest document |
| `ManifestRef` | `string` | Manifest URL or path |
| `Sender` | `Address` | Sender address |
| `Recipient` | `Address` | Recipient address |
| `Units` | `[]PackageUnit` | Physical packages |
| `TotalWeight` | `float64` | Sum of `RealWeightKG` across all units |
| `TotalVolumetricWeight` | `float64` | Sum of `VolumetricWeightKG` across all units |
| `DeclaredValue` | `float64` | Declared monetary value |
| `PaymentForm` | `string` | Freight payment form code |
| `CollectOnDeliveryAmount` | `float64` | Requested COD amount |
| `CollectOnDeliveryFeePercent` | `float64` | Carrier fee percentage |
| `CollectOnDeliveryChargedAmount` | `float64` | Final COD `amount × (1 + fee/100)` |
| `Observations` | `string` | Free-text notes sent to carrier |
| `DispatchBatchID` | `*string` | Assigned batch UUID (nil for directly generated) |
| `QuotationID` | `*string` | Source quotation UUID (nil if no quote used) |
| `QuotedFreightCost` | `float64` | Freight cost snapshot from quotation |
| `ShipmentMode` | `ShipmentMode` | `"parcel"` or `"express"` |
| `DraftSnapshot` | `string` | JSON snapshot of mark fields at draft time |
| `FailureReason` | `string` | Carrier error message; set on `FAILED` transition |
| `CreatedAt` / `UpdatedAt` | `time.Time` | |

---

### `MarkStatus` Enum

| Value | Description |
|-------|-------------|
| `PENDING` | Initial state before any processing |
| `QUOTED` | Drafted into an open dispatch batch; awaiting close |
| `GENERATED` | Carrier generated the mark outside the batch flow |
| `CREATED` | Carrier confirmed during batch close |
| `FAILED` | Carrier submission rejected |
| `VOIDED` | Locally voided by operator |
| `REMOVED` | Removed from a batch before close |

---

### Status Transition Reference

```
PENDING ──────────────────────────────────▶ GENERATED  (GenerateMark, direct)
PENDING ──────────────────────────────────▶ FAILED     (GenerateMark, carrier error)
PENDING ──────────────────────────────────▶ QUOTED     (DraftMark into batch)
QUOTED  ──────────────────────────────────▶ CREATED    (batch close, carrier success)
QUOTED  ──────────────────────────────────▶ FAILED     (batch close, carrier error)
QUOTED  ──────────────────────────────────▶ REMOVED    (RemoveDraftMark)
GENERATED / CREATED / FAILED ─────────────▶ VOIDED     (VoidMark)
```

No FSM is enforced — transitions are driven by service logic, not a state machine. Only
`QUOTED` marks may be removed from a batch.

---

### `Address`

| Field | Type | Notes |
|-------|------|-------|
| `Name` | `string` | Contact display name |
| `LegalName` | `string` | Company legal name; overrides `Name` at carrier level for B2B |
| `ID` | `string` | Legal document number |
| `IDType` | `string` | Document type (`CC`, `NIT`, etc.) |
| `AddressLine` | `string` | Street address |
| `CityCode` | `string` | TCC city code (5 or 8 digits; see [01-tcc/01-TCC.md](01-tcc/01-TCC.md)) |
| `Phone` | `string` | Contact phone |
| `Email` | `string` | Contact email |

`Normalize()` trims whitespace from every field.

---

### `Dimensions`

| Field | Type | Notes |
|-------|------|-------|
| `HeightCM` | `float64` | |
| `WidthCM` | `float64` | |
| `DepthCM` | `float64` | |
| `RealWeightKG` | `float64` | Physical weight |
| `VolumetricWeightKG` | `float64` | Auto-computed as `H × W × D × 0.0004` if zero |
| `DeclaredValueCOP` | `float64` | Declared monetary value |

---

### `PackageUnit`

| Field | Type |
|-------|------|
| `Description` | `string` |
| `Dimensions` | `Dimensions` |
| `PackageType` | `string` |

---

### `QuotationResult`

| Field | Type | Notes |
|-------|------|-------|
| `ID` | `string` | Stored quotation record ID |
| `OrderID` | `string` | |
| `CarrierID` | `string` | |
| `OriginCityCode` / `DestCityCode` | `string` | |
| `FreightCost` | `float64` | Carrier-quoted freight cost |
| `EstimatedDays` | `int` | Estimated transit days |
| `CurrencyCode` | `string` | e.g. `"COP"` |
| `ExpiresAt` | `time.Time` | Quote validity; default +24 h |
| `CollectOnDeliveryAmount` | `float64` | Requested COD amount |
| `CollectOnDeliveryFeePercent` | `float64` | |
| `CollectOnDeliveryFeeAmount` | `float64` | `amount × feePercent/100` |
| `CollectOnDeliveryChargedAmount` | `float64` | `round(amount × (1+fee/100) × 100)/100` |

---

### `DispatchBatch`

| Field | Type | Notes |
|-------|------|-------|
| `ID` | `string (UUID)` | Primary key |
| `CarrierID` | `string` | All marks in the batch must share this carrier |
| `Status` | `BatchStatus` | `OPEN` or `CLOSED` |
| `CreatedBy` | `string` | Operator identifier |
| `MarkIDs` | `[]string` | IDs of drafted marks |
| `CreatedAt` | `time.Time` | |
| `ClosedAt` | `*time.Time` | Set when batch closes |

---

### `TrackingHistory`

| Field | Type |
|-------|------|
| `CarrierID` | `string` |
| `TrackingNumber` | `string` |
| `GlobalStatus` | `TrackingStatus` |
| `LastUpdate` | `time.Time` |
| `History` | `[]TrackingEvent` |

**`TrackingEvent`:** `{ Date time.Time, Code string, Text string, City string, Status TrackingStatus }`

**`TrackingStatus` enum:** `PROCESSING` | `ORIGIN` | `COMPLETED` | `RETURN` | `INCIDENCE`

---

## Database Tables

### `shipping_marks`

Address fields are stored flattened (e.g. `sender_name`, `sender_city_code`, ...).

| Column | Type | Notes |
|--------|------|-------|
| `id` | `varchar` | PK |
| `order_id` | `varchar` | INDEX |
| `carrier_id` | `varchar` | INDEX |
| `tracking_number` | `varchar` | UNIQUE, nullable |
| `status` | `varchar` | |
| `document_type` / `document_ref` | `varchar` | |
| `manifest_type` / `manifest_ref` | `varchar` | |
| sender/recipient fields (8 each) | `varchar` | Flattened `Address` |
| `total_weight` / `total_volumetric_weight` | `float` | |
| `declared_value` | `float` | |
| `payment_form` | `varchar` | |
| `collect_on_delivery_*` (3 fields) | `float` | |
| `observations` | `varchar` | |
| `dispatch_batch_id` | `varchar` | INDEX, nullable |
| `quotation_id` | `varchar` | nullable |
| `quoted_freight_cost` | `float` | |
| `shipment_mode` | `varchar` | |
| `draft_snapshot` | `text` (JSON) | |
| `failure_reason` | `varchar` | |
| `created_at` / `updated_at` | `timestamp` | |

### `shipping_mark_units`

| Column | Type |
|--------|------|
| `id` | `varchar` PK |
| `shipping_mark_id` | `varchar` INDEX |
| `description` | `varchar` |
| `package_type` | `varchar` |
| `height_cm`, `width_cm`, `depth_cm` | `float` |
| `real_weight_kg`, `volumetric_weight_kg` | `float` |
| `declared_value` | `float` |

### `dispatch_batches`

| Column | Type | Notes |
|--------|------|-------|
| `id` | `varchar` | PK |
| `carrier_id` | `varchar` | INDEX |
| `status` | `varchar` | INDEX |
| `created_by` | `varchar` | |
| `created_at` | `timestamp` | |
| `closed_at` | `timestamp` | nullable |

Mark membership is tracked by `shipping_marks.dispatch_batch_id` — no join table.

### `quotations`

| Column | Type | Notes |
|--------|------|-------|
| `id` | `varchar` | PK |
| `order_id` | `varchar` | INDEX |
| `carrier_id` | `varchar` | INDEX |
| `origin_city_code` / `dest_city_code` | `varchar` | |
| `freight_cost` | `float` | |
| `estimated_days` | `int` | |
| `currency_code` | `varchar` | |
| `expires_at` | `timestamp` | |
| `request_snapshot` | `text` (JSON) | Full request for replay |
| `raw_response` | `text` | Raw carrier response body |
| `created_at` | `timestamp` | |

---

## Domain Errors

| Error | Meaning |
|-------|---------|
| `ErrInvalidID` | ID is blank |
| `ErrCarrierNotSupported` | No provider for this carrier |
| `ErrQuotationNotSupported` | Carrier does not support quoting |
| `ErrTrackingNotSupported` | Carrier does not support tracking |
| `ErrInsufficientBalance` | Carrier balance check failed |
| `ErrInvalidMarkStatus` | Status transition not allowed |
| `ErrInvalidBatchStatus` | Batch is in wrong state |
| `ErrBatchClosed` | Mutation attempted on a closed batch |
| `ErrBatchCarrierMismatch` | Mark carrier differs from batch carrier |
| `ErrMarkNotDraft` | Remove called on non-QUOTED mark |
| `ErrInvalidShipmentMode` | Mode is not `parcel` or `express` |
| `ErrNotFound` | Entity not found |
