ALTER TABLE orders ADD COLUMN coupon_code TEXT NULL;
ALTER TABLE orders ADD COLUMN coupon_discount_amount REAL NULL;
ALTER TABLE orders ADD COLUMN coupon_discount_type TEXT NULL;

UPDATE orders
SET
    coupon_code = (
        SELECT code
        FROM order_applied_coupons oc
        WHERE oc.order_id = orders.id
        ORDER BY oc.applied_at DESC, oc.id DESC
        LIMIT 1
    ),
    coupon_discount_amount = (
        SELECT discount_amount
        FROM order_applied_coupons oc
        WHERE oc.order_id = orders.id
        ORDER BY oc.applied_at DESC, oc.id DESC
        LIMIT 1
    ),
    coupon_discount_type = (
        SELECT discount_type
        FROM order_applied_coupons oc
        WHERE oc.order_id = orders.id
        ORDER BY oc.applied_at DESC, oc.id DESC
        LIMIT 1
    )
WHERE EXISTS (
    SELECT 1
    FROM order_applied_coupons oc
    WHERE oc.order_id = orders.id
);

DROP TABLE IF EXISTS storefront_static_pages;
DROP TABLE IF EXISTS storefront_renderable_versions;
DROP TABLE IF EXISTS storefront_renderables;

DROP TABLE IF EXISTS coupon_usages;
DROP TABLE IF EXISTS coupon_included_tag_ids;
DROP TABLE IF EXISTS coupon_included_category_ids;
DROP TABLE IF EXISTS coupon_included_product_ids;
DROP TABLE IF EXISTS coupon_assigned_contact_ids;
DROP TABLE IF EXISTS coupon_assigned_emails;
DROP TABLE IF EXISTS coupons;
DROP TABLE IF EXISTS order_applied_coupons;

DROP TABLE IF EXISTS campaigns;
DROP TABLE IF EXISTS segments;

DROP TABLE IF EXISTS rfm_group_conditions;
DROP TABLE IF EXISTS rfm_band_configs;
DROP TABLE IF EXISTS rfm_groups;
