ALTER TABLE products ADD COLUMN price DOUBLE NULL;

CREATE TABLE IF NOT EXISTS product_tags (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    product_id VARCHAR(64) NOT NULL,
    position INT NOT NULL,
    tag VARCHAR(128) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_product_tags_product_tag (product_id, tag),
    KEY idx_product_tags_product_id (product_id),
    KEY idx_product_tags_tag (tag)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS categories (
    id VARCHAR(64) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT NULL,
    parent_id VARCHAR(64) NULL,
    include_children BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_categories_slug (slug),
    KEY idx_categories_parent_id (parent_id),
    KEY idx_categories_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS category_filter_tags (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    category_id VARCHAR(64) NOT NULL,
    position INT NOT NULL,
    tag VARCHAR(128) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_category_filter_tags_cat_tag (category_id, tag),
    KEY idx_category_filter_tags_category_id (category_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS category_filter_price_ranges (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    category_id VARCHAR(64) NOT NULL,
    min_price DOUBLE NULL,
    max_price DOUBLE NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_category_filter_price_ranges_category_id (category_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS category_filter_category_refs (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    category_id VARCHAR(64) NOT NULL,
    ref_category_id VARCHAR(64) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_category_filter_category_refs_pair (category_id, ref_category_id),
    KEY idx_category_filter_category_refs_category_id (category_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS category_products (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    category_id VARCHAR(64) NOT NULL,
    product_id VARCHAR(64) NOT NULL,
    position INT NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY idx_category_products_pair (category_id, product_id),
    KEY idx_category_products_category_id (category_id),
    KEY idx_category_products_product_id (product_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
