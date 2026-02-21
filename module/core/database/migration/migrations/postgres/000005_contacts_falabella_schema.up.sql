CREATE TABLE IF NOT EXISTS contacts (
    id VARCHAR(64) PRIMARY KEY,
    document_type VARCHAR(16),
    document_number VARCHAR(128),
    document_key VARCHAR(191),
    legal_name VARCHAR(255),
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    email VARCHAR(255) NOT NULL,
    phone VARCHAR(64),
    address VARCHAR(512),
    address_extra VARCHAR(512),
    city_code VARCHAR(64),
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_contacts_document_key ON contacts(document_key);
CREATE UNIQUE INDEX IF NOT EXISTS idx_contacts_email ON contacts(email);
CREATE INDEX IF NOT EXISTS idx_contacts_document_type ON contacts(document_type);
CREATE INDEX IF NOT EXISTS idx_contacts_document_number ON contacts(document_number);
CREATE INDEX IF NOT EXISTS idx_contacts_deleted_at ON contacts(deleted_at);

CREATE TABLE IF NOT EXISTS contact_metadata (
    id BIGSERIAL PRIMARY KEY,
    contact_id VARCHAR(64) NOT NULL,
    key VARCHAR(128) NOT NULL,
    value TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_contacts_metadata_contact_key ON contact_metadata(contact_id, key);
CREATE INDEX IF NOT EXISTS idx_contact_metadata_contact_id ON contact_metadata(contact_id);

CREATE TABLE IF NOT EXISTS falabella_sync_execution (
    execution_id VARCHAR(191) PRIMARY KEY,
    started_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS falabella_sync_status (
    execution_id VARCHAR(191) NOT NULL,
    feed_id VARCHAR(191) PRIMARY KEY,
    product_id VARCHAR(128) NOT NULL,
    sku VARCHAR(128) NOT NULL,
    step VARCHAR(16) NOT NULL,
    action VARCHAR(16) NOT NULL,
    status VARCHAR(16) NOT NULL,
    synced_at TIMESTAMPTZ NOT NULL,
    resolved_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_falabella_sync_status_execution_id ON falabella_sync_status(execution_id);
CREATE INDEX IF NOT EXISTS idx_falabella_sync_status_product_id ON falabella_sync_status(product_id);
CREATE INDEX IF NOT EXISTS idx_falabella_sync_status_status ON falabella_sync_status(status);
