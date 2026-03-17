CREATE TABLE IF NOT EXISTS variation_affinity_mv (
    contact_id      String,
    variation_name  String,
    variation_value String,
    affinity_score  Float64,
    total_spent     Float64,
    purchase_count  UInt32
) ENGINE = SummingMergeTree()
ORDER BY (contact_id, variation_name, variation_value);
