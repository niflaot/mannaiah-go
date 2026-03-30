-- Rollback search indexes.

DROP INDEX IF EXISTS idx_contacts_first_name;
DROP INDEX IF EXISTS idx_contacts_last_name;
DROP INDEX IF EXISTS idx_contacts_email;
DROP INDEX IF EXISTS idx_contacts_document_number;
DROP INDEX IF EXISTS idx_contacts_phone;
DROP INDEX IF EXISTS idx_contacts_legal_name;

DROP INDEX IF EXISTS idx_orders_identifier;
DROP INDEX IF EXISTS idx_orders_realm;
DROP INDEX IF EXISTS idx_orders_contact_id;
DROP INDEX IF EXISTS idx_orders_payment_method;

DROP INDEX IF EXISTS idx_products_sku;

DROP INDEX IF EXISTS idx_categories_name;
DROP INDEX IF EXISTS idx_categories_slug;

DROP INDEX IF EXISTS idx_shipping_marks_tracking_number;
DROP INDEX IF EXISTS idx_shipping_marks_order_id;
DROP INDEX IF EXISTS idx_shipping_marks_carrier_id;
DROP INDEX IF EXISTS idx_shipping_marks_status;
DROP INDEX IF EXISTS idx_shipping_marks_dispatch_batch_id;

DROP INDEX IF EXISTS idx_campaigns_name;
DROP INDEX IF EXISTS idx_campaigns_slug;
DROP INDEX IF EXISTS idx_campaigns_status;
DROP INDEX IF EXISTS idx_campaigns_channel;
DROP INDEX IF EXISTS idx_campaigns_segment_id;

DROP INDEX IF EXISTS idx_segments_name;
DROP INDEX IF EXISTS idx_segments_slug;
DROP INDEX IF EXISTS idx_segments_channel;
DROP INDEX IF EXISTS idx_segments_parent_segment_id;

DROP INDEX IF EXISTS idx_tags_name;

DROP INDEX IF EXISTS idx_variations_name;
DROP INDEX IF EXISTS idx_variations_value;
DROP INDEX IF EXISTS idx_variations_definition;
