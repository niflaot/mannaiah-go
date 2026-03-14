CREATE TABLE IF NOT EXISTS membership_events (
    contact_id String,
    channel String,
    action String,
    source String,
    occurred_at DateTime64(3)
) ENGINE = MergeTree()
ORDER BY (contact_id, channel, occurred_at);
