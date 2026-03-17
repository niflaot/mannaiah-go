CREATE TABLE IF NOT EXISTS rfm_scores_mv (
    contact_id     String,
    recency_days   UInt32,
    frequency      UInt32,
    monetary       Float64,
    updated_at     DateTime64(3)
) ENGINE = ReplacingMergeTree(updated_at)
ORDER BY (contact_id);
