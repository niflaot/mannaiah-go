# Mannaiah Product Architecture

> This document provides a detailed, implementation-level breakdown of the Product Catalog architecture in Mannaiah, focusing on the Multi-Realm data model and API structure.

---

## 1. Overview

The Mannaiah Product Catalog is designed to be **Multi-Realm**. This means a single product entity (identified by one unique ID and internal SKU) can present itself differently depending on the context (the "Realm") it is being viewed in.

For example, a product might have:
- A generic name and description for the **'default'** realm (your main e-commerce site).
- A specialized, shorter name and different attribute set for a **'marketplace-a'** realm.
- A wholesale-specific description for a **'b2b'** realm.

This architecture avoids duplicating products for different sales channels. Instead, we attach multiple **Datasheets** to a single Product entity.

---

## 2. The Product Model

The core entity is the `Product`, defined in `src/features/products/schemas/product.schema.ts`.

### Core Fields

| Field | Type | Description |
|---|---|---|
| `_id` | UUID | Unique system identifier. |
| `sku` | String | **Stock Keeping Unit**. Unique index. The primary human-readable identifier for the product. |
| `timestamps` | Date | `createdAt` and `updatedAt` are automatically managed. |

### The Gallery (`gallery`)

The gallery is a list of images associated with the product. It supports realm-based visibility and variation-specific filtering.

| Field | Description |
|---|---|
| `assetId` | Reference to the **Assets** system (S3 object ID). |
| `isMain` | Boolean flag. Only **one** image per product can be true. |
| `excludedRealms` | Array of strings. If a realm ID is listed here (e.g., `['b2b']`), this image will **not** be shown in that realm. |
| `variationIds` | Array of strings. Links the image to specific variations (e.g., "Red"). If set, the frontend can filter the gallery to show only relevant images when a user selects that variation. |

### The Component: Datasheets (`datasheets`)

**This is the heart of the multi-realm system.** A datasheet contains all the text and metadata that might vary between sales channels.

| Field | Description |
|---|---|
| `realm` | **Required.** The identifier key for the realm (e.g., `'default'`, `'pos'`, `'external-market'`). |
| `name` | The product name as it should appear in this realm. |
| `description` | HTML or text description specific to this realm. |
| `attributes` | A flexible Key-Value object (`Record<string, any>`) for storing realm-specific specs, categories, or flags (e.g., `{ "material": "wool", "season": "winter" }`). |

### Variations & Variants

Mannaiah uses a two-step system for product options:

1.  **Variations (`variations`)**: A simple list of IDs pointing to the `Variations` collection.
    *   *Purpose*: Defines "What options does this product have?"
    *   *Example*: `['id-for-color-red', 'id-for-size-large']`.

2.  **Variants (`variants`)**: Concrete combinations of variations that result in a specific physical item.
    *   *Purpose*: Defines "What specific SKU corresponds to this combination of options?"
    *   *Structure*:
        *   `variationIds`: `['id-for-color-red', 'id-for-size-large']`
        *   `sku`: `'SHIRT-RED-L'` (Can be different from the main product SKU).

---

## 3. Understanding Realms

A **Realm** is strictly a logical concept; it does not have a strict database schema validation enum, allowing flexibility to add new sales channels dynamically.

However, the convention implies:
*   **`'default'`**: The primary data set, used when no specific realm is requested or as a fallback.
*   **External Integrations**: Specific keys (e.g., `'marketplace-x'`) used to store data required by external sync processes.

**Logic Flow:**
When fetching a product for a specific channel, the consumer should looking for a datasheet where `realm === target_realm`. If missing, the consumer typically falls back to `realm === 'default'`.

---

## 4. API Routes & Permissions

Product management is handled by the `ProductsController` in `src/features/products/products.controller.ts`.

All routes are protected by:
1.  **`JwtAuthGuard`**: Requires a valid JWT.
2.  **`PermissionsGuard`**: Checks strictly for the required scope.

| Method | Endpoint | Description | Required Permission |
|---|---|---|---|
| `POST` | `/products` | Create a new product. Validates that referenced assets and variations exist. | `products:create` |
| `GET` | `/products` | List all products. | `products:read` |
| `GET` | `/products/:id` | Get single product by UUID. | `products:read` |
| `PATCH` | `/products/:id` | Update product fields. Merges updates (e.g., adding a datasheet creates or updates it). | `products:update` |
| `DELETE` | `/products/:id` | Soft-delete a product. | `products:delete` |

> **Note**: Permissions also support the wildcard `products:manage`, which grants all of the above.

---

## 5. Data Payloads

### Create Product Example

Here is a comprehensive JSON payload for creating a complex product with multiple realms and variants.

```json
{
  "sku": "TSHIRT-001",
  "gallery": [
    {
      "assetId": "asset-uuid-1",
      "isMain": true,
      "excludedRealms": []
    },
    {
      "assetId": "asset-uuid-2",
      "isMain": false,
      "excludedRealms": ["b2b"],
      "variationIds": ["var-id-red"]
    }
  ],
  "datasheets": [
    {
      "realm": "default",
      "name": "Classic T-Shirt",
      "description": "<p>A comfortable cotton t-shirt.</p>",
      "attributes": {
        "material": "Cotton",
        "care": "Machine wash"
      }
    },
    {
      "realm": "b2b-wholesale",
      "name": "Bulk T-Shirt (Pack of 10)",
      "description": "Wholesale pack for retailers.",
      "attributes": {
        "pack_size": 10,
        "pricing_tier": "gold"
      }
    }
  ],
  "variations": [
    "var-id-red",
    "var-id-blue",
    "var-id-large",
    "var-id-medium"
  ],
  "variants": [
    {
      "variationIds": ["var-id-red", "var-id-large"],
      "sku": "TSHIRT-001-RED-L"
    },
    {
      "variationIds": ["var-id-blue", "var-id-medium"],
      "sku": "TSHIRT-001-BLUE-M"
    }
  ]
}
```
