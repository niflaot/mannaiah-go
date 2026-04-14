CREATE TABLE storefront_renderables (
    id                          VARCHAR(64)   NOT NULL,
    kind                        VARCHAR(64)   NOT NULL,
    metadata_json               LONGTEXT      NOT NULL,
    content_json                LONGTEXT      NOT NULL,
    snapshot_hash               CHAR(64)      NOT NULL,
    draft                       TINYINT(1)    NOT NULL DEFAULT 1,
    latest_published_version_id VARCHAR(64)   NULL,
    latest_published_at         DATETIME(3)   NULL,
    created_at                  DATETIME(3)   NOT NULL,
    updated_at                  DATETIME(3)   NOT NULL,
    PRIMARY KEY (id),
    KEY idx_storefront_renderables_kind_draft (kind, draft),
    KEY idx_storefront_renderables_latest_published_at (latest_published_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE storefront_renderable_versions (
    id                VARCHAR(64)   NOT NULL,
    renderable_id     VARCHAR(64)   NOT NULL,
    source_version_id VARCHAR(64)   NULL,
    metadata_json     LONGTEXT      NOT NULL,
    content_json      LONGTEXT      NOT NULL,
    snapshot_hash     CHAR(64)      NOT NULL,
    published_at      DATETIME(3)   NOT NULL,
    created_at        DATETIME(3)   NOT NULL,
    PRIMARY KEY (id),
    KEY idx_storefront_renderable_versions_renderable_published (renderable_id, published_at),
    KEY idx_storefront_renderable_versions_renderable_hash (renderable_id, snapshot_hash),
    CONSTRAINT fk_storefront_renderable_versions_renderable
        FOREIGN KEY (renderable_id) REFERENCES storefront_renderables(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_storefront_renderable_versions_source
        FOREIGN KEY (source_version_id) REFERENCES storefront_renderable_versions(id)
        ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE storefront_static_pages (
    id             VARCHAR(64)   NOT NULL,
    renderable_id  VARCHAR(64)   NOT NULL,
    title          VARCHAR(255)  NOT NULL,
    url            VARCHAR(512)  NOT NULL,
    seo_tags_json  LONGTEXT      NOT NULL,
    created_at     DATETIME(3)   NOT NULL,
    updated_at     DATETIME(3)   NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_storefront_static_pages_renderable_id (renderable_id),
    UNIQUE KEY uk_storefront_static_pages_url (url),
    KEY idx_storefront_static_pages_title (title),
    CONSTRAINT fk_storefront_static_pages_renderable
        FOREIGN KEY (renderable_id) REFERENCES storefront_renderables(id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;