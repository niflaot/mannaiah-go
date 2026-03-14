CREATE TABLE IF NOT EXISTS sync_runs (
    id TEXT PRIMARY KEY,
    kind TEXT NOT NULL,
    trigger TEXT NOT NULL,
    status TEXT NOT NULL,
    started_at DATETIME NOT NULL,
    ended_at DATETIME NULL,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    processed_count INTEGER NOT NULL DEFAULT 0,
    succeeded_count INTEGER NOT NULL DEFAULT 0,
    failed_count INTEGER NOT NULL DEFAULT 0,
    skipped_count INTEGER NOT NULL DEFAULT 0,
    error_count INTEGER NOT NULL DEFAULT 0,
    metadata_json TEXT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sync_runs_kind_started ON sync_runs (kind, started_at);
CREATE INDEX IF NOT EXISTS idx_sync_runs_status_started ON sync_runs (status, started_at);
CREATE INDEX IF NOT EXISTS idx_sync_runs_started_at ON sync_runs (started_at);

CREATE TABLE IF NOT EXISTS sync_run_errors (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL,
    error_type TEXT NOT NULL,
    error_code TEXT NULL,
    message TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (run_id) REFERENCES sync_runs (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_sync_run_errors_run_id ON sync_run_errors (run_id);
CREATE INDEX IF NOT EXISTS idx_sync_run_errors_created_at ON sync_run_errors (created_at);
