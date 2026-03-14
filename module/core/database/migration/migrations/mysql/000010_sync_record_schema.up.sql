CREATE TABLE IF NOT EXISTS sync_runs (
    id VARCHAR(36) PRIMARY KEY,
    kind VARCHAR(100) NOT NULL,
    sync_trigger VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL,
    started_at DATETIME(3) NOT NULL,
    ended_at DATETIME(3) NULL,
    duration_ms BIGINT NOT NULL DEFAULT 0,
    processed_count INT NOT NULL DEFAULT 0,
    succeeded_count INT NOT NULL DEFAULT 0,
    failed_count INT NOT NULL DEFAULT 0,
    skipped_count INT NOT NULL DEFAULT 0,
    error_count INT NOT NULL DEFAULT 0,
    metadata_json TEXT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    INDEX idx_sync_runs_kind_started (kind, started_at),
    INDEX idx_sync_runs_status_started (status, started_at),
    INDEX idx_sync_runs_started_at (started_at)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS sync_run_errors (
    id VARCHAR(36) PRIMARY KEY,
    run_id VARCHAR(36) NOT NULL,
    error_type VARCHAR(64) NOT NULL,
    error_code VARCHAR(128) NULL,
    message TEXT NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_sync_run_errors_run FOREIGN KEY (run_id) REFERENCES sync_runs (id) ON DELETE CASCADE,
    INDEX idx_sync_run_errors_run_id (run_id),
    INDEX idx_sync_run_errors_created_at (created_at)
) ENGINE=InnoDB;
