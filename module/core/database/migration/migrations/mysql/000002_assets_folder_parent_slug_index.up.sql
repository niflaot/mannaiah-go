SET @drop_legacy_idx_sql = (
    SELECT IF(
        COUNT(1) > 0,
        'ALTER TABLE asset_folders DROP INDEX idx_asset_folders_slug',
        'SELECT 1'
    )
    FROM information_schema.statistics
    WHERE table_schema = DATABASE()
      AND table_name = 'asset_folders'
      AND index_name = 'idx_asset_folders_slug'
);
PREPARE drop_legacy_idx_stmt FROM @drop_legacy_idx_sql;
EXECUTE drop_legacy_idx_stmt;
DEALLOCATE PREPARE drop_legacy_idx_stmt;
