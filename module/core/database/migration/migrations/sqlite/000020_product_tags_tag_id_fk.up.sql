-- Backfill canonical tag registry with any pre-existing product tags created before v2.4.0.
INSERT OR IGNORE INTO tags (name, created_at, updated_at)
SELECT DISTINCT tag, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM product_tags
WHERE tag NOT IN (SELECT name FROM tags);

-- Recreate product_tags with tag_id replacing the tag string column.
-- SQLite does not support ALTER TABLE DROP COLUMN or ADD FOREIGN KEY on existing tables.
CREATE TABLE product_tags_new
(
    id         INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    product_id TEXT    NOT NULL,
    position   INTEGER NOT NULL,
    tag_id     INTEGER NOT NULL REFERENCES tags (id)
);

-- Backfill rows by joining on tag name.
INSERT INTO product_tags_new (id, product_id, position, tag_id)
SELECT pt.id, pt.product_id, pt.position, t.id
FROM product_tags pt
         JOIN tags t ON t.name = pt.tag;

DROP TABLE product_tags;
ALTER TABLE product_tags_new RENAME TO product_tags;

CREATE UNIQUE INDEX IF NOT EXISTS idx_product_tags_product_tag_id ON product_tags (product_id, tag_id);
CREATE INDEX IF NOT EXISTS idx_product_tags_product_id ON product_tags (product_id);
CREATE INDEX IF NOT EXISTS idx_product_tags_tag_id ON product_tags (tag_id);
