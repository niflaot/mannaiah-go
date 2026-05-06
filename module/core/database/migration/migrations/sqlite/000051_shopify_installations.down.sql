DROP TABLE IF EXISTS shopify_installations;

ALTER TABLE shopify_sync_links RENAME TO shopify_sync_links_new;

CREATE TABLE IF NOT EXISTS shopify_sync_links (
    id                TEXT      NOT NULL,
    kind              TEXT      NOT NULL,
    shopify_id        TEXT      NOT NULL,
    mannaiah_id       TEXT      NOT NULL,
    last_known_status TEXT      NOT NULL DEFAULT '',
    last_synced_at    DATETIME  NULL,
    created_at        DATETIME  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE (kind, shopify_id),
    UNIQUE (kind, mannaiah_id)
);

INSERT INTO shopify_sync_links (
    id,
    kind,
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
    shopify_id,
    mannaiah_id,
    last_known_status,
    last_synced_at,
    created_at,
    updated_at
FROM shopify_sync_links_new;

DROP TABLE shopify_sync_links_new;

CREATE INDEX IF NOT EXISTS idx_shopify_sync_links_mannaiah_id ON shopify_sync_links (mannaiah_id);