DROP INDEX IF EXISTS idx_falabella_sync_status_task;

PRAGMA foreign_keys=off;

CREATE TABLE falabella_sync_status_rollback_000007 (
    execution_id TEXT NOT NULL,
    feed_id TEXT NOT NULL,
    product_id TEXT NOT NULL,
    sku TEXT NOT NULL,
    step TEXT NOT NULL,
    action TEXT NOT NULL,
    status TEXT NOT NULL,
    synced_at DATETIME NOT NULL,
    resolved_at DATETIME NULL,
    PRIMARY KEY (feed_id)
);

INSERT INTO falabella_sync_status_rollback_000007 (
    execution_id,
    feed_id,
    product_id,
    sku,
    step,
    action,
    status,
    synced_at,
    resolved_at
)
SELECT
    execution_id,
    feed_id,
    product_id,
    sku,
    step,
    action,
    status,
    synced_at,
    resolved_at
FROM falabella_sync_status;

DROP TABLE falabella_sync_status;

ALTER TABLE falabella_sync_status_rollback_000007 RENAME TO falabella_sync_status;

CREATE INDEX IF NOT EXISTS idx_falabella_sync_status_execution_id ON falabella_sync_status(execution_id);
CREATE INDEX IF NOT EXISTS idx_falabella_sync_status_product_id ON falabella_sync_status(product_id);
CREATE INDEX IF NOT EXISTS idx_falabella_sync_status_status ON falabella_sync_status(status);

PRAGMA foreign_keys=on;
