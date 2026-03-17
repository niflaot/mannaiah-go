CREATE TABLE IF NOT EXISTS category_affinity_mv (
    contact_id     String,
    category_id    String,
    category_name  String,
    affinity_score Float64,
    total_spent    Float64,
    purchase_count UInt32
) ENGINE = SummingMergeTree()
ORDER BY (contact_id, category_id);

INSERT INTO category_affinity_mv
SELECT
    oi.contact_id                                                                AS contact_id,
    pt.category_id                                                               AS category_id,
    any(pt.category_name)                                                        AS category_name,
    sum(oi.value * exp(-0.01 * dateDiff('day', oi.order_created_at, now64(3)))) AS affinity_score,
    sum(oi.value)                                                                AS total_spent,
    toUInt32(count(*))                                                           AS purchase_count
FROM order_items_fact oi FINAL
INNER JOIN product_taxonomy pt FINAL ON oi.product_id = pt.product_id
WHERE pt.category_id != ''
GROUP BY oi.contact_id, pt.category_id;
