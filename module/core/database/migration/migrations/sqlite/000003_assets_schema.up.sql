CREATE TABLE IF NOT EXISTS asset_folders (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    parent_folder_id TEXT,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_asset_folders_parent_slug ON asset_folders(parent_folder_id, slug);
CREATE INDEX IF NOT EXISTS idx_asset_folders_parent_folder_id ON asset_folders(parent_folder_id);
CREATE INDEX IF NOT EXISTS idx_asset_folders_deleted_at ON asset_folders(deleted_at);

CREATE TABLE IF NOT EXISTS assets (
    id TEXT PRIMARY KEY,
    key TEXT NOT NULL,
    name TEXT NOT NULL,
    original_name TEXT NOT NULL,
    folder_id TEXT,
    mime_type TEXT NOT NULL,
    size INTEGER NOT NULL,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_assets_key ON assets(key);
CREATE INDEX IF NOT EXISTS idx_assets_folder_id ON assets(folder_id);
CREATE INDEX IF NOT EXISTS idx_assets_deleted_at ON assets(deleted_at);

CREATE TABLE IF NOT EXISTS asset_tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    asset_id TEXT NOT NULL,
    name TEXT NOT NULL,
    color TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_asset_tags_asset_name ON asset_tags(asset_id, name);
CREATE INDEX IF NOT EXISTS idx_asset_tags_asset_id ON asset_tags(asset_id);

CREATE TABLE IF NOT EXISTS asset_metadata (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    asset_id TEXT NOT NULL,
    key TEXT NOT NULL,
    value TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_asset_metadata_asset_key ON asset_metadata(asset_id, key);
CREATE INDEX IF NOT EXISTS idx_asset_metadata_asset_id ON asset_metadata(asset_id);

CREATE TABLE IF NOT EXISTS folder_tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    folder_id TEXT NOT NULL,
    name TEXT NOT NULL,
    color TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_folder_tags_folder_name ON folder_tags(folder_id, name);
CREATE INDEX IF NOT EXISTS idx_folder_tags_folder_id ON folder_tags(folder_id);
