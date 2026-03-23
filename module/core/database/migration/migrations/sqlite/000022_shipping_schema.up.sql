CREATE TABLE dispatch_batches (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    carrier_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'OPEN',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    closed_at DATETIME NULL
);
CREATE INDEX idx_dispatch_batches_carrier_id ON dispatch_batches(carrier_id);
CREATE INDEX idx_dispatch_batches_status ON dispatch_batches(status);

CREATE TABLE shipping_marks (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL,
    carrier_id TEXT NOT NULL,
    tracking_number TEXT UNIQUE,
    status TEXT NOT NULL DEFAULT 'PENDING',
    document_type TEXT,
    document_ref TEXT,
    sender_name TEXT NOT NULL,
    sender_id TEXT NOT NULL,
    sender_id_type TEXT NOT NULL,
    sender_address TEXT NOT NULL,
    sender_city_code TEXT NOT NULL,
    sender_phone TEXT NOT NULL,
    sender_email TEXT NOT NULL,
    recipient_name TEXT NOT NULL,
    recipient_id TEXT NOT NULL,
    recipient_id_type TEXT NOT NULL,
    recipient_address TEXT NOT NULL,
    recipient_city_code TEXT NOT NULL,
    recipient_phone TEXT NOT NULL,
    recipient_email TEXT NOT NULL,
    total_weight REAL NOT NULL DEFAULT 0,
    total_volumetric_weight REAL NOT NULL DEFAULT 0,
    declared_value REAL NOT NULL DEFAULT 0,
    payment_form TEXT NOT NULL DEFAULT '',
    observations TEXT,
    dispatch_batch_id TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (dispatch_batch_id) REFERENCES dispatch_batches(id) ON DELETE SET NULL
);
CREATE INDEX idx_shipping_marks_order_id ON shipping_marks(order_id);
CREATE INDEX idx_shipping_marks_carrier_id ON shipping_marks(carrier_id);
CREATE INDEX idx_shipping_marks_dispatch_batch_id ON shipping_marks(dispatch_batch_id);

CREATE TABLE shipping_mark_units (
    id TEXT PRIMARY KEY,
    shipping_mark_id TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    package_type TEXT NOT NULL DEFAULT '',
    height_cm REAL NOT NULL DEFAULT 0,
    width_cm REAL NOT NULL DEFAULT 0,
    depth_cm REAL NOT NULL DEFAULT 0,
    real_weight_kg REAL NOT NULL DEFAULT 0,
    volumetric_weight_kg REAL NOT NULL DEFAULT 0,
    declared_value REAL NOT NULL DEFAULT 0,
    FOREIGN KEY (shipping_mark_id) REFERENCES shipping_marks(id) ON DELETE CASCADE
);
CREATE INDEX idx_shipping_mark_units_mark_id ON shipping_mark_units(shipping_mark_id);

CREATE TABLE shipping_quotations (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL,
    carrier_id TEXT NOT NULL,
    origin_city_code TEXT NOT NULL,
    dest_city_code TEXT NOT NULL,
    freight_cost REAL NOT NULL DEFAULT 0,
    estimated_days INTEGER NOT NULL DEFAULT 0,
    currency_code TEXT NOT NULL DEFAULT 'COP',
    expires_at DATETIME,
    request_snapshot TEXT NOT NULL DEFAULT '',
    raw_response TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_shipping_quotations_order_id ON shipping_quotations(order_id);
CREATE INDEX idx_shipping_quotations_carrier_id ON shipping_quotations(carrier_id);
