CREATE TABLE coupons (
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

CREATE TABLE coupon_assigned_emails (
    id         INTEGER  NOT NULL PRIMARY KEY AUTOINCREMENT,
    coupon_id  TEXT     NOT NULL,
    email      TEXT     NOT NULL,
    UNIQUE (coupon_id, email)
);

CREATE TABLE coupon_assigned_contact_ids (
    id          INTEGER  NOT NULL PRIMARY KEY AUTOINCREMENT,
    coupon_id   TEXT     NOT NULL,
    contact_id  TEXT     NOT NULL,
    UNIQUE (coupon_id, contact_id)
);

CREATE TABLE coupon_included_product_ids (
    id          INTEGER  NOT NULL PRIMARY KEY AUTOINCREMENT,
    coupon_id   TEXT     NOT NULL,
    product_id  TEXT     NOT NULL,
    UNIQUE (coupon_id, product_id)
);

CREATE TABLE coupon_included_category_ids (
    id           INTEGER  NOT NULL PRIMARY KEY AUTOINCREMENT,
    coupon_id    TEXT     NOT NULL,
    category_id  TEXT     NOT NULL,
    UNIQUE (coupon_id, category_id)
);

CREATE TABLE coupon_included_tag_ids (
    id         INTEGER  NOT NULL PRIMARY KEY AUTOINCREMENT,
    coupon_id  TEXT     NOT NULL,
    tag_id     TEXT     NOT NULL,
    UNIQUE (coupon_id, tag_id)
);

CREATE TABLE coupon_usages (
    id         INTEGER  NOT NULL PRIMARY KEY AUTOINCREMENT,
    coupon_id  TEXT     NOT NULL,
    order_id   TEXT     NOT NULL,
    email      TEXT     NOT NULL DEFAULT '',
    used_at    DATETIME NOT NULL,
    UNIQUE (coupon_id, order_id)
);
