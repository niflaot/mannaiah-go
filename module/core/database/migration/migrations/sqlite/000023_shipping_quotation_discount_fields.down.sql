-- SQLite rollback recreates shipping_quotations without discount fields.
CREATE TABLE shipping_quotations_backup (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL,
    carrier_id TEXT NOT NULL,
    origin_city_code TEXT NOT NULL,
    dest_city_code TEXT NOT NULL,
    freight_cost REAL NOT NULL DEFAULT 0,
    estimated_days INTEGER NOT NULL DEFAULT 0,
    currency_code TEXT NOT NULL DEFAULT 'COP',
    expires_at DATETIME,
    request_snapshot TEXT NOT NULL DEFAULT '',
    raw_response TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO shipping_quotations_backup (
    id,
    order_id,
    carrier_id,
    origin_city_code,
    dest_city_code,
    freight_cost,
    estimated_days,
    currency_code,
    expires_at,
    request_snapshot,
    raw_response,
    created_at
)
SELECT
    id,
    order_id,
    carrier_id,
    origin_city_code,
    dest_city_code,
    freight_cost,
    estimated_days,
    currency_code,
    expires_at,
    request_snapshot,
    raw_response,
    created_at
FROM shipping_quotations;

DROP TABLE shipping_quotations;
ALTER TABLE shipping_quotations_backup RENAME TO shipping_quotations;
CREATE INDEX idx_shipping_quotations_order_id ON shipping_quotations(order_id);
CREATE INDEX idx_shipping_quotations_carrier_id ON shipping_quotations(carrier_id);
