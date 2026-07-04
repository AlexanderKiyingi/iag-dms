-- Wave 3: binary attachments (retail-execution photos, document uploads).
CREATE TABLE IF NOT EXISTS dms_attachments (
    id           TEXT PRIMARY KEY,
    owner_type   TEXT NOT NULL DEFAULT '',
    owner_id     TEXT NOT NULL DEFAULT '',
    filename     TEXT NOT NULL DEFAULT '',
    content_type TEXT NOT NULL DEFAULT 'application/octet-stream',
    size_bytes   BIGINT NOT NULL DEFAULT 0,
    storage_key  TEXT NOT NULL,
    uploaded_by  TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_dms_attachments_owner ON dms_attachments (owner_type, owner_id);
