-- SQLite does not support DROP COLUMN before 3.35.0; recreate the table without the new columns.
CREATE TABLE campaigns_backup AS SELECT
    id, name, slug, channel, segment_id, subject, html_body, text_body,
    status, total_recipients, sent_count, failed_count, created_at, updated_at
FROM campaigns;

DROP TABLE campaigns;

ALTER TABLE campaigns_backup RENAME TO campaigns;
