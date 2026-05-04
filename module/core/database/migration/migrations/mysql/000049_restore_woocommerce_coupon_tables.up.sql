CREATE TABLE IF NOT EXISTS coupons (
    id                     VARCHAR(64)      NOT NULL,
    code                   VARCHAR(128)     NOT NULL,
    origin                 VARCHAR(128)     NOT NULL DEFAULT '',
    discount_type          VARCHAR(32)      NOT NULL,
    discount_amount        DECIMAL(18,4)    NOT NULL DEFAULT 0,
    max_usages_global      INT              NULL,
    max_usages_per_email   INT              NULL,
    active                 TINYINT(1)       NOT NULL DEFAULT 1,
    expires_at             DATETIME(3)      NULL,
    woocommerce_id         INT              NULL,
    created_at             DATETIME(3)      NULL,
    updated_at             DATETIME(3)      NULL,
    deleted_at             DATETIME(3)      NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_coupons_code (code),
    KEY idx_coupons_origin (origin),
    KEY idx_coupons_active (active),
    KEY idx_coupons_woocommerce_id (woocommerce_id),
    KEY idx_coupons_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS coupon_assigned_emails (
    id         BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
    coupon_id  VARCHAR(64)      NOT NULL,
    email      VARCHAR(320)     NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_coupon_assigned_emails_coupon_email (coupon_id, email),
    KEY idx_coupon_assigned_emails_coupon_id (coupon_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS coupon_assigned_contact_ids (
    id          BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
    coupon_id   VARCHAR(64)      NOT NULL,
    contact_id  VARCHAR(64)      NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_coupon_assigned_contacts_coupon_contact (coupon_id, contact_id),
    KEY idx_coupon_assigned_contacts_coupon_id (coupon_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS coupon_included_product_ids (
    id          BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
    coupon_id   VARCHAR(64)      NOT NULL,
    product_id  VARCHAR(64)      NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_coupon_included_products_coupon_product (coupon_id, product_id),
    KEY idx_coupon_included_products_coupon_id (coupon_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS coupon_included_category_ids (
    id           BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
    coupon_id    VARCHAR(64)      NOT NULL,
    category_id  VARCHAR(64)      NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_coupon_included_categories_coupon_category (coupon_id, category_id),
    KEY idx_coupon_included_categories_coupon_id (coupon_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS coupon_included_tag_ids (
    id         BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
    coupon_id  VARCHAR(64)      NOT NULL,
    tag_id     VARCHAR(64)      NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_coupon_included_tags_coupon_tag (coupon_id, tag_id),
    KEY idx_coupon_included_tags_coupon_id (coupon_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS coupon_usages (
    id         BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
    coupon_id  VARCHAR(64)      NOT NULL,
    order_id   VARCHAR(64)      NOT NULL,
    email      VARCHAR(320)     NOT NULL DEFAULT '',
    used_at    DATETIME(3)      NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_coupon_usages_coupon_order (coupon_id, order_id),
    KEY idx_coupon_usages_coupon_id (coupon_id),
    KEY idx_coupon_usages_email (coupon_id, email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS order_applied_coupons (
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

INSERT IGNORE INTO order_applied_coupons (order_id, coupon_id, code, discount_type, discount_amount, applied_at)
SELECT id, '', coupon_code, COALESCE(coupon_discount_type, ''), COALESCE(coupon_discount_amount, 0), NOW(3)
FROM orders
WHERE COALESCE(coupon_code, '') <> '';