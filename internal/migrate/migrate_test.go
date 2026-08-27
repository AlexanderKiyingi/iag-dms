package migrate

import (
	"strings"
	"testing"

	"github.com/iag/dms/backend/db"
)

// Checks that do not need a database. They cannot prove the SQL runs — only
// Postgres can say that — but they can hold the checksum-heal table honest,
// which is the part that fails silently and dangerously.

func loadEmbedded(t *testing.T) map[string]Migration {
	t.Helper()
	migs, err := load(db.Migrations())
	if err != nil {
		t.Fatalf("load embedded migrations: %v", err)
	}
	if len(migs) == 0 {
		t.Fatal("no migrations are embedded")
	}
	out := make(map[string]Migration, len(migs))
	for _, m := range migs {
		out[m.Version] = m
	}
	return out
}

// An entry naming a version that no longer ships is dead weight, and worse, it
// hides a rename: the guard would go on protecting a file that is not there.
func TestSupersededChecksumsNameRealMigrations(t *testing.T) {
	embedded := loadEmbedded(t)
	for version := range supersededChecksums {
		if _, ok := embedded[version]; !ok {
			t.Errorf("supersededChecksums names %q, which is not an embedded migration", version)
		}
	}
}

// A superseded checksum that equals the current file's checksum means the entry
// is either stale or was recorded from the wrong revision. Either way it grants
// an exemption that can never fire, so the guard silently stops protecting that
// migration for the value someone thought they were healing.
func TestSupersededChecksumsAreNotTheCurrentChecksum(t *testing.T) {
	embedded := loadEmbedded(t)
	for version, sums := range supersededChecksums {
		m, ok := embedded[version]
		if !ok {
			continue // reported by the test above
		}
		if len(sums) == 0 {
			t.Errorf("%s has an empty superseded set; drop the entry instead", version)
		}
		for sum := range sums {
			if sum == m.Checksum {
				t.Errorf("%s lists its own current checksum %s as superseded", version, sum)
			}
		}
	}
}

// The defect this heal exists for: 0002 seeded demo forecast points against SKU
// BG-AA-250 with an unconditional INSERT. Nothing creates that SKU — the
// catalogue is written by application seed code, which runs after migrations —
// so the foreign key failed and DMS could not migrate from scratch at all.
//
// If the guard on dms_skus is ever removed, fresh databases break again and the
// only symptom is a failed deploy of a brand-new environment.
func TestForecastPointsSeedIsGuardedOnTheSKUExisting(t *testing.T) {
	m, ok := loadEmbedded(t)["0002_forecast_points"]
	if !ok {
		t.Fatal("0002_forecast_points is not embedded")
	}
	if !strings.Contains(m.Body, "dms_skus") {
		t.Fatal("0002 seeds dms_forecast_points without checking dms_skus: " +
			"a fresh database will fail the foreign key")
	}
	if !strings.Contains(m.Body, "WHERE EXISTS") {
		t.Error("expected the seed to select through a WHERE EXISTS guard")
	}
}
