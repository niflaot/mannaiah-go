ALTER TABLE shipping_quotations ADD COLUMN full_freight_cost REAL NOT NULL DEFAULT 0;
ALTER TABLE shipping_quotations ADD COLUMN discount_percent REAL NOT NULL DEFAULT 0;
ALTER TABLE shipping_quotations ADD COLUMN discounted_freight_cost REAL NOT NULL DEFAULT 0;
UPDATE shipping_quotations SET full_freight_cost = freight_cost, discounted_freight_cost = freight_cost WHERE full_freight_cost = 0;
