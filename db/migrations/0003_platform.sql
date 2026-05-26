BEGIN;

CREATE TABLE IF NOT EXISTS dms_audit_entries (
    id         TEXT PRIMARY KEY,
    logged_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_name  TEXT NOT NULL DEFAULT '',
    action     TEXT NOT NULL,
    detail     TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS dms_audit_entries_logged_at_idx ON dms_audit_entries (logged_at DESC);

CREATE TABLE IF NOT EXISTS dms_api_audit (
    id          BIGSERIAL PRIMARY KEY,
    logged_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    method      TEXT NOT NULL,
    path        TEXT NOT NULL,
    status_code INT NOT NULL,
    user_name   TEXT NOT NULL DEFAULT '',
    duration_ms INT NOT NULL DEFAULT 0,
    client_ip   TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS dms_api_audit_logged_at_idx ON dms_api_audit (logged_at DESC);

CREATE TABLE IF NOT EXISTS dms_signals (
    id          TEXT PRIMARY KEY,
    kind        TEXT NOT NULL DEFAULT 'ops',
    entity_id   TEXT NOT NULL DEFAULT '',
    entity_name TEXT NOT NULL DEFAULT '',
    signal_type TEXT NOT NULL,
    strength    TEXT NOT NULL DEFAULT 'medium',
    action_hint TEXT NOT NULL DEFAULT '',
    observed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS dms_signals_observed_at_idx ON dms_signals (observed_at DESC);

INSERT INTO dms_id_counters (prefix, next_value) VALUES ('AUD', 1000) ON CONFLICT DO NOTHING;

INSERT INTO dms_signals (id, kind, entity_id, entity_name, signal_type, strength, action_hint, observed_at)
VALUES
    ('SIG-001', 'stock', 'D-001', 'Kampala Premium Beverages', 'Stock-out risk · Bugisu AA', 'high', 'Replen within 48h', NOW() - INTERVAL '2 hours'),
    ('SIG-002', 'invoice', 'INV-2418', 'Kampala Premium', 'Invoice due today', 'high', 'Collect or extend terms', NOW() - INTERVAL '1 hour'),
    ('SIG-003', 'field', 'FF-04', 'A. Achieng', 'Beat B-08 ahead of SLA', 'medium', 'Review journey adherence', NOW() - INTERVAL '30 minutes'),
    ('SIG-004', 'order', 'SO-19848', 'Shoprite Lugogo', 'Order in picking > 2h', 'medium', 'Expedite WMS pick', NOW() - INTERVAL '15 minutes')
ON CONFLICT DO NOTHING;

COMMIT;
