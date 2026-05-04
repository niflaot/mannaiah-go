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

INSERT OR IGNORE INTO order_applied_coupons (order_id, coupon_id, code, discount_type, discount_amount, applied_at)
SELECT id, '', coupon_code, COALESCE(coupon_discount_type, ''), COALESCE(coupon_discount_amount, 0), CURRENT_TIMESTAMP
FROM orders
WHERE COALESCE(coupon_code, '') <> '';