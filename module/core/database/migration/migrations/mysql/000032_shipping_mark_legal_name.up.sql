ALTER TABLE shipping_marks ADD COLUMN sender_legal_name VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE shipping_marks ADD COLUMN recipient_legal_name VARCHAR(255) NOT NULL DEFAULT '';
