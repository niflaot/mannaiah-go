ALTER TABLE shipping_marks ADD COLUMN quotation_id VARCHAR(64) NULL;
ALTER TABLE shipping_marks ADD COLUMN quoted_freight_cost DECIMAL(15,2) NOT NULL DEFAULT 0;
ALTER TABLE shipping_marks ADD COLUMN draft_snapshot TEXT NULL;
