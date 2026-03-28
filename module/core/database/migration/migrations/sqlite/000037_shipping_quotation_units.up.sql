CREATE TABLE IF NOT EXISTS shipping_quotation_units (
    id TEXT PRIMARY KEY,
    shipping_quotation_id TEXT NOT NULL,
    unit_index INTEGER NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    package_type TEXT NOT NULL DEFAULT '',
    height_cm REAL NOT NULL DEFAULT 0,
    width_cm REAL NOT NULL DEFAULT 0,
    depth_cm REAL NOT NULL DEFAULT 0,
    real_weight_kg REAL NOT NULL DEFAULT 0,
    volumetric_weight_kg REAL NOT NULL DEFAULT 0,
    declared_value REAL NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (shipping_quotation_id) REFERENCES shipping_quotations(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_shipping_quotation_units_quotation_id ON shipping_quotation_units (shipping_quotation_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_shipping_quotation_units_quotation_index ON shipping_quotation_units (shipping_quotation_id, unit_index);
