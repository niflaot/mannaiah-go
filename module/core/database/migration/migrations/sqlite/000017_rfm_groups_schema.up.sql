CREATE TABLE IF NOT EXISTS rfm_groups (
    id          TEXT    NOT NULL PRIMARY KEY,
    name        TEXT    NOT NULL,
    slug        TEXT    NOT NULL,
    description TEXT    NULL,
    created_at  DATETIME NULL,
    updated_at  DATETIME NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_rfm_groups_slug ON rfm_groups (slug);

CREATE TABLE IF NOT EXISTS rfm_band_configs (
    id         INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    dimension  TEXT    NOT NULL,
    ascending  BOOLEAN NOT NULL DEFAULT 1,
    band5_min  REAL    NOT NULL DEFAULT 0,
    band4_min  REAL    NOT NULL DEFAULT 0,
    band3_min  REAL    NOT NULL DEFAULT 0,
    band2_min  REAL    NOT NULL DEFAULT 0,
    updated_at DATETIME NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_rfm_band_configs_dimension ON rfm_band_configs (dimension);

CREATE TABLE IF NOT EXISTS rfm_group_conditions (
    id       INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    group_id TEXT    NOT NULL,
    r_min    INTEGER NULL,
    r_max    INTEGER NULL,
    f_min    INTEGER NULL,
    f_max    INTEGER NULL,
    m_min    REAL    NULL,
    m_max    REAL    NULL
);

CREATE INDEX IF NOT EXISTS idx_rfm_group_conditions_group_id ON rfm_group_conditions (group_id);
