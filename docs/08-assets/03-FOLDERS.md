# Assets — Folder System

---

## Overview

Folders provide a hierarchical organisational layer on top of the flat object store. An asset
may belong to at most one folder. Folders may nest to arbitrary depth. The folder tree is
assembled in-memory at query time — there is no adjacency table, only a `parent_folder_id`
foreign key chain.

---

## `Folder` Domain Type

| Field | Type | Notes |
|-------|------|-------|
| `ID` | `string (UUID)` | Primary key |
| `Name` | `string` | User-facing display label |
| `Slug` | `string` | URL-safe identifier, auto-built from `Name` if omitted |
| `ParentFolderID` | `string` | Optional UUID of parent folder |
| `Tags` | `[]Tag` | Up to 5 classification labels |
| `Children` | `[]Folder` | Populated in tree responses only — not persisted |
| `CreatedAt` | `time.Time` | |
| `UpdatedAt` | `time.Time` | |
| `IsDeleted` | `bool` | Soft-delete flag |
| `DeletedAt` | `*time.Time` | nil when not deleted |

---

## Slug Generation

Slugs are derived from the folder name via `BuildFolderSlug(name)`:

| Input | Output |
|-------|--------|
| `"My Folder"` | `"my-folder"` |
| `"Productos 2026"` | `"productos-2026"` |
| `"---logo --- brand"` | `"logo-brand"` |
| `"Imágenes & Arte"` | `"imgenes-arte"` _(non-ASCII stripped)_ |

**Rules applied in order:**
1. Lowercase entire string.
2. Replace spaces with hyphens.
3. Strip any character that is not `[a-z0-9-]`.
4. Collapse repeated consecutive hyphens to one.
5. Trim leading and trailing hyphens.

Slugs must be unique **within the same parent scope**. The database enforces this with a
composite unique index on `(parent_folder_id, slug)`. Two folders with the same name can
co-exist under different parents without conflict.

> `parent_folder_id = NULL` is treated as the root level. Each root-level folder must have a
> unique slug among other root-level folders.

---

## Database Tables

### `asset_folders`

| Column | Type | Constraints |
|--------|------|-------------|
| `id` | `varchar(64)` | PRIMARY KEY |
| `name` | `varchar(255)` | NOT NULL |
| `slug` | `varchar(191)` | NOT NULL, composite UNIQUE (priority 2) |
| `parent_folder_id` | `varchar(64)` | nullable, INDEX, composite UNIQUE (priority 1) |
| `created_at` | `datetime` | |
| `updated_at` | `datetime` | |
| `deleted_at` | `datetime` | nullable (soft-delete) |

### `folder_tags`

| Column | Type | Constraints |
|--------|------|-------------|
| `id` | `uint` | PRIMARY KEY |
| `folder_id` | `varchar(64)` | INDEX, composite UNIQUE with `name` |
| `name` | `varchar(64)` | |
| `color` | `varchar(7)` | NOT NULL |

---

## Cycle Detection

When updating a folder's `ParentFolderID`, the repository walks the proposed parent chain
upward to verify the folder being updated does not appear as an ancestor of its new parent
(which would create a cycle).

**Example cycle attempt:**
```
Current: A → B → C
Attempt: Set A's parent to C
Result: rejected — C is a descendant of A
```

---

## Tree Assembly

`GetFolderTree` loads all non-deleted folders (ordered `parent_folder_id ASC, name ASC`) in
a single query, then assembles the tree in-memory using a two-pass algorithm:

1. **Index pass**: build `map[id]*Folder` from the flat list.
2. **Link pass**: for each folder, append it to `parent.Children` if it has a parent. Root
   folders (no parent) are kept in the root result slice.

The result is a fully nested structure without any additional queries.

---

## Soft Delete Behaviour

When a folder is soft-deleted:
1. The `deleted_at` timestamp is set on the `asset_folders` row.
2. All `assets.folder_id` fields pointing to this folder are set to `NULL` — assets are
   **detached** from the folder but not themselves deleted.

This means deleting a folder never removes files. Assets become "unfoldered" and will appear
in list results without a `folderId` value.

---

## Example Folder Structure

```
(root)
├── product-images/
│   ├── web/          ← assets tagged "web"
│   └── print/        ← assets tagged "print"
└── brand/
    └── logos/        ← SVG and PNG logos
```

Corresponding `asset_folders` rows:

| id | name | slug | parent_folder_id |
|----|------|------|-----------------|
| `f1` | Product Images | `product-images` | NULL |
| `f2` | Web | `web` | `f1` |
| `f3` | Print | `print` | `f1` |
| `f4` | Brand | `brand` | NULL |
| `f5` | Logos | `logos` | `f4` |

A `GET /assets/folders/tree` response flattens this into:

```json
{
  "data": [
    {
      "id": "f1", "name": "Product Images", "slug": "product-images",
      "children": [
        { "id": "f2", "name": "Web", "slug": "web", "children": [] },
        { "id": "f3", "name": "Print", "slug": "print", "children": [] }
      ]
    },
    {
      "id": "f4", "name": "Brand", "slug": "brand",
      "children": [
        { "id": "f5", "name": "Logos", "slug": "logos", "children": [] }
      ]
    }
  ]
}
```
