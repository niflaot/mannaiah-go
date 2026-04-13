CREATE TABLE order_applied_coupons (
    id               BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
    order_id         VARCHAR(64)      NOT NULL,
    coupon_id        VARCHAR(64)      NOT NULL DEFAULT '',
    code             VARCHAR(128)     NOT NULL,
    discount_type    VARCHAR(32)      NOT NULL,
    discount_amount  DECIMAL(18,4)    NOT NULL DEFAULT 0,
    applied_at       DATETIME(3)      NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_order_applied_coupons_order_code (order_id, code),
    KEY idx_order_applied_coupons_order_id (order_id),
    KEY idx_order_applied_coupons_coupon_id (coupon_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
