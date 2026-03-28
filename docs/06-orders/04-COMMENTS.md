# Orders — Comment System

Order comments provide a threaded, append-based communication log attached to each order. Comments
serve two purposes: **internal operations notes** (not surfaced to external consumers) and
**external-facing notes** (visible to connected systems or customers via integrations).

---

## Comment Structure

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `string` | UUID, assigned at creation |
| `Author` | `string` | Required — identifier of who wrote the comment (user ID, system name, etc.) |
| `Comment` | `string` | Required — the comment body |
| `Internal` | `bool` | `true` = visible only to operators; `false` = may be surfaced externally |
| `OccurredAt` | `time.Time` | Timestamp of the comment |

---

## Internal vs External Comments

| `internal` | Audience | Typical use |
|------------|----------|-------------|
| `true` | Operations team only | Escalation notes, fraud flags, carrier contact logs, internal triage decisions |
| `false` | May be shown externally | Customer-facing updates, shipping notifications, automated status messages |

External consumers (e.g. WooCommerce webhook integrations) should filter on `internal = false`
when forwarding order notes.

---

## Comment Lifecycle

Comments are stored in `order_comments` with `occurred_at`. Unlike status history, comments
support **update and delete** operations for corrections.

| Operation | Endpoint | Permission |
|-----------|----------|------------|
| Add comment | `POST /orders/:id/comments` | `order:triage` |
| Update comment | `PATCH /orders/:id/comments/:commentId` | `order:triage` |
| Delete comment | `DELETE /orders/:id/comments/:commentId` | `order:triage` |

Only `author`, `comment`, and `internal` can be changed on update. `OccurredAt` is immutable.
Returns `ErrCommentNotFound` if the `commentId` does not exist on the given order.

---

## Status Notes vs Comments

The status system (`StatusEntry`) has its own `NoteOwner` and `Note` fields, which allow attaching
a short note directly to a status transition. This is distinct from the comment thread:

| Feature | Status Note | Comment |
|---------|-------------|---------|
| Attached to | A status transition event | The order directly (not a status) |
| Can be updated | No (append-only history) | Yes |
| Can be deleted | No | Yes |
| Has author | Via `StatusEntry.Author` | Via `Comment.Author` |
| Internal flag | No | Yes |

Use status notes for brief transition context ("awaiting payment"); use comments for richer
ongoing communication and triage notes.

---

## Example Comment Thread

```json
"comments": [
  {
    "id": "cmt-001",
    "author": "ops-maria",
    "comment": "Customer called — requested address change. Waiting for updated address.",
    "internal": true,
    "occurredAt": "2026-03-10T09:14:00Z"
  },
  {
    "id": "cmt-002",
    "author": "ops-maria",
    "comment": "Address confirmed. Ready to dispatch.",
    "internal": true,
    "occurredAt": "2026-03-10T11:30:00Z"
  },
  {
    "id": "cmt-003",
    "author": "shipping-agent",
    "comment": "Shipment label generated. Tracking: 9400111899560369517226.",
    "internal": false,
    "occurredAt": "2026-03-10T14:05:00Z"
  }
]
```
