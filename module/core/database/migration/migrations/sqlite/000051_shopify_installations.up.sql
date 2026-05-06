ALTER TABLE shopify_sync_links RENAME TO shopify_sync_links_old;

CREATE TABLE IF NOT EXISTS shopify_sync_links (
    id                TEXT      NOT NULL,
    kind              TEXT      NOT NULL,
    shop_domain       TEXT      NOT NULL DEFAULT '',
    shopify_id        TEXT      NOT NULL,
    mannaiah_id       TEXT      NOT NULL,
    last_known_status TEXT      NOT NULL DEFAULT '',
    last_synced_at    DATETIME  NULL,
    created_at        DATETIME  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE (kind, shop_domain, shopify_id),
    UNIQUE (kind, shop_domain, mannaiah_id)
);

INSERT INTO shopify_sync_links (
    id,
    kind,
    shop_domain,
    shopify_id,
    mannaiah_id,
    last_known_status,
    last_synced_at,
    created_at,
    updated_at
)
SELECT
    id,
    kind,
    '',
    shopify_id,
    mannaiah_id,
    last_known_status,
    last_synced_at,
    created_at,
    updated_at
FROM shopify_sync_links_old;

DROP TABLE shopify_sync_links_old;

CREATE INDEX IF NOT EXISTS idx_shopify_sync_links_mannaiah_id ON shopify_sync_links (mannaiah_id);
CREATE INDEX IF NOT EXISTS idx_shopify_sync_links_shop_domain ON shopify_sync_links (shop_domain);

CREATE TABLE IF NOT EXISTS shopify_installations (
    id             TEXT      NOT NULL,
    shop_domain    TEXT      NOT NULL,
    access_token   TEXT      NOT NULL,
    scopes         TEXT      NOT NULL,
    installed_at   DATETIME  NOT NULL,
    uninstalled_at DATETIME  NULL,
    created_at     DATETIME  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     DATETIME  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE (shop_domain)
);