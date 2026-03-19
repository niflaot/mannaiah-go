-- Backfill canonical tag registry with any pre-existing product tags created before v2.4.0.
-- Rows already present are skipped via INSERT IGNORE.
INSERT IGNORE INTO tags (name, created_at, updated_at)
SELECT DISTINCT tag, NOW(3), NOW(3)
FROM product_tags
WHERE tag NOT IN (SELECT name FROM tags);

-- Add tag_id FK column as nullable for the backfill pass.
ALTER TABLE product_tags ADD COLUMN tag_id BIGINT UNSIGNED NULL;

-- Backfill tag_id by joining on tag name.
UPDATE product_tags pt
    JOIN tags t ON t.name = pt.tag
SET pt.tag_id = t.id;

-- Make tag_id NOT NULL now that every row has a value.
ALTER TABLE product_tags MODIFY COLUMN tag_id BIGINT UNSIGNED NOT NULL;

-- Add FK constraint so product_tags can never reference a non-existent tag.
ALTER TABLE product_tags
    ADD CONSTRAINT fk_product_tags_tag_id FOREIGN KEY (tag_id) REFERENCES tags (id);

-- Drop the old string-based unique index and the tag column itself.
ALTER TABLE product_tags
    DROP INDEX idx_product_tags_product_tag,
    DROP INDEX idx_product_tags_tag,
    DROP COLUMN tag;

-- Add new indexes on tag_id.
ALTER TABLE product_tags
    ADD UNIQUE KEY idx_product_tags_product_tag_id (product_id, tag_id),
    ADD KEY idx_product_tags_tag_id (tag_id);
