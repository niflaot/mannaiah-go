ALTER TABLE shipping_marks ADD COLUMN collect_on_delivery_amount REAL NOT NULL DEFAULT 0;
ALTER TABLE shipping_marks ADD COLUMN collect_on_delivery_discount_percent REAL NOT NULL DEFAULT 0;
ALTER TABLE shipping_marks ADD COLUMN collect_on_delivery_charged_amount REAL NOT NULL DEFAULT 0;

UPDATE shipping_marks
SET collect_on_delivery_charged_amount = collect_on_delivery_amount
WHERE collect_on_delivery_amount > 0
  AND collect_on_delivery_charged_amount = 0;
