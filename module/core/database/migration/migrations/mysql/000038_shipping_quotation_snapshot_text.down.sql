UPDATE shipping_quotations
SET request_snapshot = JSON_QUOTE(request_snapshot)
WHERE JSON_VALID(request_snapshot) = 0;

ALTER TABLE shipping_quotations
    MODIFY COLUMN request_snapshot JSON NOT NULL;
