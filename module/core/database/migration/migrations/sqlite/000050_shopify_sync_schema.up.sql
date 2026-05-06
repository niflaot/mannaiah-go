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

CREATE INDEX IF NOT EXISTS idx_shopify_sync_links_mannaiah_id ON shopify_sync_links (mannaiah_id);

CREATE TABLE IF NOT EXISTS shopify_webhook_deliveries (
    delivery_id  TEXT      NOT NULL,
    topic        TEXT      NOT NULL,
    processed_at DATETIME  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (delivery_id)
);

CREATE INDEX IF NOT EXISTS idx_shopify_webhook_deliveries_topic ON shopify_webhook_deliveries (topic);