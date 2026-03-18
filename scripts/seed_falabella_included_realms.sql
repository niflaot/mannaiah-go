-- seed_falabella_included_realms.sql
--
-- Adds every existing gallery image to the 'falabella' included realm so that
-- all pre-existing images remain visible in Falabella after the
-- excludedRealms → includedRealms migration (000018).
--
-- Run once against production after deploying migration 000018.

INSERT INTO product_gallery_included_realms (gallery_item_id, position, realm)
SELECT id, 0, 'falabella'
FROM product_gallery_items;
