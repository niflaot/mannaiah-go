ALTER TABLE shipping_quotations
    ADD COLUMN collect_on_delivery_fee_amount REAL NOT NULL DEFAULT 0;
