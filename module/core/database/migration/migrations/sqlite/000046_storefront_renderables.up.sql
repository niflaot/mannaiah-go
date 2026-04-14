CREATE TABLE storefront_renderables (
    id                          TEXT      NOT NULL PRIMARY KEY,
    kind                        TEXT      NOT NULL,
    metadata_json               TEXT      NOT NULL,
    content_json                TEXT      NOT NULL,
    snapshot_hash               TEXT      NOT NULL,
    draft                       INTEGER   NOT NULL DEFAULT 1,
    latest_published_version_id TEXT      NULL,
    latest_published_at         DATETIME  NULL,
    created_at                  DATETIME  NOT NULL,
    updated_at                  DATETIME  NOT NULL
);

CREATE INDEX idx_storefront_renderables_kind_draft
    ON storefront_renderables (kind, draft);
CREATE INDEX idx_storefront_renderables_latest_published_at
    ON storefront_renderables (latest_published_at);

CREATE TABLE storefront_renderable_versions (
    id                TEXT      NOT NULL PRIMARY KEY,
    renderable_id     TEXT      NOT NULL,
    source_version_id TEXT      NULL,
    metadata_json     TEXT      NOT NULL,
    content_json      TEXT      NOT NULL,
    snapshot_hash     TEXT      NOT NULL,
    published_at      DATETIME  NOT NULL,
    created_at        DATETIME  NOT NULL,
    FOREIGN KEY (renderable_id) REFERENCES storefront_renderables(id) ON DELETE CASCADE,
    FOREIGN KEY (source_version_id) REFERENCES storefront_renderable_versions(id) ON DELETE SET NULL
);

CREATE INDEX idx_storefront_renderable_versions_renderable_published
    ON storefront_renderable_versions (renderable_id, published_at);
CREATE INDEX idx_storefront_renderable_versions_renderable_hash
    ON storefront_renderable_versions (renderable_id, snapshot_hash);

CREATE TABLE storefront_static_pages (
    id             TEXT      NOT NULL PRIMARY KEY,
    renderable_id  TEXT      NOT NULL UNIQUE,
    title          TEXT      NOT NULL,
    url            TEXT      NOT NULL UNIQUE,
    seo_tags_json  TEXT      NOT NULL,
    created_at     DATETIME  NOT NULL,
    updated_at     DATETIME  NOT NULL,
    FOREIGN KEY (renderable_id) REFERENCES storefront_renderables(id) ON DELETE CASCADE
);

CREATE INDEX idx_storefront_static_pages_title
    ON storefront_static_pages (title);