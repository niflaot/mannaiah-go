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

DROP TABLE IF EXISTS coupon_usages;
DROP TABLE IF EXISTS coupon_included_tag_ids;
DROP TABLE IF EXISTS coupon_included_category_ids;
DROP TABLE IF EXISTS coupon_included_product_ids;
DROP TABLE IF EXISTS coupon_assigned_contact_ids;
DROP TABLE IF EXISTS coupon_assigned_emails;
DROP TABLE IF EXISTS coupons;
DROP TABLE IF EXISTS order_applied_coupons;