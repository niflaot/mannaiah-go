ALTER TABLE segments ADD COLUMN parent_segment_id VARCHAR(36) NULL DEFAULT NULL AFTER channel;
ALTER TABLE segments ADD INDEX idx_segments_parent_segment_id (parent_segment_id);
