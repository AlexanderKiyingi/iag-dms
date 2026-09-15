-- Secondary sales: the distributor → retailer side of the Distribution desk.
-- Van load (primary, no revenue) → secondary invoice (AR) → collection against
-- that invoice → outlet return → van/cash close. Each resource is a typed
-- document: the columns the service filters and joins on are real, the rest
-- of the record rides in `doc` so a screen field never needs a migration.

CREATE TABLE IF NOT EXISTS dms_van_loads (
    id         TEXT PRIMARY KEY,
    status     TEXT NOT NULL DEFAULT 'loaded',
    outlet_id  TEXT NOT NULL DEFAULT '',
    ref_id     TEXT NOT NULL DEFAULT '',          -- vehicle
    doc        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS dms_van_loads_status_idx ON dms_van_loads (status);

CREATE TABLE IF NOT EXISTS dms_secondary_invoices (
    id         TEXT PRIMARY KEY,
    status     TEXT NOT NULL DEFAULT 'invoiced',
    outlet_id  TEXT NOT NULL DEFAULT '',
    ref_id     TEXT NOT NULL DEFAULT '',          -- van load
    doc        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS dms_secondary_invoices_outlet_idx ON dms_secondary_invoices (outlet_id);
CREATE INDEX IF NOT EXISTS dms_secondary_invoices_ref_idx ON dms_secondary_invoices (ref_id);

CREATE TABLE IF NOT EXISTS dms_collections (
    id         TEXT PRIMARY KEY,
    status     TEXT NOT NULL DEFAULT 'collected',
    outlet_id  TEXT NOT NULL DEFAULT '',
    ref_id     TEXT NOT NULL DEFAULT '',          -- secondary invoice
    doc        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS dms_collections_ref_idx ON dms_collections (ref_id);

CREATE TABLE IF NOT EXISTS dms_outlet_returns (
    id         TEXT PRIMARY KEY,
    status     TEXT NOT NULL DEFAULT 'received',
    outlet_id  TEXT NOT NULL DEFAULT '',
    ref_id     TEXT NOT NULL DEFAULT '',          -- secondary invoice
    doc        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS dms_outlet_returns_ref_idx ON dms_outlet_returns (ref_id);

CREATE TABLE IF NOT EXISTS dms_van_recons (
    id         TEXT PRIMARY KEY,
    status     TEXT NOT NULL DEFAULT 'open',
    outlet_id  TEXT NOT NULL DEFAULT '',
    ref_id     TEXT NOT NULL DEFAULT '',          -- van load
    doc        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS dms_schemes (
    id         TEXT PRIMARY KEY,
    status     TEXT NOT NULL DEFAULT 'active',
    outlet_id  TEXT NOT NULL DEFAULT '',
    ref_id     TEXT NOT NULL DEFAULT '',
    doc        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS dms_rep_targets (
    id         TEXT PRIMARY KEY,
    status     TEXT NOT NULL DEFAULT 'active',
    outlet_id  TEXT NOT NULL DEFAULT '',
    ref_id     TEXT NOT NULL DEFAULT '',          -- rep
    doc        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS dms_damage_claims (
    id         TEXT PRIMARY KEY,
    status     TEXT NOT NULL DEFAULT 'open',
    outlet_id  TEXT NOT NULL DEFAULT '',
    ref_id     TEXT NOT NULL DEFAULT '',          -- outlet return
    doc        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
