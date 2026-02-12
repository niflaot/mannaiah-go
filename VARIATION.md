# Mannaiah Variation Architecture
# Mannaiah Variation Architecture

> This document provides a detailed, implementation-level breakdown of the Variation system in Mannaiah, explaining how product attributes like Color and Size are defined and managed.

---

## 1. Overview

**Variations** in Mannaiah are reusable entities that define potential attributes for products. Instead of embedding "Red" or "Small" text strings directly into every product, we create central Variation records.

This allows for:
- **Consistency**: "Navy Blue" is defined once and referenced everywhere.
- **Rich Data**: A color variation stores both a name ("Navy Blue") and a machine-readable value (`#000080`).
- **Filtering**: Products can be efficiently filtered by variation IDs.

---

## 2. The Variation Model

The core entity is the `Variation`, defined in `src/features/variations/schemas/variation.schema.ts`.

### Core Fields

| Field | Type | Description |
|---|---|---|
| `_id` | UUID | Unique system identifier. |
| `name` | String | The human-readable label (e.g., "XL", "Midnight Blue"). |
| `value` | String | The underlying value. For colors, this is typically a hex code. For sizes/text, it matches the name. |
| `definition` | Enum | The **type** of variation. See below. |
| `timestamps` | Date | `createdAt` and `updatedAt` are automatically managed. |

### The Definition Enum

The `definition` field is strict about what kind of attribute this variation represents. It is an enum (`VariationDefinition`):

| Enum Value | Purpose | Example Name | Example Value |
|---|---|---|---|
| `COLOR` | Represents a visual color. | "Red" | `#FF0000` |
| `SIZE` | Represents a clothing or physical size. | "XL" | `XL` |
| `TEXT` | Generic attributes (material, style). | "Cotton" | `cotton` |

> **Important Constraint**: The `definition` of a variation is **immutable**. You cannot change a `COLOR` variation into a `SIZE` variation after creation.

---

## 3. API Routes & Permissions

Variation management is handled by the `VariationsController` in `src/features/variations/variations.controller.ts`.

All routes are protected by:
1.  **`JwtAuthGuard`**: Requires a valid JWT.
2.  **`PermissionsGuard`**: Checks strictly for the required scope.

| Method | Endpoint | Description | Required Permission |
|---|---|---|---|
| `POST` | `/variations` | Create a new variation. | `variations:create` |
| `GET` | `/variations` | List all variations. | `variations:read` |
| `GET` | `/variations/:id` | Get single variation by UUID. | `variations:read` |
| `PATCH` | `/variations/:id` | Update name or value. **Ignores** any attempts to change `definition`. | `variations:update` |
| `DELETE` | `/variations/:id` | Soft-delete a variation. | `variations:delete` |

> **Note**: Permissions also support the wildcard `variations:manage`, which grants all of the above.

---

## 4. Data Payloads

### Create Variation Examples

#### Creating a Color
```json
{
  "name": "Forest Green",
  "definition": "COLOR",
  "value": "#228B22"
}
```

#### Creating a Size
```json
{
  "name": "XXL",
  "definition": "SIZE",
  "value": "XXL"
}
```

### Update Variation Example

```json
{
  "name": "Dark Forest Green",
  "value": "#006400"
}
```

> **Warning**: If you include `"definition": "SIZE"` in the update payload for an existing Color variation, the backend **silently deletes** that field from the payload and only updates the name/value. The type remains `COLOR`.
