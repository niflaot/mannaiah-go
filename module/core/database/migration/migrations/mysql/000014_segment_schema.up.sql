CREATE TABLE IF NOT EXISTS segments (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(191) NOT NULL,
    channel VARCHAR(32) NOT NULL,
    filters_json LONGTEXT NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_segments_slug (slug),
    INDEX idx_segments_channel_created_at (channel, created_at)
) ENGINE=InnoDB;
