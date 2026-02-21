CREATE TABLE IF NOT EXISTS variations (
    id VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    definition VARCHAR(32) NOT NULL,
    value VARCHAR(255) NOT NULL,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    KEY idx_variations_definition (definition),
    KEY idx_variations_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS products (
    id VARCHAR(64) NOT NULL,
    sku VARCHAR(255) NOT NULL,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_products_sku (sku),
    KEY idx_products_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS product_gallery_items (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    product_id VARCHAR(64) NOT NULL,
    position INT NOT NULL,
    asset_id VARCHAR(64) NOT NULL,
    is_main BOOLEAN NOT NULL,
    PRIMARY KEY (id),
    KEY idx_product_gallery_items_product_id (product_id),
    KEY idx_product_gallery_items_position (position)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS product_gallery_excluded_realms (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    gallery_item_id BIGINT UNSIGNED NOT NULL,
    position INT NOT NULL,
    realm VARCHAR(128) NOT NULL,
    PRIMARY KEY (id),
    KEY idx_product_gallery_excluded_realms_gallery_item_id (gallery_item_id),
    KEY idx_product_gallery_excluded_realms_position (position)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS product_gallery_variations (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    gallery_item_id BIGINT UNSIGNED NOT NULL,
    position INT NOT NULL,
    variation_id VARCHAR(64) NOT NULL,
    PRIMARY KEY (id),
    KEY idx_product_gallery_variations_gallery_item_id (gallery_item_id),
    KEY idx_product_gallery_variations_position (position)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS product_datasheets (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    product_id VARCHAR(64) NOT NULL,
    position INT NOT NULL,
    realm VARCHAR(128) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT NULL,
    PRIMARY KEY (id),
    KEY idx_product_datasheets_product_id (product_id),
    KEY idx_product_datasheets_position (position)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS product_datasheet_attributes (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    datasheet_id BIGINT UNSIGNED NOT NULL,
    `key` VARCHAR(128) NOT NULL,
    value_json TEXT NOT NULL,
    PRIMARY KEY (id),
    KEY idx_product_datasheet_attributes_datasheet_id (datasheet_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS product_variation_links (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    product_id VARCHAR(64) NOT NULL,
    position INT NOT NULL,
    variation_id VARCHAR(64) NOT NULL,
    PRIMARY KEY (id),
    KEY idx_product_variation_links_product_id (product_id),
    KEY idx_product_variation_links_position (position)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS product_variants (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    product_id VARCHAR(64) NOT NULL,
    position INT NOT NULL,
    sku VARCHAR(255) NULL,
    PRIMARY KEY (id),
    KEY idx_product_variants_product_id (product_id),
    KEY idx_product_variants_position (position)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS product_variant_variations (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    variant_id BIGINT UNSIGNED NOT NULL,
    position INT NOT NULL,
    variation_id VARCHAR(64) NOT NULL,
    PRIMARY KEY (id),
    KEY idx_product_variant_variations_variant_id (variant_id),
    KEY idx_product_variant_variations_position (position)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS orders (
    id VARCHAR(64) NOT NULL,
    identifier VARCHAR(255) NOT NULL,
    realm VARCHAR(128) NOT NULL,
    contact_id VARCHAR(64) NOT NULL,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_orders_realm_identifier (realm, identifier),
    KEY idx_orders_contact_id (contact_id),
    KEY idx_orders_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS order_items (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    order_id VARCHAR(64) NOT NULL,
    position INT NOT NULL,
    sku VARCHAR(255) NOT NULL,
    alternate_name VARCHAR(255) NULL,
    quantity INT NOT NULL,
    value DOUBLE NOT NULL DEFAULT 0,
    product_id VARCHAR(64) NULL,
    resolution_source VARCHAR(64) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_order_items_order_position (order_id, position),
    KEY idx_order_items_sku (sku),
    KEY idx_order_items_alternate_name (alternate_name),
    KEY idx_order_items_product_id (product_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS order_status_history (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    order_id VARCHAR(64) NOT NULL,
    position INT NOT NULL,
    status VARCHAR(32) NOT NULL,
    author VARCHAR(255) NOT NULL,
    description TEXT NULL,
    note_owner VARCHAR(255) NULL,
    note TEXT NULL,
    occurred_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_order_status_order_position (order_id, position),
    KEY idx_order_status_status (status),
    KEY idx_order_status_occurred_at (occurred_at),
    KEY idx_order_status_order_occurred (order_id, occurred_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS order_comments (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    order_id VARCHAR(64) NOT NULL,
    author VARCHAR(255) NOT NULL,
    comment TEXT NOT NULL,
    internal BOOLEAN NOT NULL DEFAULT FALSE,
    occurred_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id),
    KEY idx_order_comments_order_id (order_id),
    KEY idx_order_comments_occurred_at (occurred_at),
    KEY idx_order_comments_order_occurred (order_id, occurred_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS order_shipping_addresses (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    order_id VARCHAR(64) NOT NULL,
    address VARCHAR(512) NOT NULL,
    address2 VARCHAR(512) NULL,
    phone VARCHAR(64) NULL,
    city_code VARCHAR(64) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_order_shipping_addresses_order_id (order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS order_shipping_charges (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    order_id VARCHAR(64) NOT NULL,
    position INT NOT NULL,
    method_id VARCHAR(128) NULL,
    method_title VARCHAR(255) NULL,
    price DOUBLE NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY idx_order_shipping_charges_order_position (order_id, position)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS order_metadata (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    order_id VARCHAR(64) NOT NULL,
    `key` VARCHAR(128) NOT NULL,
    value TEXT NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_order_metadata_order_key (order_id, `key`),
    KEY idx_order_metadata_order_id (order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
