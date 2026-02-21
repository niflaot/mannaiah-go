CREATE TABLE IF NOT EXISTS contacts (
    id VARCHAR(64) NOT NULL,
    document_type VARCHAR(16) NULL,
    document_number VARCHAR(128) NULL,
    document_key VARCHAR(191) NULL,
    legal_name VARCHAR(255) NULL,
    first_name VARCHAR(255) NULL,
    last_name VARCHAR(255) NULL,
    email VARCHAR(255) NOT NULL,
    phone VARCHAR(64) NULL,
    address VARCHAR(512) NULL,
    address_extra VARCHAR(512) NULL,
    city_code VARCHAR(64) NULL,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_contacts_document_key (document_key),
    UNIQUE KEY idx_contacts_email (email),
    KEY idx_contacts_document_type (document_type),
    KEY idx_contacts_document_number (document_number),
    KEY idx_contacts_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS contact_metadata (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    contact_id VARCHAR(64) NOT NULL,
    `key` VARCHAR(128) NOT NULL,
    value TEXT NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_contacts_metadata_contact_key (contact_id, `key`),
    KEY idx_contact_metadata_contact_id (contact_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS falabella_sync_execution (
    execution_id VARCHAR(191) NOT NULL,
    started_at DATETIME(3) NOT NULL,
    PRIMARY KEY (execution_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS falabella_sync_status (
    execution_id VARCHAR(191) NOT NULL,
    feed_id VARCHAR(191) NOT NULL,
    product_id VARCHAR(128) NOT NULL,
    sku VARCHAR(128) NOT NULL,
    step VARCHAR(16) NOT NULL,
    action VARCHAR(16) NOT NULL,
    status VARCHAR(16) NOT NULL,
    synced_at DATETIME(3) NOT NULL,
    resolved_at DATETIME(3) NULL,
    PRIMARY KEY (feed_id),
    KEY idx_falabella_sync_status_execution_id (execution_id),
    KEY idx_falabella_sync_status_product_id (product_id),
    KEY idx_falabella_sync_status_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
