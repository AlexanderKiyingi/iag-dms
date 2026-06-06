package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ApplyCommercialEvent updates DMS state from an iag.commercial envelope.
func (r *Repository) ApplyCommercialEvent(ctx context.Context, eventType string, raw json.RawMessage) error {
	if r.pool == nil {
		return nil
	}
	var data map[string]any
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &data)
	}
	switch eventType {
	case "crm.deal.won":
		return r.applyCRMDealWon(ctx, data)
	case "crm.lead.converted":
		return r.applyCRMLeadConverted(ctx, data)
	case "crm.outlet.synced":
		return r.applyCRMOutletSynced(ctx, data)
	default:
		return nil
	}
}

func (r *Repository) applyCRMDealWon(ctx context.Context, data map[string]any) error {
	dealID := stringField(data, "deal_id", "dealId", "id")
	account := stringField(data, "account", "account_name", "accountName")
	amount := stringField(data, "amount", "value")
	detail := fmt.Sprintf("Deal won · %s · %s", account, amount)
	if dealID != "" {
		detail = fmt.Sprintf("Deal %s won · %s · %s", dealID, account, amount)
	}
	return r.insertCommercialSignal(ctx, "commercial", dealID, account, "deal.won", "high", detail)
}

func (r *Repository) applyCRMLeadConverted(ctx context.Context, data map[string]any) error {
	leadID := stringField(data, "lead_id", "leadId", "id")
	name := stringField(data, "name", "company")
	detail := fmt.Sprintf("Lead converted · %s — assign beat coverage", name)
	return r.insertCommercialSignal(ctx, "commercial", leadID, name, "lead.converted", "medium", detail)
}

func (r *Repository) applyCRMOutletSynced(ctx context.Context, data map[string]any) error {
	crmRef := stringField(data, "crm_account_id", "account_id", "accountId", "crm_ref", "crmRef")
	dmsRef := stringField(data, "dms_ref", "dmsRef", "outlet_id", "outletId", "id")
	name := stringField(data, "name", "outlet_name", "outletName")
	if dmsRef != "" && crmRef != "" {
		_, err := r.pool.Exec(ctx, `
			UPDATE dms_outlets SET crm_ref = $2 WHERE id = $1 OR crm_ref = $2
		`, dmsRef, crmRef)
		if err != nil {
			return err
		}
	}
	detail := fmt.Sprintf("CRM outlet synced · %s", name)
	if crmRef != "" {
		detail = fmt.Sprintf("CRM account %s linked to DMS %s", crmRef, dmsRef)
	}
	return r.insertCommercialSignal(ctx, "bridge", dmsRef, name, "outlet.synced", "medium", detail)
}

func (r *Repository) insertCommercialSignal(ctx context.Context, kind, entityID, entityName, signalType, strength, hint string) error {
	id := "SIG-" + strings.ToUpper(uuid.NewString()[:8])
	_, err := r.pool.Exec(ctx, `
		INSERT INTO dms_signals (id, kind, entity_id, entity_name, signal_type, strength, action_hint, observed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,NOW())
		ON CONFLICT (id) DO NOTHING
	`, id, kind, entityID, entityName, signalType, strength, hint)
	return err
}

func stringField(data map[string]any, keys ...string) string {
	if data == nil {
		return ""
	}
	for _, k := range keys {
		if v, ok := data[k]; ok {
			switch t := v.(type) {
			case string:
				if strings.TrimSpace(t) != "" {
					return strings.TrimSpace(t)
				}
			case float64:
				return fmt.Sprintf("%.0f", t)
			}
		}
	}
	return ""
}
