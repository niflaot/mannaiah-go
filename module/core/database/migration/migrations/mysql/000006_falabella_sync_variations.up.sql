CREATE TABLE IF NOT EXISTS falabella_sync_status_variation (
    feed_id VARCHAR(191) NOT NULL,
    variation_id VARCHAR(128) NOT NULL,
    PRIMARY KEY (feed_id, variation_id),
    KEY idx_falabella_sync_status_variation_feed_id (feed_id),
    KEY idx_falabella_sync_status_variation_variation_id (variation_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
