ALTER TABLE shipping_marks
    ADD COLUMN collect_on_delivery_amount DECIMAL(15,2) NOT NULL DEFAULT 0 AFTER payment_form,
    ADD COLUMN collect_on_delivery_discount_percent DECIMAL(5,2) NOT NULL DEFAULT 0 AFTER collect_on_delivery_amount,
    ADD COLUMN collect_on_delivery_charged_amount DECIMAL(15,2) NOT NULL DEFAULT 0 AFTER collect_on_delivery_discount_percent;

UPDATE shipping_marks
SET collect_on_delivery_charged_amount = collect_on_delivery_amount
WHERE collect_on_delivery_amount > 0
  AND collect_on_delivery_charged_amount = 0;
