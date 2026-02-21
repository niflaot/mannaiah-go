CREATE TABLE IF NOT EXISTS contacts (
    id TEXT PRIMARY KEY,
    document_type TEXT,
    document_number TEXT,
    document_key TEXT,
    legal_name TEXT,
    first_name TEXT,
    last_name TEXT,
    email TEXT NOT NULL,
    phone TEXT,
    address TEXT,
    address_extra TEXT,
    city_code TEXT,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_contacts_document_key ON contacts(document_key);
CREATE UNIQUE INDEX IF NOT EXISTS idx_contacts_email ON contacts(email);
CREATE INDEX IF NOT EXISTS idx_contacts_document_type ON contacts(document_type);
CREATE INDEX IF NOT EXISTS idx_contacts_document_number ON contacts(document_number);
CREATE INDEX IF NOT EXISTS idx_contacts_deleted_at ON contacts(deleted_at);

CREATE TABLE IF NOT EXISTS contact_metadata (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    contact_id TEXT NOT NULL,
    key TEXT NOT NULL,
    value TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_contacts_metadata_contact_key ON contact_metadata(contact_id, key);
CREATE INDEX IF NOT EXISTS idx_contact_metadata_contact_id ON contact_metadata(contact_id);

CREATE TABLE IF NOT EXISTS falabella_sync_execution (
    execution_id TEXT PRIMARY KEY,
    started_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS falabella_sync_status (
    execution_id TEXT NOT NULL,
    feed_id TEXT PRIMARY KEY,
    product_id TEXT NOT NULL,
    sku TEXT NOT NULL,
    step TEXT NOT NULL,
    action TEXT NOT NULL,
    status TEXT NOT NULL,
    synced_at DATETIME NOT NULL,
    resolved_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_falabella_sync_status_execution_id ON falabella_sync_status(execution_id);
CREATE INDEX IF NOT EXISTS idx_falabella_sync_status_product_id ON falabella_sync_status(product_id);
CREATE INDEX IF NOT EXISTS idx_falabella_sync_status_status ON falabella_sync_status(status);
