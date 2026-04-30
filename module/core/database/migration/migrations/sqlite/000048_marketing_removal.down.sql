CREATE TABLE IF NOT EXISTS campaigns (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    channel TEXT NOT NULL,
    segment_id TEXT NOT NULL,
    subject TEXT NOT NULL,
    html_body TEXT NOT NULL,
    text_body TEXT NOT NULL,
    status TEXT NOT NULL,
    total_recipients INTEGER NOT NULL DEFAULT 0,
    sent_count INTEGER NOT NULL DEFAULT 0,
    failed_count INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    template_vars TEXT NOT NULL DEFAULT '',
    product_blocks TEXT NOT NULL DEFAULT ''
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_campaigns_slug ON campaigns (slug);
CREATE INDEX IF NOT EXISTS idx_campaigns_status_created_at ON campaigns (status, created_at);
CREATE INDEX IF NOT EXISTS idx_campaigns_segment_id ON campaigns (segment_id);

CREATE TABLE IF NOT EXISTS segments (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    channel TEXT NOT NULL,
    parent_segment_id TEXT NULL DEFAULT NULL,
    filters_json TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_segments_slug ON segments (slug);
CREATE INDEX IF NOT EXISTS idx_segments_channel_created_at ON segments (channel, created_at);
CREATE INDEX IF NOT EXISTS idx_segments_parent_segment_id ON segments (parent_segment_id);

CREATE TABLE IF NOT EXISTS rfm_groups (
    id TEXT NOT NULL PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    description TEXT NULL,
    created_at DATETIME NULL,
    updated_at DATETIME NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_rfm_groups_slug ON rfm_groups (slug);

CREATE TABLE IF NOT EXISTS rfm_band_configs (
    id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    dimension TEXT NOT NULL,
    ascending BOOLEAN NOT NULL DEFAULT 1,
    band5_min REAL NOT NULL DEFAULT 0,
    band4_min REAL NOT NULL DEFAULT 0,
    band3_min REAL NOT NULL DEFAULT 0,
    band2_min REAL NOT NULL DEFAULT 0,
    updated_at DATETIME NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_rfm_band_configs_dimension ON rfm_band_configs (dimension);

CREATE TABLE IF NOT EXISTS rfm_group_conditions (
    id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    group_id TEXT NOT NULL,
    r_min INTEGER NULL,
    r_max INTEGER NULL,
    f_min INTEGER NULL,
    f_max INTEGER NULL,
    m_min REAL NULL,
    m_max REAL NULL
);

CREATE INDEX IF NOT EXISTS idx_rfm_group_conditions_group_id ON rfm_group_conditions (group_id);

CREATE TABLE IF NOT EXISTS coupons (
    id                     TEXT             NOT NULL,
    code                   TEXT             NOT NULL,
    origin                 TEXT             NOT NULL DEFAULT '',
    discount_type          TEXT             NOT NULL,
    discount_amount        REAL             NOT NULL DEFAULT 0,
    max_usages_global      INTEGER          NULL,
    max_usages_per_email   INTEGER          NULL,
    active                 INTEGER          NOT NULL DEFAULT 1,
    expires_at             DATETIME         NULL,
    woocommerce_id         INTEGER          NULL,
    created_at             DATETIME         NULL,
    updated_at             DATETIME         NULL,
    deleted_at             DATETIME         NULL,
    PRIMARY KEY (id),
    UNIQUE (code)
);

CREATE TABLE IF NOT EXISTS coupon_assigned_emails (
    id         INTEGER  NOT NULL PRIMARY KEY AUTOINCREMENT,
    coupon_id  TEXT     NOT NULL,
    email      TEXT     NOT NULL,
    UNIQUE (coupon_id, email)
);

CREATE TABLE IF NOT EXISTS coupon_assigned_contact_ids (
    id          INTEGER  NOT NULL PRIMARY KEY AUTOINCREMENT,
    coupon_id   TEXT     NOT NULL,
    contact_id  TEXT     NOT NULL,
    UNIQUE (coupon_id, contact_id)
);

CREATE TABLE IF NOT EXISTS coupon_included_product_ids (
    id          INTEGER  NOT NULL PRIMARY KEY AUTOINCREMENT,
    coupon_id   TEXT     NOT NULL,
    product_id  TEXT     NOT NULL,
    UNIQUE (coupon_id, product_id)
);

CREATE TABLE IF NOT EXISTS coupon_included_category_ids (
    id           INTEGER  NOT NULL PRIMARY KEY AUTOINCREMENT,
    coupon_id    TEXT     NOT NULL,
    category_id  TEXT     NOT NULL,
    UNIQUE (coupon_id, category_id)
);

CREATE TABLE IF NOT EXISTS coupon_included_tag_ids (
    id         INTEGER  NOT NULL PRIMARY KEY AUTOINCREMENT,
    coupon_id  TEXT     NOT NULL,
    tag_id     TEXT     NOT NULL,
    UNIQUE (coupon_id, tag_id)
);

CREATE TABLE IF NOT EXISTS coupon_usages (
    id         INTEGER  NOT NULL PRIMARY KEY AUTOINCREMENT,
    coupon_id  TEXT     NOT NULL,
    order_id   TEXT     NOT NULL,
    email      TEXT     NOT NULL DEFAULT '',
    used_at    DATETIME NOT NULL,
    UNIQUE (coupon_id, order_id)
);

CREATE TABLE IF NOT EXISTS order_applied_coupons (
    id               INTEGER  NOT NULL PRIMARY KEY AUTOINCREMENT,
    order_id         TEXT     NOT NULL,
    coupon_id        TEXT     NOT NULL DEFAULT '',
    code             TEXT     NOT NULL,
    discount_type    TEXT     NOT NULL,
    discount_amount  REAL     NOT NULL DEFAULT 0,
    applied_at       DATETIME NOT NULL,
    UNIQUE (order_id, code)
);

INSERT INTO order_applied_coupons (order_id, coupon_id, code, discount_type, discount_amount, applied_at)
SELECT id, '', coupon_code, COALESCE(coupon_discount_type, ''), COALESCE(coupon_discount_amount, 0), CURRENT_TIMESTAMP
FROM orders
WHERE COALESCE(coupon_code, '') <> '';

CREATE TABLE storefront_renderables (
    id                          TEXT      NOT NULL PRIMARY KEY,
    kind                        TEXT      NOT NULL,
    metadata_json               TEXT      NOT NULL,
    content_json                TEXT      NOT NULL,
    snapshot_hash               TEXT      NOT NULL,
    draft                       INTEGER   NOT NULL DEFAULT 1,
    latest_published_version_id TEXT      NULL,
    latest_published_at         DATETIME  NULL,
    created_at                  DATETIME  NOT NULL,
    updated_at                  DATETIME  NOT NULL
);

CREATE INDEX idx_storefront_renderables_kind_draft
    ON storefront_renderables (kind, draft);
CREATE INDEX idx_storefront_renderables_latest_published_at
    ON storefront_renderables (latest_published_at);

CREATE TABLE storefront_renderable_versions (
    id                TEXT      NOT NULL PRIMARY KEY,
    renderable_id     TEXT      NOT NULL,
    source_version_id TEXT      NULL,
    metadata_json     TEXT      NOT NULL,
    content_json      TEXT      NOT NULL,
    snapshot_hash     TEXT      NOT NULL,
    published_at      DATETIME  NOT NULL,
    created_at        DATETIME  NOT NULL,
    FOREIGN KEY (renderable_id) REFERENCES storefront_renderables(id) ON DELETE CASCADE,
    FOREIGN KEY (source_version_id) REFERENCES storefront_renderable_versions(id) ON DELETE SET NULL
);

CREATE INDEX idx_storefront_renderable_versions_renderable_published
    ON storefront_renderable_versions (renderable_id, published_at);
CREATE INDEX idx_storefront_renderable_versions_renderable_hash
    ON storefront_renderable_versions (renderable_id, snapshot_hash);

CREATE TABLE storefront_static_pages (
    id             TEXT      NOT NULL PRIMARY KEY,
    renderable_id  TEXT      NOT NULL UNIQUE,
    title          TEXT      NOT NULL,
    url            TEXT      NOT NULL UNIQUE,
    seo_tags_json  TEXT      NOT NULL,
    archived_at    DATETIME  NULL,
    created_at     DATETIME  NOT NULL,
    updated_at     DATETIME  NOT NULL,
    FOREIGN KEY (renderable_id) REFERENCES storefront_renderables(id) ON DELETE CASCADE
);

CREATE INDEX idx_storefront_static_pages_title
    ON storefront_static_pages (title);
CREATE INDEX idx_storefront_static_pages_archived_at
    ON storefront_static_pages (archived_at);

ALTER TABLE orders DROP COLUMN coupon_discount_type;
ALTER TABLE orders DROP COLUMN coupon_discount_amount;
ALTER TABLE orders DROP COLUMN coupon_code;
