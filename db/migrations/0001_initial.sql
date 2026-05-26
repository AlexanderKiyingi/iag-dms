BEGIN;

CREATE TABLE IF NOT EXISTS dms_id_counters (
    prefix     TEXT PRIMARY KEY,
    next_value BIGINT NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS dms_distributors (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    tier            INT NOT NULL DEFAULT 1,
    region          TEXT NOT NULL DEFAULT '',
    manager         TEXT NOT NULL DEFAULT '',
    outlets         INT NOT NULL DEFAULT 0,
    sell_in_rate    NUMERIC(5, 2) NOT NULL DEFAULT 0,
    revenue_ugx     NUMERIC(18, 2) NOT NULL DEFAULT 0,
    status          TEXT NOT NULL DEFAULT 'active',
    onboarded_at    TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS dms_outlets (
    id               TEXT PRIMARY KEY,
    name             TEXT NOT NULL,
    address          TEXT NOT NULL DEFAULT '',
    channel          TEXT NOT NULL,
    distributor_id   TEXT NOT NULL REFERENCES dms_distributors (id),
    beat_id          TEXT NOT NULL DEFAULT '',
    qtd_value_ugx    NUMERIC(18, 2) NOT NULL DEFAULT 0,
    frequency        TEXT NOT NULL DEFAULT '1x/wk',
    score            TEXT NOT NULL DEFAULT 'B',
    status           TEXT NOT NULL DEFAULT 'active',
    lat              DOUBLE PRECISION,
    lng              DOUBLE PRECISION,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS dms_outlets_distributor_idx ON dms_outlets (distributor_id);
CREATE INDEX IF NOT EXISTS dms_outlets_channel_idx ON dms_outlets (channel);
CREATE INDEX IF NOT EXISTS dms_outlets_status_idx ON dms_outlets (status);

CREATE TABLE IF NOT EXISTS dms_orders (
    id              TEXT PRIMARY KEY,
    outlet_id       TEXT NOT NULL REFERENCES dms_outlets (id),
    outlet_name     TEXT NOT NULL DEFAULT '',
    distributor_id  TEXT NOT NULL REFERENCES dms_distributors (id),
    rep_id          TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'draft',
    amount_ugx      NUMERIC(18, 2) NOT NULL DEFAULT 0,
    currency        TEXT NOT NULL DEFAULT 'UGX',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS dms_orders_status_idx ON dms_orders (status);

CREATE TABLE IF NOT EXISTS dms_beats (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    rep_id       TEXT NOT NULL DEFAULT '',
    rep_name     TEXT NOT NULL DEFAULT '',
    stop_count   INT NOT NULL DEFAULT 0,
    distance_km  NUMERIC(8, 2) NOT NULL DEFAULT 0,
    status       TEXT NOT NULL DEFAULT 'planned',
    progress     TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS dms_beat_outlets (
    beat_id   TEXT NOT NULL REFERENCES dms_beats (id) ON DELETE CASCADE,
    outlet_id TEXT NOT NULL REFERENCES dms_outlets (id) ON DELETE CASCADE,
    seq       INT NOT NULL DEFAULT 0,
    PRIMARY KEY (beat_id, outlet_id)
);

CREATE TABLE IF NOT EXISTS dms_field_reps (
    id       TEXT PRIMARY KEY,
    name     TEXT NOT NULL,
    beat_id  TEXT NOT NULL DEFAULT '',
    region   TEXT NOT NULL DEFAULT '',
    level    TEXT NOT NULL DEFAULT '',
    status   TEXT NOT NULL DEFAULT 'active'
);

CREATE TABLE IF NOT EXISTS dms_check_ins (
    id         TEXT PRIMARY KEY,
    rep_id     TEXT NOT NULL,
    outlet_id  TEXT NOT NULL REFERENCES dms_outlets (id),
    lat        DOUBLE PRECISION NOT NULL DEFAULT 0,
    lng        DOUBLE PRECISION NOT NULL DEFAULT 0,
    arrived_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status     TEXT NOT NULL DEFAULT 'active'
);

CREATE TABLE IF NOT EXISTS dms_visit_reports (
    id         TEXT PRIMARY KEY,
    rep_id     TEXT NOT NULL,
    outlet_id  TEXT NOT NULL REFERENCES dms_outlets (id),
    outcome    TEXT NOT NULL DEFAULT '',
    notes      TEXT NOT NULL DEFAULT '',
    lat        DOUBLE PRECISION NOT NULL DEFAULT 0,
    lng        DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS dms_promotions (
    id      TEXT PRIMARY KEY,
    name    TEXT NOT NULL,
    sku     TEXT NOT NULL DEFAULT '',
    roi     NUMERIC(6, 2) NOT NULL DEFAULT 0,
    status  TEXT NOT NULL DEFAULT 'active',
    outlets INT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS dms_claims (
    id          TEXT PRIMARY KEY,
    outlet_id   TEXT NOT NULL REFERENCES dms_outlets (id),
    claim_type  TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'open',
    amount_ugx  NUMERIC(18, 2) NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS dms_dispatches (
    id         TEXT PRIMARY KEY,
    truck_id   TEXT NOT NULL DEFAULT '',
    driver     TEXT NOT NULL DEFAULT '',
    status     TEXT NOT NULL DEFAULT 'planned',
    eta        TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS dms_dispatch_orders (
    dispatch_id TEXT NOT NULL REFERENCES dms_dispatches (id) ON DELETE CASCADE,
    order_id    TEXT NOT NULL REFERENCES dms_orders (id) ON DELETE CASCADE,
    PRIMARY KEY (dispatch_id, order_id)
);

CREATE TABLE IF NOT EXISTS dms_skus (
    code        TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    cover_days  NUMERIC(6, 2) NOT NULL DEFAULT 0,
    qty_on_hand INT NOT NULL DEFAULT 0,
    warehouse   TEXT NOT NULL DEFAULT '',
    sca_score   NUMERIC(5, 2) NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS dms_stock_positions (
    distributor_id TEXT NOT NULL REFERENCES dms_distributors (id),
    sku            TEXT NOT NULL REFERENCES dms_skus (code),
    cover_days     NUMERIC(6, 2) NOT NULL DEFAULT 0,
    qty            INT NOT NULL DEFAULT 0,
    status         TEXT NOT NULL DEFAULT 'ok',
    PRIMARY KEY (distributor_id, sku)
);

CREATE TABLE IF NOT EXISTS dms_invoices (
    id               TEXT PRIMARY KEY,
    distributor_id   TEXT NOT NULL REFERENCES dms_distributors (id),
    distributor_name TEXT NOT NULL DEFAULT '',
    amount_ugx       NUMERIC(18, 2) NOT NULL DEFAULT 0,
    due_date         DATE NOT NULL,
    status           TEXT NOT NULL DEFAULT 'open',
    order_id         TEXT REFERENCES dms_orders (id)
);

CREATE TABLE IF NOT EXISTS dms_pricing_templates (
    id       TEXT PRIMARY KEY,
    name     TEXT NOT NULL,
    channel  TEXT NOT NULL,
    version  TEXT NOT NULL DEFAULT '',
    currency TEXT NOT NULL DEFAULT 'UGX'
);

CREATE TABLE IF NOT EXISTS dms_report_templates (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    data_source TEXT NOT NULL,
    schedule    TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS dms_execution_tasks (
    id        TEXT PRIMARY KEY,
    outlet_id TEXT NOT NULL REFERENCES dms_outlets (id),
    task_type TEXT NOT NULL,
    status    TEXT NOT NULL DEFAULT 'pending',
    detail    TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS dms_alerts (
    id     TEXT PRIMARY KEY,
    kind   TEXT NOT NULL,
    title  TEXT NOT NULL,
    detail TEXT NOT NULL DEFAULT ''
);

COMMIT;
