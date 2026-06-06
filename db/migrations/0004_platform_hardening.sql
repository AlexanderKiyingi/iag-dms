BEGIN;

CREATE TABLE IF NOT EXISTS dms_event_outbox (
    id            BIGSERIAL PRIMARY KEY,
    event_type    TEXT NOT NULL,
    event_key     TEXT,
    payload       JSONB NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    available_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    dispatched_at TIMESTAMPTZ,
    attempts      INT NOT NULL DEFAULT 0,
    last_error    TEXT
);

CREATE INDEX IF NOT EXISTS dms_event_outbox_due_idx
    ON dms_event_outbox (available_at)
    WHERE dispatched_at IS NULL;

CREATE TABLE IF NOT EXISTS dms_report_jobs (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    template_id  TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'queued',
    row_count    INT NOT NULL DEFAULT 0,
    result       JSONB,
    email_to     TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS dms_report_jobs_created_idx ON dms_report_jobs (created_at DESC);

ALTER TABLE dms_outlets
    ADD COLUMN IF NOT EXISTS crm_ref TEXT;

CREATE INDEX IF NOT EXISTS dms_outlets_crm_ref_idx ON dms_outlets (crm_ref) WHERE crm_ref IS NOT NULL;

COMMIT;
