ALTER TABLE storefront_static_pages
    ADD COLUMN archived_at DATETIME(3) NULL AFTER seo_tags_json;

ALTER TABLE storefront_static_pages
    ADD KEY idx_storefront_static_pages_archived_at (archived_at);