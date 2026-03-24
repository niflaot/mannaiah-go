ALTER TABLE dispatch_batches DROP COLUMN name;
ALTER TABLE dispatch_batches ADD COLUMN created_by VARCHAR(255) NOT NULL DEFAULT 'system';
