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

INSERT INTO dms_forecast_points (sku, point_date, forecast, actual, lower_bound, upper_bound)
VALUES
    ('BG-AA-250', '2026-05-26', 420, 398, NULL, NULL),
    ('BG-AA-250', '2026-06-02', 445, NULL, 410, 480)
ON CONFLICT DO NOTHING;

COMMIT;
