ALTER TABLE shipping_quotations
    ADD COLUMN full_freight_cost DECIMAL(15,2) NOT NULL DEFAULT 0 AFTER freight_cost,
    ADD COLUMN discount_percent DECIMAL(5,2) NOT NULL DEFAULT 0 AFTER full_freight_cost,
    ADD COLUMN discounted_freight_cost DECIMAL(15,2) NOT NULL DEFAULT 0 AFTER discount_percent;

UPDATE shipping_quotations
SET
    full_freight_cost = freight_cost,
    discounted_freight_cost = freight_cost
WHERE full_freight_cost = 0
  AND discounted_freight_cost = 0;
