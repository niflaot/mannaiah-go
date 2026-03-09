DROP INDEX idx_falabella_sync_status_task ON falabella_sync_status;

ALTER TABLE falabella_sync_status
    DROP COLUMN task;
