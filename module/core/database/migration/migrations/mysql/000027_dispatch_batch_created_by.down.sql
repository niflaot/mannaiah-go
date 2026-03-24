ALTER TABLE dispatch_batches DROP COLUMN created_by;
ALTER TABLE dispatch_batches ADD COLUMN name VARCHAR(255) NOT NULL DEFAULT '';
