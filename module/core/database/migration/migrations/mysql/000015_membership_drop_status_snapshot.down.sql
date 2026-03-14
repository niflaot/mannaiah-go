CREATE TABLE IF NOT EXISTS membership_status (
    contact_id VARCHAR(36) NOT NULL,
    channel VARCHAR(32) NOT NULL,
    action VARCHAR(32) NOT NULL,
    source VARCHAR(64) NOT NULL,
    occurred_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    PRIMARY KEY (contact_id, channel),
    INDEX idx_membership_status_channel_action (channel, action),
    INDEX idx_membership_status_updated_at (updated_at)
) ENGINE=InnoDB;
