ALTER TABLE shipping_marks
    ADD COLUMN custom_tracking_url VARCHAR(2048) NULL DEFAULT NULL AFTER failure_reason;
