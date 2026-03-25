ALTER TABLE shipping_marks ADD COLUMN sender_legal_name TEXT NOT NULL DEFAULT '';
ALTER TABLE shipping_marks ADD COLUMN recipient_legal_name TEXT NOT NULL DEFAULT '';
