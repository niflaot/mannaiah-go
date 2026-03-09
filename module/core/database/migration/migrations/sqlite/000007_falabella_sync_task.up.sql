ALTER TABLE falabella_sync_status
    ADD COLUMN task TEXT NOT NULL DEFAULT 'data';

UPDATE falabella_sync_status
SET task = CASE
    WHEN step = 'image' THEN 'image'
    ELSE 'data'
END;

CREATE INDEX IF NOT EXISTS idx_falabella_sync_status_task ON falabella_sync_status(task);
