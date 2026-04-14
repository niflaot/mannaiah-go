ALTER TABLE storefront_static_pages ADD COLUMN archived_at DATETIME NULL;

CREATE INDEX idx_storefront_static_pages_archived_at
    ON storefront_static_pages (archived_at);