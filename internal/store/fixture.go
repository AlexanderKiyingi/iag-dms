package store

import "github.com/iag/dms/backend/internal/models"

// Fixture is the reference data a test or a local dev session can preload into
// the in-memory repository, which otherwise starts empty.
type Fixture struct {
	Distributors []models.Distributor
	Beats        []models.Beat
	Reps         []models.FieldRep
	Outlets      []models.Outlet
}

// NewMemoryWith returns an in-memory repository populated from the fixture.
func NewMemoryWith(f Fixture) *Repository {
	r := New(nil)
	r.mem.mu.Lock()
	defer r.mem.mu.Unlock()
	r.mem.distributors = append(r.mem.distributors, f.Distributors...)
	r.mem.beats = append(r.mem.beats, f.Beats...)
	r.mem.reps = append(r.mem.reps, f.Reps...)
	r.mem.outlets = append(r.mem.outlets, f.Outlets...)
	return r
}
