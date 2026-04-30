CREATE TABLE IF NOT EXISTS campaigns (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(191) NOT NULL,
    channel VARCHAR(32) NOT NULL,
    segment_id VARCHAR(36) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    html_body LONGTEXT NOT NULL,
    text_body LONGTEXT NOT NULL,
    status VARCHAR(32) NOT NULL,
    total_recipients INT NOT NULL DEFAULT 0,
    sent_count INT NOT NULL DEFAULT 0,
    failed_count INT NOT NULL DEFAULT 0,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    template_vars JSON NOT NULL DEFAULT (JSON_OBJECT()),
    product_blocks JSON NOT NULL DEFAULT (JSON_ARRAY()),
    UNIQUE KEY uq_campaigns_slug (slug),
    INDEX idx_campaigns_status_created_at (status, created_at),
    INDEX idx_campaigns_segment_id (segment_id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS segments (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(191) NOT NULL,
    channel VARCHAR(32) NOT NULL,
    parent_segment_id VARCHAR(36) NULL DEFAULT NULL,
    filters_json LONGTEXT NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_segments_slug (slug),
    INDEX idx_segments_channel_created_at (channel, created_at),
    INDEX idx_segments_parent_segment_id (parent_segment_id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS rfm_groups (
    id VARCHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(191) NOT NULL,
    description TEXT NULL,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_rfm_groups_slug (slug)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS rfm_band_configs (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    dimension VARCHAR(32) NOT NULL,
    ascending TINYINT(1) NOT NULL DEFAULT 1,
    band5_min DOUBLE NOT NULL DEFAULT 0,
    band4_min DOUBLE NOT NULL DEFAULT 0,
    band3_min DOUBLE NOT NULL DEFAULT 0,
    band2_min DOUBLE NOT NULL DEFAULT 0,
    updated_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_rfm_band_configs_dimension (dimension)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS rfm_group_conditions (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    group_id VARCHAR(36) NOT NULL,
    r_min INT NULL,
    r_max INT NULL,
    f_min INT NULL,
    f_max INT NULL,
    m_min DOUBLE NULL,
    m_max DOUBLE NULL,
    PRIMARY KEY (id),
    KEY idx_rfm_group_conditions_group_id (group_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

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

INSERT INTO order_applied_coupons (order_id, coupon_id, code, discount_type, discount_amount, applied_at)
SELECT id, '', coupon_code, COALESCE(coupon_discount_type, ''), COALESCE(coupon_discount_amount, 0), NOW(3)
FROM orders
WHERE COALESCE(coupon_code, '') <> '';

CREATE TABLE IF NOT EXISTS storefront_renderables (
    id                          VARCHAR(64)   NOT NULL,
    kind                        VARCHAR(64)   NOT NULL,
    metadata_json               LONGTEXT      NOT NULL,
    content_json                LONGTEXT      NOT NULL,
    snapshot_hash               CHAR(64)      NOT NULL,
    draft                       TINYINT(1)    NOT NULL DEFAULT 1,
    latest_published_version_id VARCHAR(64)   NULL,
    latest_published_at         DATETIME(3)   NULL,
    created_at                  DATETIME(3)   NOT NULL,
    updated_at                  DATETIME(3)   NOT NULL,
    PRIMARY KEY (id),
    KEY idx_storefront_renderables_kind_draft (kind, draft),
    KEY idx_storefront_renderables_latest_published_at (latest_published_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS storefront_renderable_versions (
    id                VARCHAR(64)   NOT NULL,
    renderable_id     VARCHAR(64)   NOT NULL,
    source_version_id VARCHAR(64)   NULL,
    metadata_json     LONGTEXT      NOT NULL,
    content_json      LONGTEXT      NOT NULL,
    snapshot_hash     CHAR(64)      NOT NULL,
    published_at      DATETIME(3)   NOT NULL,
    created_at        DATETIME(3)   NOT NULL,
    PRIMARY KEY (id),
    KEY idx_storefront_renderable_versions_renderable_published (renderable_id, published_at),
    KEY idx_storefront_renderable_versions_renderable_hash (renderable_id, snapshot_hash),
    CONSTRAINT fk_storefront_renderable_versions_renderable
        FOREIGN KEY (renderable_id) REFERENCES storefront_renderables(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_storefront_renderable_versions_source
        FOREIGN KEY (source_version_id) REFERENCES storefront_renderable_versions(id)
        ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS storefront_static_pages (
    id             VARCHAR(64)   NOT NULL,
    renderable_id  VARCHAR(64)   NOT NULL,
    title          VARCHAR(255)  NOT NULL,
    url            VARCHAR(512)  NOT NULL,
    seo_tags_json  LONGTEXT      NOT NULL,
    archived_at    DATETIME(3)   NULL,
    created_at     DATETIME(3)   NOT NULL,
    updated_at     DATETIME(3)   NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_storefront_static_pages_renderable_id (renderable_id),
    UNIQUE KEY uk_storefront_static_pages_url (url),
    KEY idx_storefront_static_pages_title (title),
    KEY idx_storefront_static_pages_archived_at (archived_at),
    CONSTRAINT fk_storefront_static_pages_renderable
        FOREIGN KEY (renderable_id) REFERENCES storefront_renderables(id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE orders
    DROP COLUMN coupon_discount_type,
    DROP COLUMN coupon_discount_amount,
    DROP COLUMN coupon_code;
