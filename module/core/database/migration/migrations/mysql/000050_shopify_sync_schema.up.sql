CREATE TABLE IF NOT EXISTS shopify_sync_links (
    id                VARCHAR(32)   NOT NULL,
    kind              VARCHAR(32)   NOT NULL,
    shopify_id        VARCHAR(128)  NOT NULL,
    mannaiah_id       VARCHAR(64)   NOT NULL,
    last_known_status VARCHAR(32)   NOT NULL DEFAULT '',
    last_synced_at    DATETIME(3)   NULL,
    created_at        DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at        DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY idx_shopify_sync_links_kind_shopify (kind, shopify_id),
    UNIQUE KEY idx_shopify_sync_links_kind_mannaiah (kind, mannaiah_id),
    KEY idx_shopify_sync_links_mannaiah_id (mannaiah_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS shopify_webhook_deliveries (
    delivery_id  VARCHAR(255)  NOT NULL,
    topic        VARCHAR(255)  NOT NULL,
    processed_at DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (delivery_id),
    KEY idx_shopify_webhook_deliveries_topic (topic)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;