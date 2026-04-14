DROP INDEX IF EXISTS idx_storefront_static_pages_archived_at;

ALTER TABLE storefront_static_pages DROP COLUMN archived_at;