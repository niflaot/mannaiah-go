CREATE TABLE IF NOT EXISTS shopify_installations (
    id             VARCHAR(32)   NOT NULL,
    shop_domain    VARCHAR(255)  NOT NULL,
    access_token   VARCHAR(255)  NOT NULL,
    scopes         VARCHAR(500)  NOT NULL,
    installed_at   DATETIME(3)   NOT NULL,
    uninstalled_at DATETIME(3)   NULL,
    created_at     DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at     DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY idx_shopify_installations_shop_domain (shop_domain)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE shopify_sync_links
    ADD COLUMN shop_domain VARCHAR(255) NOT NULL DEFAULT '' AFTER kind;

DROP INDEX idx_shopify_sync_links_kind_shopify ON shopify_sync_links;
DROP INDEX idx_shopify_sync_links_kind_mannaiah ON shopify_sync_links;

CREATE UNIQUE INDEX idx_shopify_sync_links_kind_shopify ON shopify_sync_links (kind, shop_domain, shopify_id);
CREATE UNIQUE INDEX idx_shopify_sync_links_kind_mannaiah ON shopify_sync_links (kind, shop_domain, mannaiah_id);
CREATE INDEX idx_shopify_sync_links_shop_domain ON shopify_sync_links (shop_domain);