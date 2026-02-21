DROP INDEX IF EXISTS idx_asset_folders_parent_slug;
CREATE UNIQUE INDEX IF NOT EXISTS idx_asset_folders_slug ON asset_folders(slug);
