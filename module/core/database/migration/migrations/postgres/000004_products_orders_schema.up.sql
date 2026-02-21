CREATE TABLE IF NOT EXISTS variations (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    definition VARCHAR(32) NOT NULL,
    value VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_variations_definition ON variations(definition);
CREATE INDEX IF NOT EXISTS idx_variations_deleted_at ON variations(deleted_at);

CREATE TABLE IF NOT EXISTS products (
    id VARCHAR(64) PRIMARY KEY,
    sku VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_products_sku ON products(sku);
CREATE INDEX IF NOT EXISTS idx_products_deleted_at ON products(deleted_at);

CREATE TABLE IF NOT EXISTS product_gallery_items (
    id BIGSERIAL PRIMARY KEY,
    product_id VARCHAR(64) NOT NULL,
    position INTEGER NOT NULL,
    asset_id VARCHAR(64) NOT NULL,
    is_main BOOLEAN NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_product_gallery_items_product_id ON product_gallery_items(product_id);
CREATE INDEX IF NOT EXISTS idx_product_gallery_items_position ON product_gallery_items(position);

CREATE TABLE IF NOT EXISTS product_gallery_excluded_realms (
    id BIGSERIAL PRIMARY KEY,
    gallery_item_id BIGINT NOT NULL,
    position INTEGER NOT NULL,
    realm VARCHAR(128) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_product_gallery_excluded_realms_gallery_item_id ON product_gallery_excluded_realms(gallery_item_id);
CREATE INDEX IF NOT EXISTS idx_product_gallery_excluded_realms_position ON product_gallery_excluded_realms(position);

CREATE TABLE IF NOT EXISTS product_gallery_variations (
    id BIGSERIAL PRIMARY KEY,
    gallery_item_id BIGINT NOT NULL,
    position INTEGER NOT NULL,
    variation_id VARCHAR(64) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_product_gallery_variations_gallery_item_id ON product_gallery_variations(gallery_item_id);
CREATE INDEX IF NOT EXISTS idx_product_gallery_variations_position ON product_gallery_variations(position);

CREATE TABLE IF NOT EXISTS product_datasheets (
    id BIGSERIAL PRIMARY KEY,
    product_id VARCHAR(64) NOT NULL,
    position INTEGER NOT NULL,
    realm VARCHAR(128) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT
);

CREATE INDEX IF NOT EXISTS idx_product_datasheets_product_id ON product_datasheets(product_id);
CREATE INDEX IF NOT EXISTS idx_product_datasheets_position ON product_datasheets(position);

CREATE TABLE IF NOT EXISTS product_datasheet_attributes (
    id BIGSERIAL PRIMARY KEY,
    datasheet_id BIGINT NOT NULL,
    key VARCHAR(128) NOT NULL,
    value_json TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_product_datasheet_attributes_datasheet_id ON product_datasheet_attributes(datasheet_id);

CREATE TABLE IF NOT EXISTS product_variation_links (
    id BIGSERIAL PRIMARY KEY,
    product_id VARCHAR(64) NOT NULL,
    position INTEGER NOT NULL,
    variation_id VARCHAR(64) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_product_variation_links_product_id ON product_variation_links(product_id);
CREATE INDEX IF NOT EXISTS idx_product_variation_links_position ON product_variation_links(position);

CREATE TABLE IF NOT EXISTS product_variants (
    id BIGSERIAL PRIMARY KEY,
    product_id VARCHAR(64) NOT NULL,
    position INTEGER NOT NULL,
    sku VARCHAR(255)
);

CREATE INDEX IF NOT EXISTS idx_product_variants_product_id ON product_variants(product_id);
CREATE INDEX IF NOT EXISTS idx_product_variants_position ON product_variants(position);

CREATE TABLE IF NOT EXISTS product_variant_variations (
    id BIGSERIAL PRIMARY KEY,
    variant_id BIGINT NOT NULL,
    position INTEGER NOT NULL,
    variation_id VARCHAR(64) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_product_variant_variations_variant_id ON product_variant_variations(variant_id);
CREATE INDEX IF NOT EXISTS idx_product_variant_variations_position ON product_variant_variations(position);

CREATE TABLE IF NOT EXISTS orders (
    id VARCHAR(64) PRIMARY KEY,
    identifier VARCHAR(255) NOT NULL,
    realm VARCHAR(128) NOT NULL,
    contact_id VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_orders_realm_identifier ON orders(realm, identifier);
CREATE INDEX IF NOT EXISTS idx_orders_contact_id ON orders(contact_id);
CREATE INDEX IF NOT EXISTS idx_orders_deleted_at ON orders(deleted_at);

CREATE TABLE IF NOT EXISTS order_items (
    id BIGSERIAL PRIMARY KEY,
    order_id VARCHAR(64) NOT NULL,
    position INTEGER NOT NULL,
    sku VARCHAR(255) NOT NULL,
    alternate_name VARCHAR(255),
    quantity INTEGER NOT NULL,
    value DOUBLE PRECISION NOT NULL DEFAULT 0,
    product_id VARCHAR(64),
    resolution_source VARCHAR(64) NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_order_items_order_position ON order_items(order_id, position);
CREATE INDEX IF NOT EXISTS idx_order_items_sku ON order_items(sku);
CREATE INDEX IF NOT EXISTS idx_order_items_alternate_name ON order_items(alternate_name);
CREATE INDEX IF NOT EXISTS idx_order_items_product_id ON order_items(product_id);

CREATE TABLE IF NOT EXISTS order_status_history (
    id BIGSERIAL PRIMARY KEY,
    order_id VARCHAR(64) NOT NULL,
    position INTEGER NOT NULL,
    status VARCHAR(32) NOT NULL,
    author VARCHAR(255) NOT NULL,
    description TEXT,
    note_owner VARCHAR(255),
    note TEXT,
    occurred_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_order_status_order_position ON order_status_history(order_id, position);
CREATE INDEX IF NOT EXISTS idx_order_status_status ON order_status_history(status);
CREATE INDEX IF NOT EXISTS idx_order_status_occurred_at ON order_status_history(occurred_at);
CREATE INDEX IF NOT EXISTS idx_order_status_order_occurred ON order_status_history(order_id, occurred_at);

CREATE TABLE IF NOT EXISTS order_comments (
    id BIGSERIAL PRIMARY KEY,
    order_id VARCHAR(64) NOT NULL,
    author VARCHAR(255) NOT NULL,
    comment TEXT NOT NULL,
    internal BOOLEAN NOT NULL DEFAULT FALSE,
    occurred_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_order_comments_order_id ON order_comments(order_id);
CREATE INDEX IF NOT EXISTS idx_order_comments_occurred_at ON order_comments(occurred_at);
CREATE INDEX IF NOT EXISTS idx_order_comments_order_occurred ON order_comments(order_id, occurred_at);

CREATE TABLE IF NOT EXISTS order_shipping_addresses (
    id BIGSERIAL PRIMARY KEY,
    order_id VARCHAR(64) NOT NULL,
    address VARCHAR(512) NOT NULL,
    address2 VARCHAR(512),
    phone VARCHAR(64),
    city_code VARCHAR(64) NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_order_shipping_addresses_order_id ON order_shipping_addresses(order_id);

CREATE TABLE IF NOT EXISTS order_shipping_charges (
    id BIGSERIAL PRIMARY KEY,
    order_id VARCHAR(64) NOT NULL,
    position INTEGER NOT NULL,
    method_id VARCHAR(128),
    method_title VARCHAR(255),
    price DOUBLE PRECISION NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_order_shipping_charges_order_position ON order_shipping_charges(order_id, position);

CREATE TABLE IF NOT EXISTS order_metadata (
    id BIGSERIAL PRIMARY KEY,
    order_id VARCHAR(64) NOT NULL,
    key VARCHAR(128) NOT NULL,
    value TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_order_metadata_order_key ON order_metadata(order_id, key);
CREATE INDEX IF NOT EXISTS idx_order_metadata_order_id ON order_metadata(order_id);
