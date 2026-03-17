CREATE TABLE IF NOT EXISTS product_variation_taxonomy (
    product_id      String,
    sku             String,
    variation_id    String,
    variation_name  String,
    variation_value String,
    updated_at      DateTime64(3)
) ENGINE = ReplacingMergeTree(updated_at)
ORDER BY (product_id, variation_id);
