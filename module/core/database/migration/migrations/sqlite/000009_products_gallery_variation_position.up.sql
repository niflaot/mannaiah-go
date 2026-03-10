ALTER TABLE product_gallery_items ADD COLUMN variation_position INTEGER NULL;

CREATE INDEX IF NOT EXISTS idx_product_gallery_items_variation_position ON product_gallery_items(variation_position);
