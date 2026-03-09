CREATE TABLE IF NOT EXISTS falabella_sync_status_variation (
    feed_id TEXT NOT NULL,
    variation_id TEXT NOT NULL,
    PRIMARY KEY (feed_id, variation_id)
);

CREATE INDEX IF NOT EXISTS idx_falabella_sync_status_variation_feed_id ON falabella_sync_status_variation(feed_id);
CREATE INDEX IF NOT EXISTS idx_falabella_sync_status_variation_variation_id ON falabella_sync_status_variation(variation_id);
