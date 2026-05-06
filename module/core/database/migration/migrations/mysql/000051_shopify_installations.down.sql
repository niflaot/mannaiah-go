DROP TABLE IF EXISTS shopify_installations;

DROP INDEX idx_shopify_sync_links_shop_domain ON shopify_sync_links;
DROP INDEX idx_shopify_sync_links_kind_shopify ON shopify_sync_links;
DROP INDEX idx_shopify_sync_links_kind_mannaiah ON shopify_sync_links;

ALTER TABLE shopify_sync_links
    DROP COLUMN shop_domain;

CREATE UNIQUE INDEX idx_shopify_sync_links_kind_shopify ON shopify_sync_links (kind, shopify_id);
CREATE UNIQUE INDEX idx_shopify_sync_links_kind_mannaiah ON shopify_sync_links (kind, mannaiah_id);