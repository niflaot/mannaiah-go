ALTER TABLE shipping_marks
    CHANGE COLUMN collect_on_delivery_discount_percent collect_on_delivery_fee_percent DECIMAL(5,2) NOT NULL DEFAULT 0;
