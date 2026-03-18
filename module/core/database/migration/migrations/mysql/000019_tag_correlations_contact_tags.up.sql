-- tags: canonical registry of all product taxonomy tags.
-- Acts as the single source of truth for tag names.
-- Tags are soft-deleted; deletion cascades to product_tags at the application layer.
CREATE TABLE IF NOT EXISTS tags (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name       VARCHAR(128)    NOT NULL,
    created_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3)     NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_tags_name (name),
    KEY idx_tags_deleted_at (deleted_at)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;

-- tag_correlations: manually configured cross-sell probability map between product tags.
-- Managed exclusively by marketing:manage permission owners.
CREATE TABLE IF NOT EXISTS tag_correlations (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    source_tag  VARCHAR(128)    NOT NULL,
    target_tag  VARCHAR(128)    NOT NULL,
    probability DECIMAL(5, 2)   NOT NULL DEFAULT 0.00 COMMENT '0.00–100.00 purchase probability',
    notes       TEXT,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY idx_tag_correlations_pair (source_tag, target_tag),
    KEY idx_tag_correlations_source (source_tag),
    KEY idx_tag_correlations_target (target_tag)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;
