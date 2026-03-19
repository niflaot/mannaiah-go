-- Recreate product_tags with the original tag string column.
CREATE TABLE product_tags_old
(
    id         INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    product_id TEXT    NOT NULL,
    position   INTEGER NOT NULL,
    tag        TEXT    NOT NULL
);

-- Backfill tag name from canonical registry.
INSERT INTO product_tags_old (id, product_id, position, tag)
SELECT pt.id, pt.product_id, pt.position, t.name
FROM product_tags pt
         JOIN tags t ON t.id = pt.tag_id;

DROP TABLE product_tags;
ALTER TABLE product_tags_old RENAME TO product_tags;

CREATE UNIQUE INDEX IF NOT EXISTS idx_product_tags_product_tag ON product_tags (product_id, tag);
CREATE INDEX IF NOT EXISTS idx_product_tags_product_id ON product_tags (product_id);
CREATE INDEX IF NOT EXISTS idx_product_tags_tag ON product_tags (tag);
