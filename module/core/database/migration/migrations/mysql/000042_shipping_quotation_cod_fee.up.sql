ALTER TABLE shipping_quotations
    ADD COLUMN collect_on_delivery_fee_amount DECIMAL(15, 2) NOT NULL DEFAULT 0;
