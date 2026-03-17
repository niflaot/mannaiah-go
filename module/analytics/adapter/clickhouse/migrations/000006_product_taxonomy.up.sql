CREATE TABLE IF NOT EXISTS product_taxonomy (
    product_id    String,
    tag           String,
    category_id   String,
    category_name String,
    updated_at    DateTime64(3)
) ENGINE = ReplacingMergeTree(updated_at)
ORDER BY (product_id, tag, category_id);
