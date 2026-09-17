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

// DevFixture is what a local memory-mode instance is preloaded with when
// MEMORY_FIXTURE=true: one distributor for retailers to be filed under, two
// beats and the reps who walk them.
func DevFixture() Fixture {
	return Fixture{
		Distributors: []models.Distributor{{ID: "D-001", Name: "Kampala Premium Beverages", Region: "Kampala", Status: "active"}},
		Beats: []models.Beat{
			{ID: "BT-01", Name: "Nakawa Tuesday", RepID: "FF-01", RepName: "Aisha Namara", StopCount: 12, DistanceKm: 18.5, Status: "active"},
			{ID: "BT-02", Name: "Ntinda Thursday", RepID: "FF-02", RepName: "Okello Brian", StopCount: 9, DistanceKm: 11.2, Status: "active"},
		},
		Reps: []models.FieldRep{
			{ID: "FF-01", Name: "Aisha Namara", BeatID: "BT-01", Region: "Kampala", Level: "senior", Status: "active"},
			{ID: "FF-02", Name: "Okello Brian", BeatID: "BT-02", Region: "Kampala", Level: "junior", Status: "active"},
		},
	}
}
