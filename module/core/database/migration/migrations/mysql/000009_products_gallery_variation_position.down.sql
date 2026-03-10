DROP INDEX idx_product_gallery_items_variation_position ON product_gallery_items;

ALTER TABLE product_gallery_items
    DROP COLUMN variation_position;
