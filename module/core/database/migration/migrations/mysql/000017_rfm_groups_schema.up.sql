CREATE TABLE IF NOT EXISTS rfm_groups (
    id          VARCHAR(64)  NOT NULL,
    name        VARCHAR(255) NOT NULL,
    slug        VARCHAR(255) NOT NULL,
    description TEXT         NULL,
    created_at  DATETIME(3)  NULL,
    updated_at  DATETIME(3)  NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_rfm_groups_slug (slug)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS rfm_band_configs (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    dimension  VARCHAR(32)     NOT NULL,
    ascending  BOOLEAN         NOT NULL DEFAULT TRUE,
    band5_min  DOUBLE          NOT NULL DEFAULT 0,
    band4_min  DOUBLE          NOT NULL DEFAULT 0,
    band3_min  DOUBLE          NOT NULL DEFAULT 0,
    band2_min  DOUBLE          NOT NULL DEFAULT 0,
    updated_at DATETIME(3)     NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_rfm_band_configs_dimension (dimension)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS rfm_group_conditions (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    group_id    VARCHAR(64)     NOT NULL,
    r_min       INT             NULL,
    r_max       INT             NULL,
    f_min       INT             NULL,
    f_max       INT             NULL,
    m_min       DOUBLE          NULL,
    m_max       DOUBLE          NULL,
    PRIMARY KEY (id),
    KEY idx_rfm_group_conditions_group_id (group_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
