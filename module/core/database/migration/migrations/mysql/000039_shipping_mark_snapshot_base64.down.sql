ALTER TABLE shipping_marks
    DROP COLUMN response_snapshot,
    MODIFY COLUMN draft_snapshot TEXT NULL;
