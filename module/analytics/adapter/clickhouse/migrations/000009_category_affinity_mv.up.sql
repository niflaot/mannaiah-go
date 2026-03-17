CREATE TABLE IF NOT EXISTS category_affinity_mv (
    contact_id     String,
    category_id    String,
    category_name  String,
    affinity_score Float64,
    total_spent    Float64,
    purchase_count UInt32
) ENGINE = SummingMergeTree()
ORDER BY (contact_id, category_id);
