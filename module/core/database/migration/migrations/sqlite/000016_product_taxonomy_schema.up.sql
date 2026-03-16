ALTER TABLE products ADD COLUMN price DOUBLE NULL;

CREATE TABLE IF NOT EXISTS product_tags (
    id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    product_id TEXT NOT NULL,
    position INTEGER NOT NULL,
    tag TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_product_tags_product_tag ON product_tags(product_id, tag);
CREATE INDEX IF NOT EXISTS idx_product_tags_product_id ON product_tags(product_id);
CREATE INDEX IF NOT EXISTS idx_product_tags_tag ON product_tags(tag);

CREATE TABLE IF NOT EXISTS categories (
    id TEXT NOT NULL,
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NULL,
    parent_id TEXT NULL,
    include_children BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME NULL,
    updated_at DATETIME NULL,
    deleted_at DATETIME NULL,
    PRIMARY KEY (id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_slug ON categories(slug);
CREATE INDEX IF NOT EXISTS idx_categories_parent_id ON categories(parent_id);
CREATE INDEX IF NOT EXISTS idx_categories_deleted_at ON categories(deleted_at);

CREATE TABLE IF NOT EXISTS category_filter_tags (
    id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    category_id TEXT NOT NULL,
    position INTEGER NOT NULL,
    tag TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_category_filter_tags_cat_tag ON category_filter_tags(category_id, tag);
CREATE INDEX IF NOT EXISTS idx_category_filter_tags_category_id ON category_filter_tags(category_id);

CREATE TABLE IF NOT EXISTS category_filter_price_ranges (
    id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    category_id TEXT NOT NULL,
    min_price DOUBLE NULL,
    max_price DOUBLE NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_category_filter_price_ranges_category_id ON category_filter_price_ranges(category_id);

CREATE TABLE IF NOT EXISTS category_filter_category_refs (
    id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    category_id TEXT NOT NULL,
    ref_category_id TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_category_filter_category_refs_pair ON category_filter_category_refs(category_id, ref_category_id);
CREATE INDEX IF NOT EXISTS idx_category_filter_category_refs_category_id ON category_filter_category_refs(category_id);

CREATE TABLE IF NOT EXISTS category_products (
    id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    category_id TEXT NOT NULL,
    product_id TEXT NOT NULL,
    position INTEGER NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_category_products_pair ON category_products(category_id, product_id);
CREATE INDEX IF NOT EXISTS idx_category_products_category_id ON category_products(category_id);
CREATE INDEX IF NOT EXISTS idx_category_products_product_id ON category_products(product_id);
