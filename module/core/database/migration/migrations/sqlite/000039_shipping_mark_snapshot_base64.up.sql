-- SQLite already stores draft_snapshot as TEXT.
ALTER TABLE shipping_marks ADD COLUMN response_snapshot TEXT NULL;
