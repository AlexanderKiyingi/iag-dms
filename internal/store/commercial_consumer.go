package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ApplyCommercialEvent updates DMS state from an iag.commercial envelope.
func (r *Repository) ApplyCommercialEvent(ctx context.Context, eventID, eventType string, raw json.RawMessage) error {
	if r.pool == nil {
		return nil
	}
	var data map[string]any
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &data)
	}
	switch eventType {
	case "crm.deal.won":
		return r.applyCRMDealWon(ctx, eventID, data)
	case "crm.lead.converted":
		return r.applyCRMLeadConverted(ctx, eventID, data)
	case "crm.outlet.synced":
		return r.applyCRMOutletSynced(ctx, eventID, data)
	default:
		return nil
	}
}

func (r *Repository) applyCRMDealWon(ctx context.Context, eventID string, data map[string]any) error {
	dealID := stringField(data, "deal_id", "dealId", "id")
	account := stringField(data, "account", "account_name", "accountName")
	amount := stringField(data, "amount", "value")
	detail := fmt.Sprintf("Deal won · %s · %s", account, amount)
	if dealID != "" {
		detail = fmt.Sprintf("Deal %s won · %s · %s", dealID, account, amount)
	}
	return r.insertCommercialSignal(ctx, eventID, "commercial", dealID, account, "deal.won", "high", detail)
}

func (r *Repository) applyCRMLeadConverted(ctx context.Context, eventID string, data map[string]any) error {
	leadID := stringField(data, "lead_id", "leadId", "id")
	name := stringField(data, "name", "company")
	detail := fmt.Sprintf("Lead converted · %s — assign beat coverage", name)
	return r.insertCommercialSignal(ctx, eventID, "commercial", leadID, name, "lead.converted", "medium", detail)
}

func (r *Repository) applyCRMOutletSynced(ctx context.Context, eventID string, data map[string]any) error {
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
	return r.insertCommercialSignal(ctx, eventID, "bridge", dmsRef, name, "outlet.synced", "medium", detail)
}

// signalID derives a stable id from the event that produced the signal.
//
// This used to mint a fresh uuid on every call, which made the ON CONFLICT
// below unreachable: a redelivered event inserted another row rather than
// colliding with the one it already wrote. There is no dedupe table in this
// consumer, so redelivery is the normal case after any handler failure, and
// duplicate signals accumulated for the same fact.
//
// An event with no id keeps the old random behaviour — nothing else identifies
// it, so it cannot be recognised on the way back round.
func signalID(eventID string) string {
	if strings.TrimSpace(eventID) == "" {
		return "SIG-" + strings.ToUpper(uuid.NewString()[:8])
	}
	sum := sha256.Sum256([]byte(eventID))
	return "SIG-" + strings.ToUpper(hex.EncodeToString(sum[:])[:8])
}

func (r *Repository) insertCommercialSignal(ctx context.Context, eventID, kind, entityID, entityName, signalType, strength, hint string) error {
	id := signalID(eventID)
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
