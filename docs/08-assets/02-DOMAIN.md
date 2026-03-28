# Assets — Domain Model

---

## `Asset`

Represents a single uploaded binary file.

| Field | Type | Notes |
|-------|------|-------|
| `ID` | `string (UUID)` | Primary key; JSON key `_id` |
| `Key` | `string` | Object-store path: `assets/<uuid>-<originalName>` |
| `Name` | `string` | Display name; defaults to `OriginalName` if omitted |
| `OriginalName` | `string` | Filename at upload time |
| `FolderID` | `string` | Optional parent folder UUID |
| `MimeType` | `string` | MIME type determined at upload |
| `Size` | `int64` | Payload size in bytes (must be > 0) |
| `Tags` | `[]Tag` | Up to 5 classification labels |
| `Metadata` | `map[string]string` | Arbitrary key-value pairs |
| `CreatedAt` | `time.Time` | When the record was created |
| `UpdatedAt` | `time.Time` | When the record was last changed |
| `IsDeleted` | `bool` | Soft-delete flag |
| `DeletedAt` | `*time.Time` | Soft-delete timestamp (nil when not deleted) |

### Storage Key Format

```
assets/<uuid>-<originalName>
```

Examples:

| UUID prefix | Original name | Resulting key |
|-------------|--------------|---------------|
| `a1b2c3d4` | `logo.png` | `assets/a1b2c3d4-logo.png` |
| `ff001234` | `product photo.JPG` | `assets/ff001234-product photo.JPG` |

The UUID prevents collisions between files with identical names. The key is stable after
creation — the only mutation is during JPEG conversion, where the extension changes (see
[05-JPG-WORKER.md](05-JPG-WORKER.md)).

### Validation Rules (`ValidateCreate`)

| Rule | Constraint |
|------|-----------|
| `Key` | Must be non-empty |
| `MimeType` | Must be non-empty |
| `Size` | Must be > 0 |
| `Tags` count | Maximum 5 |
| Tag uniqueness | No two tags may share the same `Name` |
| Tag `Name` format | `^[a-z0-9][a-z0-9_-]{0,31}$` (max 32 chars) |
| Tag `Color` format | `^#[0-9a-f]{6}$` (lowercase 6-digit hex) |
| Metadata key length | ≤ 128 characters |
| Metadata value length | ≤ 2048 characters |

`Normalize()` is called before validation: it trims and lowercases relevant string fields.

---

## `Tag`

A short label attached to an asset or folder for classification and filtering.

| Field | Type | Constraint |
|-------|------|-----------|
| `Name` | `string` | `^[a-z0-9][a-z0-9_-]{0,31}$` |
| `Color` | `string` | `^#[0-9a-f]{6}$` |

Tags are stored in a normalised child table (`asset_tags`, not a JSON column), enabling
efficient `WHERE name IN (...)` filtering without full table scans.

---

## Database Tables

### `assets`

| Column | Type | Constraints |
|--------|------|-------------|
| `id` | `varchar(64)` | PRIMARY KEY |
| `key` | `varchar(512)` | UNIQUE NOT NULL |
| `name` | `varchar(255)` | NOT NULL |
| `original_name` | `varchar(255)` | NOT NULL |
| `folder_id` | `varchar(64)` | nullable, INDEX |
| `mime_type` | `varchar(255)` | NOT NULL |
| `size` | `bigint` | NOT NULL |
| `created_at` | `datetime` | |
| `updated_at` | `datetime` | |
| `deleted_at` | `datetime` | nullable (soft-delete) |

### `asset_tags`

Normalised 1:N child table — no JSON blobs.

| Column | Type | Constraints |
|--------|------|-------------|
| `id` | `uint` | PRIMARY KEY (autoincrement) |
| `asset_id` | `varchar(64)` | INDEX, composite UNIQUE with `name` |
| `name` | `varchar(64)` | |
| `color` | `varchar(7)` | NOT NULL |

The composite unique index `(asset_id, name)` prevents duplicate tag names per asset.

### `asset_metadata`

| Column | Type | Constraints |
|--------|------|-------------|
| `id` | `uint` | PRIMARY KEY |
| `asset_id` | `varchar(64)` | INDEX, composite UNIQUE with `key` |
| `key` | `varchar(128)` | |
| `value` | `text` | NOT NULL |

---

## Port Errors

| Error | Meaning |
|-------|---------|
| `ErrNotFound` | Asset ID does not exist or is soft-deleted |
| `ErrFolderNotFound` | `FolderID` does not exist |
| `ErrFolderAlreadyExists` | Slug collision with another folder under the same parent |

---

## Example — Metadata Usage

Metadata keys have no reserved meaning in the core module. Common patterns used in production:

```json
{
  "metadata": {
    "product.id": "prod-uuid",
    "source": "woocommerce_import",
    "width_px": "1200",
    "height_px": "900"
  }
}
```

The Falabella module uses asset metadata to track image ordering and source URLs.
