CREATE TABLE IF NOT EXISTS order_items_fact (
    order_id String,
    contact_id String,
    sku String,
    alternate_name String,
    product_id String,
    quantity UInt32,
    value Float64,
    resolution_source String,
    order_created_at DateTime64(3),
    order_updated_at DateTime64(3)
) ENGINE = ReplacingMergeTree(order_updated_at)
ORDER BY (contact_id, order_id, sku, product_id);
