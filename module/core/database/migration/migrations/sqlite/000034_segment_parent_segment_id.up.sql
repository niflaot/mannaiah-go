ALTER TABLE segments ADD COLUMN parent_segment_id TEXT NULL DEFAULT NULL;
CREATE INDEX idx_segments_parent_segment_id ON segments (parent_segment_id);
