-- Purge the DMS prototype dataset written by store.SeedPostgres (internal/store/seed_pg.go).
--
-- Every row the seed wrote is named explicitly rather than matched by prefix: new
-- records draw their ids from dms_id_counters and land in the same namespace, so a
-- prefix match could not tell demo rows from an operator's.
--
-- Deliberately preserved:
--   * dms_id_counters (0003_platform.sql) — the id sequence state; resetting it would
--     hand new records ids that collide with anything created since.
--
-- The runner wraps each migration in its own transaction, so this file does not open
-- one: a COMMIT here would end that transaction early.

-- ---- join tables and leaf records first -----------------------------------
DELETE FROM dms_beat_outlets WHERE beat_id IN (
    'B-08', 'B-02', 'B-21'
);

DELETE FROM dms_dispatch_orders WHERE dispatch_id IN (
    'DXP-2814', 'DXP-2811'
);

DELETE FROM dms_check_ins WHERE id IN (
    'CI-9001'
);

DELETE FROM dms_alerts WHERE id IN (
    'ALT-1', 'ALT-2', 'ALT-3', 'ALT-4'
);

DELETE FROM dms_execution_tasks WHERE id IN (
    'EXE-1001', 'EXE-1002'
);

DELETE FROM dms_claims WHERE id IN (
    'CLM-1042', 'CLM-1038'
);

DELETE FROM dms_invoices WHERE id IN (
    'INV-2418', 'INV-2421', 'INV-2417'
);

DELETE FROM dms_stock_positions WHERE distributor_id IN (
    'D-001', 'D-007', 'D-014', 'D-022',
    'D-029', 'D-031'
);

-- ---- routing and field structures -----------------------------------------
DELETE FROM dms_dispatches WHERE id IN (
    'DXP-2814', 'DXP-2811'
);

DELETE FROM dms_orders WHERE id IN (
    'SO-19852', 'SO-19848', 'SO-19844', 'SO-19841',
    'SO-19836'
);

DELETE FROM dms_beats WHERE id IN (
    'B-08', 'B-02', 'B-21'
);

DELETE FROM dms_field_reps WHERE id IN (
    'FF-04', 'FF-02', 'FF-05', 'FF-09',
    'FF-13'
);

-- ---- catalogues and templates ---------------------------------------------
DELETE FROM dms_promotions WHERE id IN (
    'TPM-024', 'TPM-019'
);

DELETE FROM dms_pricing_templates WHERE id IN (
    'PT-001', 'PT-002', 'PT-003', 'PT-004'
);

DELETE FROM dms_report_templates WHERE id IN (
    'RPT-001', 'RPT-002', 'RPT-003'
);

DELETE FROM dms_skus WHERE code IN (
    'BG-AA-250', 'BG-AB-500', 'SF-NT-250', 'WN-RB-500',
    'HK-IN-100'
);

-- ---- signals seeded by 0003_platform.sql ----------------------------------
DELETE FROM dms_signals WHERE id IN ('SIG-001', 'SIG-002', 'SIG-003');

-- ---- parents last ----------------------------------------------------------
-- Several tables reference outlets and distributors without ON DELETE, so an order,
-- claim or check-in an operator has since raised against a demo outlet would block the
-- delete. Each parent is removed in its own subtransaction and kept if it is still
-- referenced, so the migration cannot fail on a foreign key.
DO $$
DECLARE
    demo_outlets TEXT[] := ARRAY[
        'OUT-00214', 'OUT-00318', 'OUT-00482', 'OUT-00694',
        'OUT-00821', 'OUT-01124', 'OUT-01408', 'OUT-01872',
        'OUT-02214'
    ];
    demo_distributors TEXT[] := ARRAY[
        'D-001', 'D-007', 'D-014', 'D-022',
        'D-029', 'D-031'
    ];
    row_id TEXT;
    kept   INT := 0;
BEGIN
    FOREACH row_id IN ARRAY demo_outlets
    LOOP
        BEGIN
            DELETE FROM dms_outlets WHERE id = row_id;
        EXCEPTION WHEN foreign_key_violation THEN
            kept := kept + 1;
            RAISE NOTICE 'dms_outlets % still referenced by live records — kept', row_id;
        END;
    END LOOP;
    FOREACH row_id IN ARRAY demo_distributors
    LOOP
        BEGIN
            DELETE FROM dms_distributors WHERE id = row_id;
        EXCEPTION WHEN foreign_key_violation THEN
            kept := kept + 1;
            RAISE NOTICE 'dms_distributors % still referenced by live records — kept', row_id;
        END;
    END LOOP;
    IF kept > 0 THEN
        RAISE NOTICE 'purge: % demo parent row(s) retained because live records reference them', kept;
    END IF;
END $$;
