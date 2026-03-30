-- Rollback search indexes.

DROP INDEX idx_contacts_first_name ON contacts;
DROP INDEX idx_contacts_last_name ON contacts;
DROP INDEX idx_contacts_email ON contacts;
DROP INDEX idx_contacts_document_number ON contacts;
DROP INDEX idx_contacts_phone ON contacts;
DROP INDEX idx_contacts_legal_name ON contacts;

DROP INDEX idx_orders_identifier ON orders;
DROP INDEX idx_orders_realm ON orders;
DROP INDEX idx_orders_contact_id ON orders;
DROP INDEX idx_orders_payment_method ON orders;

DROP INDEX idx_products_sku ON products;

DROP INDEX idx_categories_name ON categories;
DROP INDEX idx_categories_slug ON categories;

DROP INDEX idx_shipping_marks_tracking_number ON shipping_marks;
DROP INDEX idx_shipping_marks_order_id ON shipping_marks;
DROP INDEX idx_shipping_marks_carrier_id ON shipping_marks;
DROP INDEX idx_shipping_marks_status ON shipping_marks;
DROP INDEX idx_shipping_marks_dispatch_batch_id ON shipping_marks;

DROP INDEX idx_campaigns_name ON campaigns;
DROP INDEX idx_campaigns_slug ON campaigns;
DROP INDEX idx_campaigns_status ON campaigns;
DROP INDEX idx_campaigns_channel ON campaigns;
DROP INDEX idx_campaigns_segment_id ON campaigns;

DROP INDEX idx_segments_name ON segments;
DROP INDEX idx_segments_slug ON segments;
DROP INDEX idx_segments_channel ON segments;
DROP INDEX idx_segments_parent_segment_id ON segments;

DROP INDEX idx_tags_name ON tags;

DROP INDEX idx_variations_name ON variations;
DROP INDEX idx_variations_value ON variations;
DROP INDEX idx_variations_definition ON variations;
