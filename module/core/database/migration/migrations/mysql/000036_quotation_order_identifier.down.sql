DROP INDEX idx_shipping_quotations_order_identifier ON shipping_quotations;

ALTER TABLE shipping_quotations
    DROP COLUMN order_identifier;
