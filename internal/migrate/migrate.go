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
			return newlyApplied, fmt.Errorf("migration %s checksum mismatch", m.Version)
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
