CREATE TABLE coupons (
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

CREATE TABLE coupon_assigned_emails (
    id         BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
    coupon_id  VARCHAR(64)      NOT NULL,
    email      VARCHAR(320)     NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_coupon_assigned_emails_coupon_email (coupon_id, email),
    KEY idx_coupon_assigned_emails_coupon_id (coupon_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE coupon_assigned_contact_ids (
    id          BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
    coupon_id   VARCHAR(64)      NOT NULL,
    contact_id  VARCHAR(64)      NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_coupon_assigned_contacts_coupon_contact (coupon_id, contact_id),
    KEY idx_coupon_assigned_contacts_coupon_id (coupon_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE coupon_included_product_ids (
    id          BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
    coupon_id   VARCHAR(64)      NOT NULL,
    product_id  VARCHAR(64)      NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_coupon_included_products_coupon_product (coupon_id, product_id),
    KEY idx_coupon_included_products_coupon_id (coupon_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE coupon_included_category_ids (
    id           BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
    coupon_id    VARCHAR(64)      NOT NULL,
    category_id  VARCHAR(64)      NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_coupon_included_categories_coupon_category (coupon_id, category_id),
    KEY idx_coupon_included_categories_coupon_id (coupon_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE coupon_included_tag_ids (
    id         BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
    coupon_id  VARCHAR(64)      NOT NULL,
    tag_id     VARCHAR(64)      NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_coupon_included_tags_coupon_tag (coupon_id, tag_id),
    KEY idx_coupon_included_tags_coupon_id (coupon_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE coupon_usages (
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
