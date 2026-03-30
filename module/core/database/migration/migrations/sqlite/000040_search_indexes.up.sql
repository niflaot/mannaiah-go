-- Search indexes for full-text search and filtered queries.

CREATE INDEX IF NOT EXISTS idx_contacts_first_name ON contacts (first_name);
CREATE INDEX IF NOT EXISTS idx_contacts_last_name ON contacts (last_name);
CREATE INDEX IF NOT EXISTS idx_contacts_email ON contacts (email);
CREATE INDEX IF NOT EXISTS idx_contacts_document_number ON contacts (document_number);
CREATE INDEX IF NOT EXISTS idx_contacts_phone ON contacts (phone);
CREATE INDEX IF NOT EXISTS idx_contacts_legal_name ON contacts (legal_name);

CREATE INDEX IF NOT EXISTS idx_orders_identifier ON orders (identifier);
CREATE INDEX IF NOT EXISTS idx_orders_realm ON orders (realm);
CREATE INDEX IF NOT EXISTS idx_orders_contact_id ON orders (contact_id);
CREATE INDEX IF NOT EXISTS idx_orders_payment_method ON orders (payment_method);

CREATE INDEX IF NOT EXISTS idx_products_sku ON products (sku);

CREATE INDEX IF NOT EXISTS idx_categories_name ON categories (name);
CREATE INDEX IF NOT EXISTS idx_categories_slug ON categories (slug);

CREATE INDEX IF NOT EXISTS idx_shipping_marks_tracking_number ON shipping_marks (tracking_number);
CREATE INDEX IF NOT EXISTS idx_shipping_marks_order_id ON shipping_marks (order_id);
CREATE INDEX IF NOT EXISTS idx_shipping_marks_carrier_id ON shipping_marks (carrier_id);
CREATE INDEX IF NOT EXISTS idx_shipping_marks_status ON shipping_marks (status);
CREATE INDEX IF NOT EXISTS idx_shipping_marks_dispatch_batch_id ON shipping_marks (dispatch_batch_id);

CREATE INDEX IF NOT EXISTS idx_campaigns_name ON campaigns (name);
CREATE INDEX IF NOT EXISTS idx_campaigns_slug ON campaigns (slug);
CREATE INDEX IF NOT EXISTS idx_campaigns_status ON campaigns (status);
CREATE INDEX IF NOT EXISTS idx_campaigns_channel ON campaigns (channel);
CREATE INDEX IF NOT EXISTS idx_campaigns_segment_id ON campaigns (segment_id);

CREATE INDEX IF NOT EXISTS idx_segments_name ON segments (name);
CREATE INDEX IF NOT EXISTS idx_segments_slug ON segments (slug);
CREATE INDEX IF NOT EXISTS idx_segments_channel ON segments (channel);
CREATE INDEX IF NOT EXISTS idx_segments_parent_segment_id ON segments (parent_segment_id);

CREATE INDEX IF NOT EXISTS idx_tags_name ON tags (name);

CREATE INDEX IF NOT EXISTS idx_variations_name ON variations (name);
CREATE INDEX IF NOT EXISTS idx_variations_value ON variations (value);
CREATE INDEX IF NOT EXISTS idx_variations_definition ON variations (definition);
