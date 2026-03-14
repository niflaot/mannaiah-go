CREATE TABLE IF NOT EXISTS email_deliveries (
    id VARCHAR(36) PRIMARY KEY,
    contact_id VARCHAR(36) NULL,
    email VARCHAR(320) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    html_body LONGTEXT NOT NULL,
    text_body LONGTEXT NOT NULL,
    idempotency_key VARCHAR(128) NOT NULL,
    provider VARCHAR(32) NOT NULL,
    provider_message_id VARCHAR(255) NULL,
    status VARCHAR(32) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_email_deliveries_idempotency_key (idempotency_key),
    UNIQUE KEY uq_email_deliveries_provider_message_id (provider_message_id),
    INDEX idx_email_deliveries_contact_id (contact_id),
    INDEX idx_email_deliveries_status_created_at (status, created_at)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS email_delivery_status_history (
    id VARCHAR(36) PRIMARY KEY,
    delivery_id VARCHAR(36) NOT NULL,
    status VARCHAR(32) NOT NULL,
    reason TEXT NULL,
    occurred_at DATETIME(3) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_email_delivery_status_history_delivery FOREIGN KEY (delivery_id) REFERENCES email_deliveries (id) ON DELETE CASCADE,
    INDEX idx_email_delivery_status_history_delivery_time (delivery_id, occurred_at),
    INDEX idx_email_delivery_status_history_status (status)
) ENGINE=InnoDB;
