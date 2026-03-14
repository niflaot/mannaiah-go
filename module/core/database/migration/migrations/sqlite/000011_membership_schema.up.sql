CREATE TABLE IF NOT EXISTS membership_status (
    contact_id TEXT NOT NULL,
    channel TEXT NOT NULL,
    action TEXT NOT NULL,
    source TEXT NOT NULL,
    occurred_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (contact_id, channel)
);

CREATE INDEX IF NOT EXISTS idx_membership_status_channel_action ON membership_status (channel, action);
CREATE INDEX IF NOT EXISTS idx_membership_status_updated_at ON membership_status (updated_at);

CREATE TABLE IF NOT EXISTS membership_stamps (
    id TEXT PRIMARY KEY,
    contact_id TEXT NOT NULL,
    channel TEXT NOT NULL,
    action TEXT NOT NULL,
    source TEXT NOT NULL,
    occurred_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_membership_stamps_contact_channel_time ON membership_stamps (contact_id, channel, occurred_at, id);
CREATE INDEX IF NOT EXISTS idx_membership_stamps_channel_action ON membership_stamps (channel, action);
