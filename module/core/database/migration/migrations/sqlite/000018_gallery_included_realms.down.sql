CREATE TABLE IF NOT EXISTS product_gallery_excluded_realms (
    id             INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    gallery_item_id INTEGER NOT NULL,
    position       INTEGER NOT NULL,
    realm          TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_product_gallery_excluded_realms_gallery_item_id ON product_gallery_excluded_realms (gallery_item_id);
CREATE INDEX IF NOT EXISTS idx_product_gallery_excluded_realms_position ON product_gallery_excluded_realms (position);

DROP TABLE IF EXISTS product_gallery_included_realms;
