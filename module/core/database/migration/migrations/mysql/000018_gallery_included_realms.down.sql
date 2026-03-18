CREATE TABLE IF NOT EXISTS product_gallery_excluded_realms (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    gallery_item_id BIGINT UNSIGNED NOT NULL,
    position INT NOT NULL,
    realm VARCHAR(128) NOT NULL,
    PRIMARY KEY (id),
    KEY idx_product_gallery_excluded_realms_gallery_item_id (gallery_item_id),
    KEY idx_product_gallery_excluded_realms_position (position)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

DROP TABLE IF EXISTS product_gallery_included_realms;
