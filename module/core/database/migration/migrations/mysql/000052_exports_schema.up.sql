CREATE TABLE IF NOT EXISTS export_reports (
  id VARCHAR(36) NOT NULL,
  report_type VARCHAR(32) NOT NULL,
  status VARCHAR(32) NOT NULL,
  stamp VARCHAR(32) NOT NULL,
  file_name VARCHAR(255) NOT NULL,
  storage_key VARCHAR(512) NOT NULL,
  sha256_hash CHAR(64) NOT NULL,
  content_type VARCHAR(128) NOT NULL,
  row_count INT NOT NULL DEFAULT 0,
  byte_size BIGINT NOT NULL DEFAULT 0,
  generated_at DATETIME(3) NOT NULL,
  created_at DATETIME(3) NOT NULL,
  updated_at DATETIME(3) NOT NULL,
  PRIMARY KEY (id)
);

CREATE INDEX idx_export_reports_type_generated_at ON export_reports (report_type, generated_at DESC, id DESC);
CREATE INDEX idx_export_reports_generated_at ON export_reports (generated_at DESC, id DESC);
CREATE INDEX idx_export_reports_sha256_hash ON export_reports (sha256_hash);
