-- Restore tag string column as nullable for the backfill pass.
ALTER TABLE product_tags ADD COLUMN tag VARCHAR(128) NULL;

-- Backfill tag name from canonical registry.
UPDATE product_tags pt
    JOIN tags t ON t.id = pt.tag_id
SET pt.tag = t.name;

-- Make tag NOT NULL.
ALTER TABLE product_tags MODIFY COLUMN tag VARCHAR(128) NOT NULL;

-- Drop FK, tag_id indexes, and tag_id column.
ALTER TABLE product_tags
    DROP FOREIGN KEY fk_product_tags_tag_id,
    DROP INDEX idx_product_tags_product_tag_id,
    DROP INDEX idx_product_tags_tag_id,
    DROP COLUMN tag_id;

-- Restore original string-based indexes.
ALTER TABLE product_tags
    ADD UNIQUE KEY idx_product_tags_product_tag (product_id, tag),
    ADD KEY idx_product_tags_tag (tag);
