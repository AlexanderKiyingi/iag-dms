package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/iag/dms/backend/internal/models"
)

// ApplyOperationsEvent updates DMS state from iag.operations envelopes.
func (r *Repository) ApplyOperationsEvent(ctx context.Context, eventID, eventType string, raw json.RawMessage) error {
	if r.pool == nil {
		return nil
	}
	var data map[string]any
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &data)
	}
	switch eventType {
	case "warehouse.pick.confirmed":
		return r.applyWarehousePickConfirmed(ctx, eventID, data)
	default:
		return nil
	}
}

func (r *Repository) applyWarehousePickConfirmed(ctx context.Context, eventID string, data map[string]any) error {
	orderRef := stringField(data, "order_ref", "orderRef", "order_id", "orderId")
	pickID := stringField(data, "pick_list_id", "pickListId", "pick_id", "pickId")
	if orderRef != "" {
		tag, err := r.pool.Exec(ctx, `
			UPDATE dms_orders SET status = 'delivery', updated_at = NOW()
			WHERE id = $1 AND status = 'picking'`, orderRef)
		if err != nil {
			return err
		}
		if tag.RowsAffected() > 0 {
			detail := fmt.Sprintf("Warehouse pick confirmed for order %s — ready for dispatch", orderRef)
			if err := r.insertCommercialSignal(ctx, eventID, "warehouse", orderRef, orderRef, "pick.confirmed", "medium", detail); err != nil {
				return err
			}
		}
	}
	if pickID == "" {
		return nil
	}
	return r.ensureVanLoadFromPick(ctx, eventID, pickID, orderRef, data)
}

// ensureVanLoadFromPick creates a van load from a confirmed warehouse pick when
// one does not already reference that pick. Idempotent on sourcePickListId.
// Vehicle may be pending-assign when the pick event has no plate — ops fill it
// before the van leaves. This is the pick→van bridge without a full GI ledger.
func (r *Repository) ensureVanLoadFromPick(ctx context.Context, eventID, pickID, orderRef string, data map[string]any) error {
	existing, _ := r.ListVanLoads(DocFilter{Limit: 5000})
	for _, load := range existing {
		if strings.EqualFold(strings.TrimSpace(load.SourcePickListID), pickID) {
			return nil
		}
	}
	lines := vanLoadLinesFromPickEvent(data)
	if len(lines) == 0 {
		return nil
	}
	vehicleID := stringField(data, "vehicle_id", "vehicleId", "van_id", "vanId", "plate")
	if vehicleID == "" {
		vehicleID = "pending-assign"
	}
	load, err := r.CreateVanLoad(models.VanLoad{
		Date:             time.Now().UTC().Format("2006-01-02"),
		VehicleID:        vehicleID,
		Driver:           stringField(data, "driver", "driver_name", "driverName"),
		Lines:            lines,
		SourcePickListID: pickID,
		SourceOrderRef:   orderRef,
		Status:           "loaded",
		Notes:            fmt.Sprintf("Auto-created from warehouse pick %s", pickID),
	})
	if err != nil {
		return err
	}
	detail := fmt.Sprintf("Van load %s created from warehouse pick %s", load.ID, pickID)
	return r.insertCommercialSignal(ctx, eventID+"-van", "warehouse", load.ID, orderRef, "van.load.from_pick", "medium", detail)
}

func vanLoadLinesFromPickEvent(data map[string]any) []models.VanLoadLine {
	raw, ok := data["lines"].([]any)
	if !ok || len(raw) == 0 {
		return nil
	}
	var lines []models.VanLoadLine
	for _, row := range raw {
		m, ok := row.(map[string]any)
		if !ok {
			continue
		}
		itemID := stringField(m, "sku", "item_id", "itemId", "item")
		qty := floatField(m, "qty", "quantity", "picked_qty", "pickedQty")
		if itemID == "" || qty <= 0 {
			continue
		}
		lines = append(lines, models.VanLoadLine{
			ItemID:     itemID,
			Quantity:   qty,
			Batch:      stringField(m, "lot_key", "lotKey", "batch"),
			ExpiryDate: stringField(m, "expiry", "expiry_date", "expiryDate"),
		})
	}
	return lines
}

func floatField(data map[string]any, keys ...string) float64 {
	for _, key := range keys {
		v, ok := data[key]
		if !ok || v == nil {
			continue
		}
		switch n := v.(type) {
		case float64:
			return n
		case float32:
			return float64(n)
		case int:
			return float64(n)
		case int64:
			return float64(n)
		case json.Number:
			f, _ := n.Float64()
			return f
		case string:
			var f float64
			fmt.Sscanf(strings.TrimSpace(n), "%f", &f)
			return f
		}
	}
	return 0
}
