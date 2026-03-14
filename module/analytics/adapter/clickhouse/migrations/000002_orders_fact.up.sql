CREATE TABLE IF NOT EXISTS orders_fact (
    order_id String,
    identifier String,
    realm String,
    contact_id String,
    current_status String,
    total_value Float64,
    item_count UInt32,
    created_at DateTime64(3),
    updated_at DateTime64(3)
) ENGINE = ReplacingMergeTree(updated_at)
ORDER BY (contact_id, order_id);
