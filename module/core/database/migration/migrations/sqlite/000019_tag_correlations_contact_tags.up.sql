CREATE TABLE IF NOT EXISTS tags (
    id         INTEGER  NOT NULL PRIMARY KEY AUTOINCREMENT,
    name       TEXT     NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_name ON tags (name);
CREATE INDEX IF NOT EXISTS idx_tags_deleted_at ON tags (deleted_at);

CREATE TABLE IF NOT EXISTS tag_correlations (
    id          INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    source_tag  TEXT    NOT NULL,
    target_tag  TEXT    NOT NULL,
    probability REAL    NOT NULL DEFAULT 0.00,
    notes       TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tag_correlations_pair ON tag_correlations (source_tag, target_tag);
CREATE INDEX IF NOT EXISTS idx_tag_correlations_source ON tag_correlations (source_tag);
CREATE INDEX IF NOT EXISTS idx_tag_correlations_target ON tag_correlations (target_tag);
