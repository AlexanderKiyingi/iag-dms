package store

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

// doc is what a typed-document table stores: an id plus the three key columns
// (status, outlet, ref) the service filters on. Everything else is JSON.
type doc interface {
	DocID() string
	DocKeys() (status, outletID, refID string)
}

// DocFilter is the query surface a typed-document list supports.
type DocFilter struct {
	Status   string
	OutletID string
	RefID    string
	Limit    int
	Offset   int
}

// docTable persists one document type. With a pool it is a Postgres table of
// (id, status, outlet_id, ref_id, doc JSONB); without one it is an in-memory
// map. Both keep insertion order so a list reads newest-first either way.
type docTable[T doc] struct {
	table string
	pool  *pgxpool.Pool

	mu    sync.RWMutex
	rows  map[string]T
	order []string
	seq   int
}

func newDocTable[T doc](pool *pgxpool.Pool, table string) *docTable[T] {
	return &docTable[T]{table: table, pool: pool, rows: map[string]T{}}
}

func (d *docTable[T]) list(ctx context.Context, f DocFilter) ([]T, int) {
	if f.Limit <= 0 {
		f.Limit = 500
	}
	if d.pool == nil {
		d.mu.RLock()
		defer d.mu.RUnlock()
		var all []T
		for i := len(d.order) - 1; i >= 0; i-- {
			row := d.rows[d.order[i]]
			status, outlet, ref := row.DocKeys()
			if (f.Status != "" && status != f.Status) || (f.OutletID != "" && outlet != f.OutletID) || (f.RefID != "" && ref != f.RefID) {
				continue
			}
			all = append(all, row)
		}
		return paginate(all, ListOpts{Limit: f.Limit, Offset: f.Offset})
	}
	rows, err := d.pool.Query(ctx, fmt.Sprintf(`
		SELECT doc FROM %s
		WHERE ($1 = '' OR status = $1) AND ($2 = '' OR outlet_id = $2) AND ($3 = '' OR ref_id = $3)
		ORDER BY created_at DESC, id DESC`, d.table), f.Status, f.OutletID, f.RefID)
	if err != nil {
		return nil, 0
	}
	defer rows.Close()
	var all []T
	for rows.Next() {
		var raw []byte
		if rows.Scan(&raw) != nil {
			continue
		}
		var row T
		if json.Unmarshal(raw, &row) == nil {
			all = append(all, row)
		}
	}
	return paginate(all, ListOpts{Limit: f.Limit, Offset: f.Offset})
}

func (d *docTable[T]) get(ctx context.Context, id string) (T, error) {
	var zero T
	if d.pool == nil {
		d.mu.RLock()
		defer d.mu.RUnlock()
		row, ok := d.rows[id]
		if !ok {
			return zero, ErrNotFound
		}
		return row, nil
	}
	var raw []byte
	if err := d.pool.QueryRow(ctx, fmt.Sprintf(`SELECT doc FROM %s WHERE id = $1`, d.table), id).Scan(&raw); err != nil {
		return zero, ErrNotFound
	}
	var row T
	if err := json.Unmarshal(raw, &row); err != nil {
		return zero, err
	}
	return row, nil
}

func (d *docTable[T]) insert(ctx context.Context, row T) error {
	status, outlet, ref := row.DocKeys()
	if d.pool == nil {
		d.mu.Lock()
		defer d.mu.Unlock()
		d.rows[row.DocID()] = row
		d.order = append(d.order, row.DocID())
		return nil
	}
	raw, err := json.Marshal(row)
	if err != nil {
		return err
	}
	_, err = d.pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s (id, status, outlet_id, ref_id, doc) VALUES ($1,$2,$3,$4,$5)`, d.table),
		row.DocID(), status, outlet, ref, raw)
	return err
}

func (d *docTable[T]) update(ctx context.Context, row T) error {
	status, outlet, ref := row.DocKeys()
	if d.pool == nil {
		d.mu.Lock()
		defer d.mu.Unlock()
		if _, ok := d.rows[row.DocID()]; !ok {
			return ErrNotFound
		}
		d.rows[row.DocID()] = row
		return nil
	}
	raw, err := json.Marshal(row)
	if err != nil {
		return err
	}
	tag, err := d.pool.Exec(ctx, fmt.Sprintf(`
		UPDATE %s SET status=$2, outlet_id=$3, ref_id=$4, doc=$5, updated_at=NOW() WHERE id=$1`, d.table),
		row.DocID(), status, outlet, ref, raw)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (d *docTable[T]) delete(ctx context.Context, id string) error {
	if d.pool == nil {
		d.mu.Lock()
		defer d.mu.Unlock()
		if _, ok := d.rows[id]; !ok {
			return ErrNotFound
		}
		delete(d.rows, id)
		for i, v := range d.order {
			if v == id {
				d.order = append(d.order[:i], d.order[i+1:]...)
				break
			}
		}
		return nil
	}
	tag, err := d.pool.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE id=$1`, d.table), id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// nextID mints a prefixed sequential id — the Postgres counter table when
// there is a pool, a per-table counter otherwise.
func (d *docTable[T]) nextID(ctx context.Context, r *Repository, prefix string) string {
	if d.pool != nil {
		id, _ := r.pgNextID(ctx, prefix)
		return id
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.seq++
	return fmt.Sprintf("%s-%05d", prefix, d.seq)
}

// sortByExpiry orders batches nearest-expiry first (FEFO); blanks last.
func sortByExpiry[T any](items []T, expiry func(T) string) {
	sort.SliceStable(items, func(i, j int) bool {
		a, b := expiry(items[i]), expiry(items[j])
		if a == "" {
			return false
		}
		if b == "" {
			return true
		}
		return a < b
	})
}
