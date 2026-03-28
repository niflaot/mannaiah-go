# Products

The `products` module is Mannaiah's **Product Information Management (PIM)** system. It maintains
the master product catalogue, the variation/attribute model, a hierarchical category tree, and a
tag graph with correlation weights. All other modules that need product data — Falabella sync,
WooCommerce sync, campaign recommendations, analytics — read from this module through the
`ProductCatalog` port, never from its internal storage directly.

## Responsibilities

- Own the canonical product representation across all sales channels.
- Model product variants and dimensions (colour, size, custom text) as first-class entities.
- Store channel-specific presentation data (name, description, attributes) in named **Realms**
  without duplicating the product graph.
- Maintain a hierarchical category tree with rule-based product membership.
- Track the full tag vocabulary and weighted correlations for recommendation engines.

## Contents

| File | Description |
|------|-------------|
| [01-PRODUCTS.md](01-PRODUCTS.md) | This overview |
| [02-DOMAIN.md](02-DOMAIN.md) | Domain model — Products, Variants, Datasheets, Realms |
| [03-VARIATIONS.md](03-VARIATIONS.md) | Variation/dimension model |
| [04-CATEGORIES.md](04-CATEGORIES.md) | Category tree, filters, and product membership |
| [05-TAGS.md](05-TAGS.md) | Tag vocabulary and correlation graph |
| [06-API.md](06-API.md) | HTTP endpoints reference |
| [01-falabella/](01-falabella/) | Falabella marketplace integration |
