CREATE TABLE IF NOT EXISTS membership_status (
    contact_id VARCHAR(36) NOT NULL,
    channel VARCHAR(32) NOT NULL,
    action VARCHAR(16) NOT NULL,
    source VARCHAR(64) NOT NULL,
    occurred_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (contact_id, channel),
    INDEX idx_membership_status_channel_action (channel, action),
    INDEX idx_membership_status_updated_at (updated_at)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS membership_stamps (
    id VARCHAR(36) PRIMARY KEY,
    contact_id VARCHAR(36) NOT NULL,
    channel VARCHAR(32) NOT NULL,
    action VARCHAR(16) NOT NULL,
    source VARCHAR(64) NOT NULL,
    occurred_at DATETIME(3) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_membership_stamps_contact_channel_time (contact_id, channel, occurred_at, id),
    INDEX idx_membership_stamps_channel_action (channel, action)
) ENGINE=InnoDB;
