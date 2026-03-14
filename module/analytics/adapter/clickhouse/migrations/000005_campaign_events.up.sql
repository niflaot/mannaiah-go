CREATE TABLE IF NOT EXISTS campaign_events (
    campaign_id String,
    contact_id String,
    channel String,
    status String,
    template_version UInt32,
    occurred_at DateTime64(3)
) ENGINE = MergeTree()
ORDER BY (campaign_id, contact_id, occurred_at);
