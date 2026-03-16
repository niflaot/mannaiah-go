DROP TABLE IF EXISTS category_products;
DROP TABLE IF EXISTS category_filter_category_refs;
DROP TABLE IF EXISTS category_filter_price_ranges;
DROP TABLE IF EXISTS category_filter_tags;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS product_tags;
ALTER TABLE products DROP COLUMN IF EXISTS price;
