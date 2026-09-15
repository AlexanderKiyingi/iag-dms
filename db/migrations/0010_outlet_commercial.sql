-- Outlet commercial profile: the retailer-facing fields the Distribution desk
-- keeps (contact, credit, KYC, geofence radius, pricing tier) and an attrs bag
-- for app-side references such as KYC attachment ids.
ALTER TABLE dms_outlets ADD COLUMN IF NOT EXISTS contact          TEXT  NOT NULL DEFAULT '';
ALTER TABLE dms_outlets ADD COLUMN IF NOT EXISTS phone            TEXT  NOT NULL DEFAULT '';
ALTER TABLE dms_outlets ADD COLUMN IF NOT EXISTS radius_m         DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE dms_outlets ADD COLUMN IF NOT EXISTS credit_limit_ugx DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE dms_outlets ADD COLUMN IF NOT EXISTS payment_terms    TEXT  NOT NULL DEFAULT '';
ALTER TABLE dms_outlets ADD COLUMN IF NOT EXISTS price_list       TEXT  NOT NULL DEFAULT '';
ALTER TABLE dms_outlets ADD COLUMN IF NOT EXISTS segment          TEXT  NOT NULL DEFAULT '';
ALTER TABLE dms_outlets ADD COLUMN IF NOT EXISTS volume_tier      TEXT  NOT NULL DEFAULT '';
ALTER TABLE dms_outlets ADD COLUMN IF NOT EXISTS kyc_status       TEXT  NOT NULL DEFAULT 'pending';
ALTER TABLE dms_outlets ADD COLUMN IF NOT EXISTS license_expiry   DATE;
ALTER TABLE dms_outlets ADD COLUMN IF NOT EXISTS notes            TEXT  NOT NULL DEFAULT '';
ALTER TABLE dms_outlets ADD COLUMN IF NOT EXISTS outstanding_ugx  DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE dms_outlets ADD COLUMN IF NOT EXISTS attrs            JSONB NOT NULL DEFAULT '{}'::jsonb;
CREATE INDEX IF NOT EXISTS dms_outlets_kyc_idx ON dms_outlets (kyc_status);
