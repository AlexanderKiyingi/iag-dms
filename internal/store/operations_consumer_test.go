package store

import (
	"context"
	"testing"

	"github.com/iag/dms/backend/internal/models"
)

func TestVanLoadLinesFromPickEvent(t *testing.T) {
	lines := vanLoadLinesFromPickEvent(map[string]any{
		"lines": []any{
			map[string]any{"sku": "SKU-1", "qty": 4.0, "lot_key": "B1"},
			map[string]any{"itemId": "SKU-2", "quantity": 2.0},
			map[string]any{"sku": "", "qty": 9.0},
		},
	})
	if len(lines) != 2 || lines[0].ItemID != "SKU-1" || lines[0].Quantity != 4 || lines[0].Batch != "B1" {
		t.Fatalf("unexpected lines: %+v", lines)
	}
}

func TestEnsureVanLoadFromPickIdempotent(t *testing.T) {
	r, _, _ := deskRepo(t)
	ctx := context.Background()
	data := map[string]any{
		"vehicle_id": "UBK 999Z",
		"lines": []any{
			map[string]any{"sku": "SKU-1", "qty": 8.0, "batch": "PICK-LOT"},
		},
	}
	if err := r.ensureVanLoadFromPick(ctx, "evt-1", "PICK-100", "SO-9", data); err != nil {
		t.Fatal(err)
	}
	if err := r.ensureVanLoadFromPick(ctx, "evt-2", "PICK-100", "SO-9", data); err != nil {
		t.Fatal(err)
	}
	loads, _ := r.ListVanLoads(DocFilter{Limit: 100})
	var matched []models.VanLoad
	for _, load := range loads {
		if load.SourcePickListID == "PICK-100" {
			matched = append(matched, load)
		}
	}
	if len(matched) != 1 {
		t.Fatalf("expected one van load for pick, got %d (%+v)", len(matched), matched)
	}
	if matched[0].SourceOrderRef != "SO-9" || matched[0].Quantity != 8 || matched[0].VehicleID != "UBK 999Z" {
		t.Fatalf("van load not filled from pick: %+v", matched[0])
	}
}

func TestEnsureVanLoadFromPickPendingVehicle(t *testing.T) {
	r, _, _ := deskRepo(t)
	if err := r.ensureVanLoadFromPick(context.Background(), "evt-3", "PICK-200", "", map[string]any{
		"lines": []any{map[string]any{"item_id": "SKU-9", "picked_qty": 3.0}},
	}); err != nil {
		t.Fatal(err)
	}
	loads, _ := r.ListVanLoads(DocFilter{Limit: 100})
	found := false
	for _, load := range loads {
		if load.SourcePickListID == "PICK-200" {
			found = true
			if load.VehicleID != "pending-assign" {
				t.Fatalf("expected pending-assign vehicle, got %q", load.VehicleID)
			}
		}
	}
	if !found {
		t.Fatal("expected van load for PICK-200")
	}
}
