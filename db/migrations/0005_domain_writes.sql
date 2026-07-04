-- Wave 2: advertised domain writes — journey assignment, pricing lifecycle, report scheduling.

-- Journey Planner: assign a beat to a rep on a given day, ordered by seq.
CREATE TABLE IF NOT EXISTS dms_journey_assignments (
    id         TEXT PRIMARY KEY,
    rep_id     TEXT NOT NULL,
    date       DATE NOT NULL,
    beat_id    TEXT NOT NULL,
    seq        INT  NOT NULL DEFAULT 0,
    status     TEXT NOT NULL DEFAULT 'planned',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_dms_journey_rep_date ON dms_journey_assignments (rep_id, date);

-- Pricing lifecycle: draft -> pending -> approved, with immutable version history.
ALTER TABLE dms_pricing_templates ADD COLUMN IF NOT EXISTS status      TEXT        NOT NULL DEFAULT 'approved';
ALTER TABLE dms_pricing_templates ADD COLUMN IF NOT EXISTS created_by  TEXT        NOT NULL DEFAULT '';
ALTER TABLE dms_pricing_templates ADD COLUMN IF NOT EXISTS approved_by TEXT        NOT NULL DEFAULT '';
ALTER TABLE dms_pricing_templates ADD COLUMN IF NOT EXISTS body        JSONB       NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE dms_pricing_templates ADD COLUMN IF NOT EXISTS updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE TABLE IF NOT EXISTS dms_pricing_versions (
    id          TEXT PRIMARY KEY,
    template_id TEXT NOT NULL,
    version     TEXT NOT NULL,
    body        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by  TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_dms_pricing_versions_tpl ON dms_pricing_versions (template_id);

-- Report scheduling: persisted delivery jobs run by the background scheduler.
CREATE TABLE IF NOT EXISTS dms_report_schedules (
    id          TEXT PRIMARY KEY,
    template_id TEXT NOT NULL DEFAULT '',
    name        TEXT NOT NULL DEFAULT '',
    cron        TEXT NOT NULL DEFAULT '',
    channel     TEXT NOT NULL DEFAULT 'email',
    recipient   TEXT NOT NULL DEFAULT '',
    active      BOOLEAN NOT NULL DEFAULT TRUE,
    last_run_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '1 day',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_dms_report_schedules_due ON dms_report_schedules (active, next_run_at);
