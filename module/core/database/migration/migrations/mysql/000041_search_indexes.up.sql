-- Search indexes for full-text search and filtered queries.
-- These indexes accelerate the LIKE-based text search and the filter/sort paths
-- used by the unified search endpoints (/search, /<resource>/search).

-- contacts: text search on name, email, document_number, phone
CREATE INDEX idx_contacts_first_name ON contacts (first_name);
CREATE INDEX idx_contacts_last_name ON contacts (last_name);
CREATE INDEX idx_contacts_email ON contacts (email);
CREATE INDEX idx_contacts_document_number ON contacts (document_number);
CREATE INDEX idx_contacts_phone ON contacts (phone);
CREATE INDEX idx_contacts_legal_name ON contacts (legal_name);

-- orders: text search on identifier, filter on realm, contact_id, payment_method
CREATE INDEX idx_orders_identifier ON orders (identifier);
CREATE INDEX idx_orders_realm ON orders (realm);
CREATE INDEX idx_orders_contact_id ON orders (contact_id);
CREATE INDEX idx_orders_payment_method ON orders (payment_method);

-- products: text search on sku, filter on price, created_at
CREATE INDEX idx_products_sku ON products (sku);

-- categories: text search on name, slug, description
CREATE INDEX idx_categories_name ON categories (name);
CREATE INDEX idx_categories_slug ON categories (slug);

-- shipping_marks: text search on tracking_number, order_id; filter on carrier_id, status
CREATE INDEX idx_shipping_marks_tracking_number ON shipping_marks (tracking_number);
CREATE INDEX idx_shipping_marks_order_id ON shipping_marks (order_id);
CREATE INDEX idx_shipping_marks_carrier_id ON shipping_marks (carrier_id);
CREATE INDEX idx_shipping_marks_status ON shipping_marks (status);
CREATE INDEX idx_shipping_marks_dispatch_batch_id ON shipping_marks (dispatch_batch_id);

-- campaigns: text search on name, slug, subject; filter on status, channel, segment_id
CREATE INDEX idx_campaigns_name ON campaigns (name);
CREATE INDEX idx_campaigns_slug ON campaigns (slug);
CREATE INDEX idx_campaigns_status ON campaigns (status);
CREATE INDEX idx_campaigns_channel ON campaigns (channel);
CREATE INDEX idx_campaigns_segment_id ON campaigns (segment_id);

-- segments: text search on name, slug; filter on channel, parent_segment_id
CREATE INDEX idx_segments_name ON segments (name);
CREATE INDEX idx_segments_slug ON segments (slug);
CREATE INDEX idx_segments_channel ON segments (channel);
CREATE INDEX idx_segments_parent_segment_id ON segments (parent_segment_id);

-- tags: text search on name
CREATE INDEX idx_tags_name ON tags (name);

-- variations: text search on name, value; filter on definition
CREATE INDEX idx_variations_name ON variations (name);
CREATE INDEX idx_variations_value ON variations (value);
CREATE INDEX idx_variations_definition ON variations (definition);
