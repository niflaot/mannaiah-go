CREATE TABLE IF NOT EXISTS contacts_snapshot (
    contact_id String,
    email String,
    first_name String,
    last_name String,
    legal_name String,
    phone String,
    city_code String,
    document_type String,
    metadata_json String,
    created_at DateTime64(3),
    updated_at DateTime64(3)
) ENGINE = ReplacingMergeTree(updated_at)
ORDER BY contact_id;
