CREATE TABLE order_applied_coupons (
    id               INTEGER  NOT NULL PRIMARY KEY AUTOINCREMENT,
    order_id         TEXT     NOT NULL,
    coupon_id        TEXT     NOT NULL DEFAULT '',
    code             TEXT     NOT NULL,
    discount_type    TEXT     NOT NULL,
    discount_amount  REAL     NOT NULL DEFAULT 0,
    applied_at       DATETIME NOT NULL,
    UNIQUE (order_id, code)
);
