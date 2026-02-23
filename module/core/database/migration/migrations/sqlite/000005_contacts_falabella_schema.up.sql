CREATE TABLE IF NOT EXISTS contacts (
    id TEXT NOT NULL,
    document_type TEXT NULL,
    document_number TEXT NULL,
    document_key TEXT NULL,
    legal_name TEXT NULL,
    first_name TEXT NULL,
    last_name TEXT NULL,
    email TEXT NOT NULL,
    phone TEXT NULL,
    address TEXT NULL,
    address_extra TEXT NULL,
    city_code TEXT NULL,
    created_at DATETIME NULL,
    updated_at DATETIME NULL,
    deleted_at DATETIME NULL,
    PRIMARY KEY (id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_contacts_document_key ON contacts(document_key);
CREATE UNIQUE INDEX IF NOT EXISTS idx_contacts_email ON contacts(email);
CREATE INDEX IF NOT EXISTS idx_contacts_document_type ON contacts(document_type);
CREATE INDEX IF NOT EXISTS idx_contacts_document_number ON contacts(document_number);
CREATE INDEX IF NOT EXISTS idx_contacts_deleted_at ON contacts(deleted_at);

CREATE TABLE IF NOT EXISTS contact_metadata (
    id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    contact_id TEXT NOT NULL,
    "key" TEXT NOT NULL,
    value TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_contacts_metadata_contact_key ON contact_metadata(contact_id, "key");
CREATE INDEX IF NOT EXISTS idx_contact_metadata_contact_id ON contact_metadata(contact_id);

CREATE TABLE IF NOT EXISTS falabella_sync_execution (
    execution_id TEXT NOT NULL,
    started_at DATETIME NOT NULL,
    PRIMARY KEY (execution_id)
);

CREATE TABLE IF NOT EXISTS falabella_sync_status (
    execution_id TEXT NOT NULL,
    feed_id TEXT NOT NULL,
    product_id TEXT NOT NULL,
    sku TEXT NOT NULL,
    step TEXT NOT NULL,
    action TEXT NOT NULL,
    status TEXT NOT NULL,
    synced_at DATETIME NOT NULL,
    resolved_at DATETIME NULL,
    PRIMARY KEY (feed_id)
);

CREATE INDEX IF NOT EXISTS idx_falabella_sync_status_execution_id ON falabella_sync_status(execution_id);
CREATE INDEX IF NOT EXISTS idx_falabella_sync_status_product_id ON falabella_sync_status(product_id);
CREATE INDEX IF NOT EXISTS idx_falabella_sync_status_status ON falabella_sync_status(status);
