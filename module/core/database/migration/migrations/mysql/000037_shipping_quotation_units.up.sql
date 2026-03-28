CREATE TABLE IF NOT EXISTS shipping_quotation_units (
    id VARCHAR(64) PRIMARY KEY,
    shipping_quotation_id VARCHAR(64) NOT NULL,
    unit_index INT NOT NULL,
    description VARCHAR(500) NOT NULL DEFAULT '',
    package_type VARCHAR(50) NOT NULL DEFAULT '',
    height_cm DECIMAL(8,2) NOT NULL DEFAULT 0,
    width_cm DECIMAL(8,2) NOT NULL DEFAULT 0,
    depth_cm DECIMAL(8,2) NOT NULL DEFAULT 0,
    real_weight_kg DECIMAL(8,2) NOT NULL DEFAULT 0,
    volumetric_weight_kg DECIMAL(8,2) NOT NULL DEFAULT 0,
    declared_value DECIMAL(15,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_shipping_quotation_units_quotation_id (shipping_quotation_id),
    UNIQUE INDEX uq_shipping_quotation_units_quotation_index (shipping_quotation_id, unit_index),
    CONSTRAINT fk_shipping_quotation_units_quotation
        FOREIGN KEY (shipping_quotation_id) REFERENCES shipping_quotations(id)
        ON DELETE CASCADE
) ENGINE=InnoDB;
