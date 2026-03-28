# Orders

The `orders` module is Mannaiah's **Order Management System (OMS)**. It owns the canonical order
representation across all sales channels, resolves customer identities and product references, and
publishes integration events consumed by shipping and analytics downstream.

## Responsibilities

- Maintain the authoritative order record with rich status history and comment threads.
- Resolve order line items to their canonical product IDs (by SKU, with fallback to alternate name).
- Guard WooCommerce-owned orders against accidental API overwrites.
- Publish domain events allowing shipping, analytics, and notification modules to react to lifecycle changes.
- Provide triage tooling (status transitions, internal/external comments, notes) for operations teams.

## Contents

| File | Description |
|------|-------------|
| [01-ORDERS.md](01-ORDERS.md) | This overview |
| [02-DOMAIN.md](02-DOMAIN.md) | Full domain model — Order, Item, Status, Comment, Shipping |
| [03-LIFECYCLE.md](03-LIFECYCLE.md) | Order lifecycle, status machine, and status history |
| [04-COMMENTS.md](04-COMMENTS.md) | Comment system — internal vs external, threading, authorship |
| [05-API.md](05-API.md) | HTTP endpoints reference |
| [06-EVENTS.md](06-EVENTS.md) | Integration events published |
