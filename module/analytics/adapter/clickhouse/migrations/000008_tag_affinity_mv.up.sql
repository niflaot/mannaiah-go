CREATE TABLE IF NOT EXISTS tag_affinity_mv (
    contact_id     String,
    tag            String,
    affinity_score Float64,
    total_spent    Float64,
    purchase_count UInt32
) ENGINE = SummingMergeTree()
ORDER BY (contact_id, tag);
