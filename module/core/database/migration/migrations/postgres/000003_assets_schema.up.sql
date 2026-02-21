CREATE TABLE IF NOT EXISTS asset_folders (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(191) NOT NULL,
    parent_folder_id VARCHAR(64),
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_asset_folders_parent_slug ON asset_folders(parent_folder_id, slug);
CREATE INDEX IF NOT EXISTS idx_asset_folders_parent_folder_id ON asset_folders(parent_folder_id);
CREATE INDEX IF NOT EXISTS idx_asset_folders_deleted_at ON asset_folders(deleted_at);

CREATE TABLE IF NOT EXISTS assets (
    id VARCHAR(64) PRIMARY KEY,
    key VARCHAR(512) NOT NULL,
    name VARCHAR(255) NOT NULL,
    original_name VARCHAR(255) NOT NULL,
    folder_id VARCHAR(64),
    mime_type VARCHAR(255) NOT NULL,
    size BIGINT NOT NULL,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_assets_key ON assets(key);
CREATE INDEX IF NOT EXISTS idx_assets_folder_id ON assets(folder_id);
CREATE INDEX IF NOT EXISTS idx_assets_deleted_at ON assets(deleted_at);

CREATE TABLE IF NOT EXISTS asset_tags (
    id BIGSERIAL PRIMARY KEY,
    asset_id VARCHAR(64) NOT NULL,
    name VARCHAR(64) NOT NULL,
    color VARCHAR(7) NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_asset_tags_asset_name ON asset_tags(asset_id, name);
CREATE INDEX IF NOT EXISTS idx_asset_tags_asset_id ON asset_tags(asset_id);

CREATE TABLE IF NOT EXISTS asset_metadata (
    id BIGSERIAL PRIMARY KEY,
    asset_id VARCHAR(64) NOT NULL,
    key VARCHAR(128) NOT NULL,
    value TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_asset_metadata_asset_key ON asset_metadata(asset_id, key);
CREATE INDEX IF NOT EXISTS idx_asset_metadata_asset_id ON asset_metadata(asset_id);

CREATE TABLE IF NOT EXISTS folder_tags (
    id BIGSERIAL PRIMARY KEY,
    folder_id VARCHAR(64) NOT NULL,
    name VARCHAR(64) NOT NULL,
    color VARCHAR(7) NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_folder_tags_folder_name ON folder_tags(folder_id, name);
CREATE INDEX IF NOT EXISTS idx_folder_tags_folder_id ON folder_tags(folder_id);
