CREATE TABLE IF NOT EXISTS export_reports (
  id TEXT NOT NULL PRIMARY KEY,
  report_type TEXT NOT NULL,
  status TEXT NOT NULL,
  stamp TEXT NOT NULL,
  file_name TEXT NOT NULL,
  storage_key TEXT NOT NULL,
  sha256_hash TEXT NOT NULL,
  content_type TEXT NOT NULL,
  row_count INTEGER NOT NULL DEFAULT 0,
  byte_size INTEGER NOT NULL DEFAULT 0,
  generated_at DATETIME NOT NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_export_reports_type_generated_at ON export_reports (report_type, generated_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_export_reports_generated_at ON export_reports (generated_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_export_reports_sha256_hash ON export_reports (sha256_hash);
