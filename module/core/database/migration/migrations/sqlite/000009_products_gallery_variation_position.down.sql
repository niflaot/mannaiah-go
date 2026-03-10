DROP INDEX IF EXISTS idx_product_gallery_items_variation_position;

ALTER TABLE product_gallery_items DROP COLUMN variation_position;
