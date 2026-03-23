CREATE TABLE IF NOT EXISTS dispatch_batches (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    carrier_id VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'OPEN',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    closed_at TIMESTAMP NULL,
    INDEX idx_dispatch_batches_carrier_id (carrier_id),
    INDEX idx_dispatch_batches_status (status)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS shipping_marks (
    id VARCHAR(64) PRIMARY KEY,
    order_id VARCHAR(255) NOT NULL,
    carrier_id VARCHAR(100) NOT NULL,
    tracking_number VARCHAR(255) NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    document_type VARCHAR(10) NULL,
    document_ref TEXT NULL,
    sender_name VARCHAR(255) NOT NULL,
    sender_id VARCHAR(50) NOT NULL,
    sender_id_type VARCHAR(10) NOT NULL,
    sender_address VARCHAR(500) NOT NULL,
    sender_city_code VARCHAR(20) NOT NULL,
    sender_phone VARCHAR(50) NOT NULL,
    sender_email VARCHAR(255) NOT NULL,
    recipient_name VARCHAR(255) NOT NULL,
    recipient_id VARCHAR(50) NOT NULL,
    recipient_id_type VARCHAR(10) NOT NULL,
    recipient_address VARCHAR(500) NOT NULL,
    recipient_city_code VARCHAR(20) NOT NULL,
    recipient_phone VARCHAR(50) NOT NULL,
    recipient_email VARCHAR(255) NOT NULL,
    total_weight DECIMAL(10,2) NOT NULL DEFAULT 0,
    total_volumetric_weight DECIMAL(10,2) NOT NULL DEFAULT 0,
    declared_value DECIMAL(15,2) NOT NULL DEFAULT 0,
    payment_form VARCHAR(50) NOT NULL DEFAULT '',
    observations TEXT NULL,
    dispatch_batch_id VARCHAR(64) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_shipping_marks_tracking_number (tracking_number),
    INDEX idx_shipping_marks_order_id (order_id),
    INDEX idx_shipping_marks_carrier_id (carrier_id),
    INDEX idx_shipping_marks_dispatch_batch_id (dispatch_batch_id),
    CONSTRAINT fk_shipping_marks_dispatch_batch
        FOREIGN KEY (dispatch_batch_id) REFERENCES dispatch_batches(id)
        ON DELETE SET NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS shipping_mark_units (
    id VARCHAR(64) PRIMARY KEY,
    shipping_mark_id VARCHAR(64) NOT NULL,
    description VARCHAR(500) NOT NULL DEFAULT '',
    package_type VARCHAR(50) NOT NULL DEFAULT '',
    height_cm DECIMAL(8,2) NOT NULL DEFAULT 0,
    width_cm DECIMAL(8,2) NOT NULL DEFAULT 0,
    depth_cm DECIMAL(8,2) NOT NULL DEFAULT 0,
    real_weight_kg DECIMAL(8,2) NOT NULL DEFAULT 0,
    volumetric_weight_kg DECIMAL(8,2) NOT NULL DEFAULT 0,
    declared_value DECIMAL(15,2) NOT NULL DEFAULT 0,
    INDEX idx_shipping_mark_units_mark_id (shipping_mark_id),
    CONSTRAINT fk_shipping_mark_units_mark
        FOREIGN KEY (shipping_mark_id) REFERENCES shipping_marks(id)
        ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS shipping_quotations (
    id VARCHAR(64) PRIMARY KEY,
    order_id VARCHAR(255) NOT NULL,
    carrier_id VARCHAR(100) NOT NULL,
    origin_city_code VARCHAR(20) NOT NULL,
    dest_city_code VARCHAR(20) NOT NULL,
    freight_cost DECIMAL(15,2) NOT NULL DEFAULT 0,
    estimated_days INT NOT NULL DEFAULT 0,
    currency_code VARCHAR(5) NOT NULL DEFAULT 'COP',
    expires_at TIMESTAMP NULL,
    request_snapshot JSON NOT NULL,
    raw_response TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_shipping_quotations_order_id (order_id),
    INDEX idx_shipping_quotations_carrier_id (carrier_id)
) ENGINE=InnoDB;
