CREATE TABLE IF NOT EXISTS email_deliveries (
    id TEXT PRIMARY KEY,
    contact_id TEXT NULL,
    email TEXT NOT NULL,
    subject TEXT NOT NULL,
    html_body TEXT NOT NULL,
    text_body TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    provider TEXT NOT NULL,
    provider_message_id TEXT NULL,
    status TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_email_deliveries_idempotency_key ON email_deliveries (idempotency_key);
CREATE UNIQUE INDEX IF NOT EXISTS uq_email_deliveries_provider_message_id ON email_deliveries (provider_message_id);
CREATE INDEX IF NOT EXISTS idx_email_deliveries_contact_id ON email_deliveries (contact_id);
CREATE INDEX IF NOT EXISTS idx_email_deliveries_status_created_at ON email_deliveries (status, created_at);

CREATE TABLE IF NOT EXISTS email_delivery_status_history (
    id TEXT PRIMARY KEY,
    delivery_id TEXT NOT NULL,
    status TEXT NOT NULL,
    reason TEXT NULL,
    occurred_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (delivery_id) REFERENCES email_deliveries (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_email_delivery_status_history_delivery_time ON email_delivery_status_history (delivery_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_email_delivery_status_history_status ON email_delivery_status_history (status);
