# Contacts

The `contacts` module is Mannaiah's **CRM backbone**. It owns the canonical customer/contact
identity, exposes a full CRUD REST API, and publishes integration events that downstream
modules (orders, campaigns, WooCommerce sync) consume to stay in sync.

## Responsibilities

- Maintain the single source of truth for every person or legal entity that interacts with Flock.
- Enforce business-level uniqueness and validation rules (one email per contact, consistent name
  conventions).
- Publish domain events so other modules react to contact lifecycle changes without tight coupling.
- Provide rich query capabilities including metadata key/value filtering and pagination.

## Contents

| File | Description |
|------|-------------|
| [01-CONTACTS.md](01-CONTACTS.md) | This overview |
| [02-DOMAIN.md](02-DOMAIN.md) | Domain model, validation rules, and business invariants |
| [03-API.md](03-API.md) | HTTP endpoints, request/response shapes, query parameters |
| [04-EVENTS.md](04-EVENTS.md) | Integration events published and their payload schemas |
