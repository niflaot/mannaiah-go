CREATE TABLE IF NOT EXISTS variations (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    definition TEXT NOT NULL,
    value TEXT NOT NULL,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_variations_definition ON variations(definition);
CREATE INDEX IF NOT EXISTS idx_variations_deleted_at ON variations(deleted_at);

CREATE TABLE IF NOT EXISTS products (
    id TEXT PRIMARY KEY,
    sku TEXT NOT NULL,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_products_sku ON products(sku);
CREATE INDEX IF NOT EXISTS idx_products_deleted_at ON products(deleted_at);

CREATE TABLE IF NOT EXISTS product_gallery_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    product_id TEXT NOT NULL,
    position INTEGER NOT NULL,
    asset_id TEXT NOT NULL,
    is_main BOOLEAN NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_product_gallery_items_product_id ON product_gallery_items(product_id);
CREATE INDEX IF NOT EXISTS idx_product_gallery_items_position ON product_gallery_items(position);

CREATE TABLE IF NOT EXISTS product_gallery_excluded_realms (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    gallery_item_id INTEGER NOT NULL,
    position INTEGER NOT NULL,
    realm TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_product_gallery_excluded_realms_gallery_item_id ON product_gallery_excluded_realms(gallery_item_id);
CREATE INDEX IF NOT EXISTS idx_product_gallery_excluded_realms_position ON product_gallery_excluded_realms(position);

CREATE TABLE IF NOT EXISTS product_gallery_variations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    gallery_item_id INTEGER NOT NULL,
    position INTEGER NOT NULL,
    variation_id TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_product_gallery_variations_gallery_item_id ON product_gallery_variations(gallery_item_id);
CREATE INDEX IF NOT EXISTS idx_product_gallery_variations_position ON product_gallery_variations(position);

CREATE TABLE IF NOT EXISTS product_datasheets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    product_id TEXT NOT NULL,
    position INTEGER NOT NULL,
    realm TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT
);

CREATE INDEX IF NOT EXISTS idx_product_datasheets_product_id ON product_datasheets(product_id);
CREATE INDEX IF NOT EXISTS idx_product_datasheets_position ON product_datasheets(position);

CREATE TABLE IF NOT EXISTS product_datasheet_attributes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    datasheet_id INTEGER NOT NULL,
    key TEXT NOT NULL,
    value_json TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_product_datasheet_attributes_datasheet_id ON product_datasheet_attributes(datasheet_id);

CREATE TABLE IF NOT EXISTS product_variation_links (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    product_id TEXT NOT NULL,
    position INTEGER NOT NULL,
    variation_id TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_product_variation_links_product_id ON product_variation_links(product_id);
CREATE INDEX IF NOT EXISTS idx_product_variation_links_position ON product_variation_links(position);

CREATE TABLE IF NOT EXISTS product_variants (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    product_id TEXT NOT NULL,
    position INTEGER NOT NULL,
    sku TEXT
);

CREATE INDEX IF NOT EXISTS idx_product_variants_product_id ON product_variants(product_id);
CREATE INDEX IF NOT EXISTS idx_product_variants_position ON product_variants(position);

CREATE TABLE IF NOT EXISTS product_variant_variations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    variant_id INTEGER NOT NULL,
    position INTEGER NOT NULL,
    variation_id TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_product_variant_variations_variant_id ON product_variant_variations(variant_id);
CREATE INDEX IF NOT EXISTS idx_product_variant_variations_position ON product_variant_variations(position);

CREATE TABLE IF NOT EXISTS orders (
    id TEXT PRIMARY KEY,
    identifier TEXT NOT NULL,
    realm TEXT NOT NULL,
    contact_id TEXT NOT NULL,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_orders_realm_identifier ON orders(realm, identifier);
CREATE INDEX IF NOT EXISTS idx_orders_contact_id ON orders(contact_id);
CREATE INDEX IF NOT EXISTS idx_orders_deleted_at ON orders(deleted_at);

CREATE TABLE IF NOT EXISTS order_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id TEXT NOT NULL,
    position INTEGER NOT NULL,
    sku TEXT NOT NULL,
    alternate_name TEXT,
    quantity INTEGER NOT NULL,
    value REAL NOT NULL DEFAULT 0,
    product_id TEXT,
    resolution_source TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_order_items_order_position ON order_items(order_id, position);
CREATE INDEX IF NOT EXISTS idx_order_items_sku ON order_items(sku);
CREATE INDEX IF NOT EXISTS idx_order_items_alternate_name ON order_items(alternate_name);
CREATE INDEX IF NOT EXISTS idx_order_items_product_id ON order_items(product_id);

CREATE TABLE IF NOT EXISTS order_status_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id TEXT NOT NULL,
    position INTEGER NOT NULL,
    status TEXT NOT NULL,
    author TEXT NOT NULL,
    description TEXT,
    note_owner TEXT,
    note TEXT,
    occurred_at DATETIME NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_order_status_order_position ON order_status_history(order_id, position);
CREATE INDEX IF NOT EXISTS idx_order_status_status ON order_status_history(status);
CREATE INDEX IF NOT EXISTS idx_order_status_occurred_at ON order_status_history(occurred_at);
CREATE INDEX IF NOT EXISTS idx_order_status_order_occurred ON order_status_history(order_id, occurred_at);

CREATE TABLE IF NOT EXISTS order_comments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id TEXT NOT NULL,
    author TEXT NOT NULL,
    comment TEXT NOT NULL,
    internal BOOLEAN NOT NULL DEFAULT 0,
    occurred_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_order_comments_order_id ON order_comments(order_id);
CREATE INDEX IF NOT EXISTS idx_order_comments_occurred_at ON order_comments(occurred_at);
CREATE INDEX IF NOT EXISTS idx_order_comments_order_occurred ON order_comments(order_id, occurred_at);

CREATE TABLE IF NOT EXISTS order_shipping_addresses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id TEXT NOT NULL,
    address TEXT NOT NULL,
    address2 TEXT,
    phone TEXT,
    city_code TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_order_shipping_addresses_order_id ON order_shipping_addresses(order_id);

CREATE TABLE IF NOT EXISTS order_shipping_charges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id TEXT NOT NULL,
    position INTEGER NOT NULL,
    method_id TEXT,
    method_title TEXT,
    price REAL NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_order_shipping_charges_order_position ON order_shipping_charges(order_id, position);

CREATE TABLE IF NOT EXISTS order_metadata (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id TEXT NOT NULL,
    key TEXT NOT NULL,
    value TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_order_metadata_order_key ON order_metadata(order_id, key);
CREATE INDEX IF NOT EXISTS idx_order_metadata_order_id ON order_metadata(order_id);
