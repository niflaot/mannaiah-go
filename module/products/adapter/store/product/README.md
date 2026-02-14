# products/adapter/store/product

GORM-based product repository implementation.

Persistence is normalized across root and relation tables:
- `products`
- `product_gallery_items`
- `product_gallery_excluded_realms`
- `product_gallery_variations`
- `product_datasheets`
- `product_datasheet_attributes`
- `product_variation_links`
- `product_variants`
- `product_variant_variations`

## Key methods / endpoints / events
- Methods: `NewRepository(db)`, `(*Repository).EnsureSchema`, `(*Repository).Create`, `(*Repository).GetByID`, `(*Repository).List`, `(*Repository).Update`, `(*Repository).Delete`
- Endpoints: data source for `/products` endpoints.
- Events: none.
