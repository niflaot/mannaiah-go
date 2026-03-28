ALTER TABLE shipping_quotations ADD COLUMN order_identifier TEXT NULL DEFAULT NULL;
CREATE INDEX idx_shipping_quotations_order_identifier ON shipping_quotations (order_identifier);
