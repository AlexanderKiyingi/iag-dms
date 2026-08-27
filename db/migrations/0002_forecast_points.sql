BEGIN;

CREATE TABLE IF NOT EXISTS dms_forecast_points (
    sku          TEXT NOT NULL REFERENCES dms_skus (code) ON DELETE CASCADE,
    point_date   DATE NOT NULL,
    forecast     NUMERIC(12, 2) NOT NULL DEFAULT 0,
    actual       NUMERIC(12, 2),
    lower_bound  NUMERIC(12, 2),
    upper_bound  NUMERIC(12, 2),
    PRIMARY KEY (sku, point_date)
);

-- These demo points hang off SKU BG-AA-250, which no migration creates: the SKU
-- catalogue is written by application seed code, which runs *after* migrations.
-- On a database that already had the catalogue the plain INSERT worked, but on a
-- fresh one it failed the foreign key and took the whole migration run with it,
-- so DMS could not be stood up from scratch at all.
--
-- Selecting through an EXISTS check keeps the original behaviour wherever the
-- SKU is present and makes this a no-op where it is not. The rows are demo data
-- either way — 0008_purge_demo_seed deletes BG-AA-250 and its points.
INSERT INTO dms_forecast_points (sku, point_date, forecast, actual, lower_bound, upper_bound)
SELECT v.sku, v.point_date, v.forecast, v.actual, v.lower_bound, v.upper_bound
  FROM (VALUES
      ('BG-AA-250'::TEXT, '2026-05-26'::DATE, 420::NUMERIC(12,2), 398::NUMERIC(12,2), NULL::NUMERIC(12,2), NULL::NUMERIC(12,2)),
      ('BG-AA-250',       '2026-06-02',       445,               NULL,                410,                 480)
  ) AS v (sku, point_date, forecast, actual, lower_bound, upper_bound)
 WHERE EXISTS (SELECT 1 FROM dms_skus s WHERE s.code = v.sku)
ON CONFLICT DO NOTHING;

COMMIT;
