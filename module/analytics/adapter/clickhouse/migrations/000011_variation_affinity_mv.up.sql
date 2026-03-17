CREATE TABLE IF NOT EXISTS variation_affinity_mv (
    contact_id      String,
    variation_name  String,
    variation_value String,
    affinity_score  Float64,
    total_spent     Float64,
    purchase_count  UInt32
) ENGINE = SummingMergeTree()
ORDER BY (contact_id, variation_name, variation_value);

INSERT INTO variation_affinity_mv
SELECT
    oi.contact_id                                                                AS contact_id,
    pvt.variation_name                                                           AS variation_name,
    pvt.variation_value                                                          AS variation_value,
    sum(oi.value * exp(-0.01 * dateDiff('day', oi.order_created_at, now64(3)))) AS affinity_score,
    sum(oi.value)                                                                AS total_spent,
    toUInt32(count(*))                                                           AS purchase_count
FROM order_items_fact oi FINAL
INNER JOIN product_variation_taxonomy pvt FINAL ON oi.product_id = pvt.product_id
GROUP BY oi.contact_id, pvt.variation_name, pvt.variation_value;
