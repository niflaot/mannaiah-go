ALTER TABLE falabella_sync_status
    ADD COLUMN task VARCHAR(16) NOT NULL DEFAULT 'data' AFTER step;

UPDATE falabella_sync_status
SET task = CASE
    WHEN step = 'image' THEN 'image'
    ELSE 'data'
END;

CREATE INDEX idx_falabella_sync_status_task ON falabella_sync_status(task);
