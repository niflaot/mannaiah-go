ALTER TABLE shipping_marks
    MODIFY COLUMN draft_snapshot LONGTEXT NULL,
    ADD COLUMN response_snapshot LONGTEXT NULL AFTER draft_snapshot;
