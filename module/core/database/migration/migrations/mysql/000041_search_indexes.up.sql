-- Search indexes for full-text search and filtered queries.
-- Each index is created conditionally to make this migration idempotent.

-- contacts
SET @q=(SELECT IF(COUNT(*)=0,'CREATE INDEX idx_contacts_first_name ON contacts (first_name)','SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='contacts' AND INDEX_NAME='idx_contacts_first_name');
PREPARE _s FROM @q; EXECUTE _s;
SET @q=(SELECT IF(COUNT(*)=0,'CREATE INDEX idx_contacts_last_name ON contacts (last_name)','SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='contacts' AND INDEX_NAME='idx_contacts_last_name');
PREPARE _s FROM @q; EXECUTE _s;
SET @q=(SELECT IF(COUNT(*)=0,'CREATE INDEX idx_contacts_email ON contacts (email)','SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='contacts' AND INDEX_NAME='idx_contacts_email');
PREPARE _s FROM @q; EXECUTE _s;
SET @q=(SELECT IF(COUNT(*)=0,'CREATE INDEX idx_contacts_document_number ON contacts (document_number)','SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='contacts' AND INDEX_NAME='idx_contacts_document_number');
PREPARE _s FROM @q; EXECUTE _s;
SET @q=(SELECT IF(COUNT(*)=0,'CREATE INDEX idx_contacts_phone ON contacts (phone)','SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='contacts' AND INDEX_NAME='idx_contacts_phone');
PREPARE _s FROM @q; EXECUTE _s;
SET @q=(SELECT IF(COUNT(*)=0,'CREATE INDEX idx_contacts_legal_name ON contacts (legal_name)','SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='contacts' AND INDEX_NAME='idx_contacts_legal_name');
PREPARE _s FROM @q; EXECUTE _s;

-- orders
SET @q=(SELECT IF(COUNT(*)=0,'CREATE INDEX idx_orders_identifier ON orders (identifier)','SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='orders' AND INDEX_NAME='idx_orders_identifier');
PREPARE _s FROM @q; EXECUTE _s;
SET @q=(SELECT IF(COUNT(*)=0,'CREATE INDEX idx_orders_realm ON orders (realm)','SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='orders' AND INDEX_NAME='idx_orders_realm');
PREPARE _s FROM @q; EXECUTE _s;
SET @q=(SELECT IF(COUNT(*)=0,'CREATE INDEX idx_orders_contact_id ON orders (contact_id)','SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='orders' AND INDEX_NAME='idx_orders_contact_id');
PREPARE _s FROM @q; EXECUTE _s;
SET @q=(SELECT IF(COUNT(*)=0,'CREATE INDEX idx_orders_payment_method ON orders (payment_method)','SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='orders' AND INDEX_NAME='idx_orders_payment_method');
PREPARE _s FROM @q; EXECUTE _s;

-- products
SET @q=(SELECT IF(COUNT(*)=0,'CREATE INDEX idx_products_sku ON products (sku)','SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='products' AND INDEX_NAME='idx_products_sku');
PREPARE _s FROM @q; EXECUTE _s;

-- categories
SET @q=(SELECT IF(COUNT(*)=0,'CREATE INDEX idx_categories_name ON categories (name)','SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='categories' AND INDEX_NAME='idx_categories_name');
PREPARE _s FROM @q; EXECUTE _s;
SET @q=(SELECT IF(COUNT(*)=0,'CREATE INDEX idx_categories_slug ON categories (slug)','SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='categories' AND INDEX_NAME='idx_categories_slug');
PREPARE _s FROM @q; EXECUTE _s;

-- shipping_marks
SET @q=(SELECT IF(COUNT(*)=0,'CREATE INDEX idx_shipping_marks_tracking_number ON shipping_marks (tracking_number)','SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='shipping_marks' AND INDEX_NAME='idx_shipping_marks_tracking_number');
PREPARE _s FROM @q; EXECUTE _s;
SET @q=(SELECT IF(COUNT(*)=0,'CREATE INDEX idx_shipping_marks_order_id ON shipping_marks (order_id)','SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='shipping_marks' AND INDEX_NAME='idx_shipping_marks_order_id');
PREPARE _s FROM @q; EXECUTE _s;
SET @q=(SELECT IF(COUNT(*)=0,'CREATE INDEX idx_shipping_marks_carrier_id ON shipping_marks (carrier_id)','SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='shipping_marks' AND INDEX_NAME='idx_shipping_marks_carrier_id');
PREPARE _s FROM @q; EXECUTE _s;
SET @q=(SELECT IF(COUNT(*)=0,'CREATE INDEX idx_shipping_marks_status ON shipping_marks (status)','SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='shipping_marks' AND INDEX_NAME='idx_shipping_marks_status');
PREPARE _s FROM @q; EXECUTE _s;
SET @q=(SELECT IF(COUNT(*)=0,'CREATE INDEX idx_shipping_marks_dispatch_batch_id ON shipping_marks (dispatch_batch_id)','SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='shipping_marks' AND INDEX_NAME='idx_shipping_marks_dispatch_batch_id');
PREPARE _s FROM @q; EXECUTE _s;

-- campaigns
SET @q=(SELECT IF(COUNT(*)=0,'CREATE INDEX idx_campaigns_name ON campaigns (name)','SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='campaigns' AND INDEX_NAME='idx_campaigns_name');
PREPARE _s FROM @q; EXECUTE _s;
SET @q=(SELECT IF(COUNT(*)=0,'CREATE INDEX idx_campaigns_slug ON campaigns (slug)','SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='campaigns' AND INDEX_NAME='idx_campaigns_slug');
PREPARE _s FROM @q; EXECUTE _s;
SET @q=(SELECT IF(COUNT(*)=0,'CREATE INDEX idx_campaigns_status ON campaigns (status)','SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='campaigns' AND INDEX_NAME='idx_campaigns_status');
PREPARE _s FROM @q; EXECUTE _s;
SET @q=(SELECT IF(COUNT(*)=0,'CREATE INDEX idx_campaigns_channel ON campaigns (channel)','SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='campaigns' AND INDEX_NAME='idx_campaigns_channel');
PREPARE _s FROM @q; EXECUTE _s;
SET @q=(SELECT IF(COUNT(*)=0,'CREATE INDEX idx_campaigns_segment_id ON campaigns (segment_id)','SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='campaigns' AND INDEX_NAME='idx_campaigns_segment_id');
PREPARE _s FROM @q; EXECUTE _s;

-- segments
SET @q=(SELECT IF(COUNT(*)=0,'CREATE INDEX idx_segments_name ON segments (name)','SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='segments' AND INDEX_NAME='idx_segments_name');
PREPARE _s FROM @q; EXECUTE _s;
SET @q=(SELECT IF(COUNT(*)=0,'CREATE INDEX idx_segments_slug ON segments (slug)','SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='segments' AND INDEX_NAME='idx_segments_slug');
PREPARE _s FROM @q; EXECUTE _s;
SET @q=(SELECT IF(COUNT(*)=0,'CREATE INDEX idx_segments_channel ON segments (channel)','SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='segments' AND INDEX_NAME='idx_segments_channel');
PREPARE _s FROM @q; EXECUTE _s;
SET @q=(SELECT IF(COUNT(*)=0,'CREATE INDEX idx_segments_parent_segment_id ON segments (parent_segment_id)','SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='segments' AND INDEX_NAME='idx_segments_parent_segment_id');
PREPARE _s FROM @q; EXECUTE _s;

-- tags
SET @q=(SELECT IF(COUNT(*)=0,'CREATE INDEX idx_tags_name ON tags (name)','SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='tags' AND INDEX_NAME='idx_tags_name');
PREPARE _s FROM @q; EXECUTE _s;

-- variations
SET @q=(SELECT IF(COUNT(*)=0,'CREATE INDEX idx_variations_name ON variations (name)','SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='variations' AND INDEX_NAME='idx_variations_name');
PREPARE _s FROM @q; EXECUTE _s;
SET @q=(SELECT IF(COUNT(*)=0,'CREATE INDEX idx_variations_value ON variations (value)','SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='variations' AND INDEX_NAME='idx_variations_value');
PREPARE _s FROM @q; EXECUTE _s;
SET @q=(SELECT IF(COUNT(*)=0,'CREATE INDEX idx_variations_definition ON variations (definition)','SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='variations' AND INDEX_NAME='idx_variations_definition');
PREPARE _s FROM @q; EXECUTE _s;

DEALLOCATE PREPARE _s;
