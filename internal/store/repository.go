package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository persists DMS data in Postgres when pool is set; otherwise in-memory (local UI dev).
type Repository struct {
	pool *pgxpool.Pool
	mem  *memoryState
}

func New(pool *pgxpool.Pool) *Repository {
	r := &Repository{pool: pool}
	if pool == nil {
		r.mem = newMemoryState()
	}
	return r
}

func (r *Repository) Ping(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	return r.pool.Ping(ctx)
}

func (r *Repository) IsEmpty(ctx context.Context) (bool, error) {
	if r.pool == nil {
		return r.mem.isEmpty(), nil
	}
	var n int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM dms_distributors`).Scan(&n)
	return n == 0, err
}
