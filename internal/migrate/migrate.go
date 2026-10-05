package migrate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// This service owns the `dms` schema on the shared Railway database. The ledger
// is schema-qualified so it can never collide with another service's global
// public.schema_migrations. db.Connect pins search_path to `dms, public`.
const schemaMigrationsDDL = `
CREATE TABLE IF NOT EXISTS dms.schema_migrations (
    version    TEXT PRIMARY KEY,
    checksum   TEXT NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`

type Migration struct {
	Version  string
	Body     string
	Checksum string
}

func Up(ctx context.Context, pool *pgxpool.Pool, fsys fs.FS) ([]string, error) {
	migs, err := load(fsys)
	if err != nil {
		return nil, err
	}
	if _, err := pool.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS dms`); err != nil {
		return nil, fmt.Errorf("create schema dms: %w", err)
	}
	if _, err := pool.Exec(ctx, schemaMigrationsDDL); err != nil {
		return nil, fmt.Errorf("create schema_migrations: %w", err)
	}
	// One-time cutover from the shared global public.schema_migrations: stamp this
	// service's already-applied versions into the per-service ledger with their
	// current file checksums, so nothing re-runs and the checksum-mismatch guard
	// below cannot fire against tables that already exist in public.
	if err := seedFromLegacyLedger(ctx, pool, migs); err != nil {
		return nil, fmt.Errorf("seed from legacy ledger: %w", err)
	}
	applied, err := loadApplied(ctx, pool)
	if err != nil {
		return nil, err
	}
	var newlyApplied []string
	for _, m := range migs {
		prev, ok := applied[m.Version]
		switch {
		case !ok:
			if err := apply(ctx, pool, m); err != nil {
				return newlyApplied, fmt.Errorf("migration %s: %w", m.Version, err)
			}
			newlyApplied = append(newlyApplied, m.Version)
			slog.Info("migration applied", "version", m.Version)
		case prev.Checksum != m.Checksum:
			if !supersededChecksums[m.Version][prev.Checksum] {
				// Name both sums: the ledger's is the one an operator has to look up
				// to heal this, and without it the log line is a dead end.
				return newlyApplied, fmt.Errorf("migration %s checksum mismatch: ledger has %s, file is %s",
					m.Version, prev.Checksum, m.Checksum)
			}
			if err := restamp(ctx, pool, m); err != nil {
				return newlyApplied, fmt.Errorf("restamp %s: %w", m.Version, err)
			}
			slog.Warn("migration file was corrected in place; ledger re-stamped",
				"version", m.Version, "was", prev.Checksum, "now", m.Checksum)
		}
	}
	return newlyApplied, nil
}

// seedFromLegacyLedger stamps this service's shipped versions into dms's ledger
// using the CURRENT file checksums, for any version already recorded in a legacy
// global public.schema_migrations. Using the file checksum keeps the
// checksum-mismatch guard in Up from firing during the shared-database cutover —
// those objects already exist in public and resolve via the search_path
// fallback. Idempotent; no-op on a fresh database.
func seedFromLegacyLedger(ctx context.Context, pool *pgxpool.Pool, migs []Migration) error {
	var hasLegacy bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'schema_migrations'
		)`).Scan(&hasLegacy); err != nil {
		return err
	}
	if !hasLegacy {
		return nil
	}
	// public.schema_migrations is a SHARED ledger: every service that predates the
	// per-service cutover wrote its versions into it, unscoped. So a version string
	// found there does not necessarily belong to THIS service - '0001_initial' is
	// written by several of them. Seeding on a bare name match would stamp a
	// migration as applied that never ran here, and the tables it creates would
	// silently never exist.
	//
	// The cutover this function exists for only makes sense on a database that has
	// actually run DMS before, and such a database necessarily has DMS tables. A
	// database with none is either brand new or has never hosted DMS, and its rows
	// in the shared ledger belong to somebody else. Refuse to read them.
	var hasOwnTables bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'dms' AND table_name <> 'schema_migrations'
		)`).Scan(&hasOwnTables); err != nil {
		return err
	}
	if !hasOwnTables {
		return nil
	}
	for _, m := range migs {
		if _, err := pool.Exec(ctx, `
			INSERT INTO dms.schema_migrations (version, checksum)
			SELECT $1, $2
			WHERE EXISTS (SELECT 1 FROM public.schema_migrations WHERE version = $1)
			ON CONFLICT (version) DO NOTHING`, m.Version, m.Checksum); err != nil {
			return fmt.Errorf("seed %s: %w", m.Version, err)
		}
	}
	return nil
}

// supersededChecksums records migrations whose file was corrected in place after
// it had already been applied somewhere. The checksum guard above exists to catch
// a file being edited under a database that has already run it, which is almost
// always a mistake — so an entry here is deliberate, names the exact prior
// checksum, and heals only that one value. Any other drift still fails.
//
//	0002_forecast_points seeded demo forecast points against SKU BG-AA-250 with a
//	plain INSERT. Nothing creates that SKU — the catalogue is written by
//	application seed code, which runs after migrations — so the foreign key failed
//	and DMS could not migrate from scratch at all. The INSERT now selects through
//	an EXISTS check: unchanged wherever the SKU is present, a no-op where it is
//	not. Databases that already applied the original file keep their rows and are
//	simply re-stamped.
//	0001_initial has one commit in this repository, dated 2026-05-26, and has
//	never been edited since. Production nevertheless recorded a different
//	checksum when it applied the file on 2026-08-27, so the build that ran it
//	carried a copy that is no longer reachable from any branch here — a
//	rewritten or force-pushed history, most likely.
//
//	The guard then did exactly its job and refused to migrate, which took the
//	whole service down: DMS crash-looped from 2026-10-03, every endpoint
//	answered 502, and 0010_outlet_commercial and 0011_secondary_sales never
//	ran — which is why creating an outlet, a van load or a collection failed.
//
//	Healing is safe here because it is bookkeeping, not re-execution: restamp
//	never re-runs the body, and every one of the twenty tables 0001_initial
//	defines already exists in the live schema with the columns it declares. The
//	database is what the current file would have produced; only the recorded
//	sum disagreed.
var supersededChecksums = map[string]map[string]bool{
	"0001_initial": {
		"3560eb31ab05a82ae2b37794c126a0e8aa23ac93a70fd4ace84649f98f06f8c8": true,
	},
	// The same lost history, four more versions deep. Healing 0001 moved the
	// migrator on to 0003 and it stopped there; each of these has one commit
	// here and has never been edited, and each recorded a different sum in
	// production. Listing them together rather than one per deploy, because a
	// crash-looping service costs a deploy cycle per discovery.
	//
	// Every table and column these four define was checked against the live
	// dms schema first: 3 tables for 0003, 3 tables and 5 columns for 0005,
	// 1 table for 0006, 3 columns for 0007 — all present. The database is
	// what these files would have produced.
	"0003_platform": {
		"3dfcad061b8488330635497cd031a3dd4cf4579e00f2d934d9f36f1fabfc4776": true,
	},
	"0005_domain_writes": {
		"4b920d0f3899955f6bca01c4b604d07abf07f3d81fd929a1a3327f2cc5711d08": true,
	},
	"0006_attachments": {
		"27dd06860816be0f8e2a1b6166ee8d7ad9424d308d74ab3165ef5431b239fefc": true,
	},
	"0007_invoice_fiscal": {
		"5e79a72b0b9301e8663231214da56adc1076078fcbf98bb52b02d33e306de080": true,
	},
	"0002_forecast_points": {
		"c23c47ebca091f5a31a8cb3c88a31625e5d1a5a4c789095f1d2935ea7fb79aee": true,
	},
}

// restamp records the current file checksum for a version that is already
// applied. It never re-runs the migration body.
func restamp(ctx context.Context, pool *pgxpool.Pool, m Migration) error {
	_, err := pool.Exec(ctx,
		`UPDATE dms.schema_migrations SET checksum = $2 WHERE version = $1`,
		m.Version, m.Checksum)
	return err
}

func load(fsys fs.FS) ([]Migration, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}
	var out []Migration
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".sql") {
			continue
		}
		body, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(body)
		out = append(out, Migration{
			Version:  strings.TrimSuffix(name, ".sql"),
			Body:     string(body),
			Checksum: hex.EncodeToString(sum[:]),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}

type appliedRow struct {
	Version  string
	Checksum string
}

func loadApplied(ctx context.Context, pool *pgxpool.Pool) (map[string]appliedRow, error) {
	rows, err := pool.Query(ctx, `SELECT version, checksum FROM dms.schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]appliedRow{}
	for rows.Next() {
		var r appliedRow
		if err := rows.Scan(&r.Version, &r.Checksum); err != nil {
			return nil, err
		}
		out[r.Version] = r
	}
	return out, rows.Err()
}

func apply(ctx context.Context, pool *pgxpool.Pool, m Migration) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, m.Body); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO dms.schema_migrations (version, checksum) VALUES ($1, $2)`,
		m.Version, m.Checksum); err != nil {
		if strings.Contains(err.Error(), "23505") {
			return errors.New("concurrent migration")
		}
		return err
	}
	return tx.Commit(ctx)
}
