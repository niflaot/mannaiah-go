CREATE TABLE IF NOT EXISTS rfm_scores_mv (
    contact_id     String,
    recency_days   UInt32,
    frequency      UInt32,
    monetary       Float64,
    updated_at     DateTime64(3)
) ENGINE = ReplacingMergeTree(updated_at)
ORDER BY (contact_id);

INSERT INTO rfm_scores_mv
SELECT
    contact_id,
    toUInt32(dateDiff('day', max(created_at), now64(3))) AS recency_days,
    toUInt32(countDistinct(order_id))                     AS frequency,
    sum(total_value)                                       AS monetary,
    now64(3)                                               AS updated_at
FROM orders_fact FINAL
GROUP BY contact_id;
