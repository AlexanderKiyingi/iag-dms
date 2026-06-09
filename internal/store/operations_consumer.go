package store

import (
	"context"
	"encoding/json"
	"fmt"
)

// ApplyOperationsEvent updates DMS state from iag.operations envelopes.
func (r *Repository) ApplyOperationsEvent(ctx context.Context, eventType string, raw json.RawMessage) error {
	if r.pool == nil {
		return nil
	}
	var data map[string]any
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &data)
	}
	switch eventType {
	case "warehouse.pick.confirmed":
		return r.applyWarehousePickConfirmed(ctx, data)
	default:
		return nil
	}
}

func (r *Repository) applyWarehousePickConfirmed(ctx context.Context, data map[string]any) error {
	orderRef := stringField(data, "order_ref", "orderRef", "order_id", "orderId")
	if orderRef == "" {
		return nil
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE dms_orders SET status = 'delivery', updated_at = NOW()
		WHERE id = $1 AND status = 'picking'`, orderRef)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	detail := fmt.Sprintf("Warehouse pick confirmed for order %s — ready for dispatch", orderRef)
	return r.insertCommercialSignal(ctx, "warehouse", orderRef, orderRef, "pick.confirmed", "medium", detail)
}
