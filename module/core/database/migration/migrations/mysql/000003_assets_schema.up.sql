CREATE TABLE IF NOT EXISTS asset_folders (
    id VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(191) NOT NULL,
    parent_folder_id VARCHAR(64) NULL,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_asset_folders_parent_slug (parent_folder_id, slug),
    KEY idx_asset_folders_parent_folder_id (parent_folder_id),
    KEY idx_asset_folders_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS assets (
    id VARCHAR(64) NOT NULL,
    `key` VARCHAR(512) NOT NULL,
    name VARCHAR(255) NOT NULL,
    original_name VARCHAR(255) NOT NULL,
    folder_id VARCHAR(64) NULL,
    mime_type VARCHAR(255) NOT NULL,
    size BIGINT NOT NULL,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_assets_key (`key`),
    KEY idx_assets_folder_id (folder_id),
    KEY idx_assets_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS asset_tags (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    asset_id VARCHAR(64) NOT NULL,
    name VARCHAR(64) NOT NULL,
    color VARCHAR(7) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_asset_tags_asset_name (asset_id, name),
    KEY idx_asset_tags_asset_id (asset_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS asset_metadata (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    asset_id VARCHAR(64) NOT NULL,
    `key` VARCHAR(128) NOT NULL,
    value TEXT NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_asset_metadata_asset_key (asset_id, `key`),
    KEY idx_asset_metadata_asset_id (asset_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS folder_tags (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    folder_id VARCHAR(64) NOT NULL,
    name VARCHAR(64) NOT NULL,
    color VARCHAR(7) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_folder_tags_folder_name (folder_id, name),
    KEY idx_folder_tags_folder_id (folder_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
