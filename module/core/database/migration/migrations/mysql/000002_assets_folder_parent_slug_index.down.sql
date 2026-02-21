SET @table_exists := (
    SELECT COUNT(1)
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'asset_folders'
);
SET @idx_exists := (
    SELECT COUNT(1)
    FROM information_schema.statistics
    WHERE table_schema = DATABASE()
      AND table_name = 'asset_folders'
      AND index_name = 'idx_asset_folders_slug'
);
SET @create_legacy_idx_sql := IF(
    @table_exists > 0 AND @idx_exists = 0,
    'CREATE UNIQUE INDEX idx_asset_folders_slug ON asset_folders(slug)',
    'SELECT 1'
);
PREPARE create_legacy_idx_stmt FROM @create_legacy_idx_sql;
EXECUTE create_legacy_idx_stmt;
DEALLOCATE PREPARE create_legacy_idx_stmt;
