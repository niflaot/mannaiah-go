ALTER TABLE shipping_quotations
    ADD COLUMN order_identifier VARCHAR(255) NULL DEFAULT NULL AFTER order_id;

CREATE INDEX idx_shipping_quotations_order_identifier ON shipping_quotations (order_identifier);
