ALTER TABLE product_gallery_items
    ADD COLUMN variation_position INT NULL AFTER position;

CREATE INDEX idx_product_gallery_items_variation_position ON product_gallery_items (variation_position);
