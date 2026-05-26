package store

import (
	"fmt"
	"strings"

	"github.com/iag/dms/backend/internal/models"
)

func (m *memoryState) patchOutlet(id string, patch models.OutletPatch) (models.Outlet, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, o := range m.outlets {
		if o.ID != id {
			continue
		}
		if patch.Name != "" {
			m.outlets[i].Name = patch.Name
		}
		if patch.Address != "" {
			m.outlets[i].Address = patch.Address
		}
		if patch.Channel != "" {
			m.outlets[i].Channel = patch.Channel
		}
		if patch.BeatID != "" {
			m.outlets[i].BeatID = patch.BeatID
		}
		if patch.Status != "" {
			m.outlets[i].Status = patch.Status
		}
		if patch.Score != "" {
			m.outlets[i].Score = patch.Score
		}
		if patch.Frequency != "" {
			m.outlets[i].Frequency = patch.Frequency
		}
		return m.outlets[i], nil
	}
	return models.Outlet{}, ErrNotFound
}

func (m *memoryState) getInvoice(id string) (models.Invoice, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, inv := range m.invoices {
		if inv.ID == id {
			return inv, nil
		}
	}
	return models.Invoice{}, ErrNotFound
}

func (m *memoryState) listVisitReports(opts ListOpts) ([]models.VisitReport, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	opts = defaultLimit(opts)
	var filtered []models.VisitReport
	for _, v := range m.visits {
		if opts.RepID != "" && v.RepID != opts.RepID {
			continue
		}
		filtered = append(filtered, v)
	}
	return paginate(filtered, opts)
}

func (m *memoryState) completeCheckIn(id string) (models.CheckIn, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, c := range m.checkIns {
		if c.ID != id {
			continue
		}
		m.checkIns[i].Status = "completed"
		for j, rep := range m.reps {
			if rep.ID == c.RepID && rep.Status == "clocked_in" {
				m.reps[j].Status = "active"
			}
		}
		return m.checkIns[i], nil
	}
	return models.CheckIn{}, ErrNotFound
}

func (m *memoryState) createClaim(in models.ClaimInput) models.Claim {
	m.mu.Lock()
	defer m.mu.Unlock()
	c := models.Claim{
		ID: fmt.Sprintf("CLM-%d", 1042+len(m.claims)), OutletID: in.OutletID,
		Type: in.Type, Status: "open", AmountUGX: in.AmountUGX, CreatedAt: now(),
	}
	m.claims = append([]models.Claim{c}, m.claims...)
	return c
}

func (m *memoryState) createPromotion(in models.PromotionInput) models.Promotion {
	m.mu.Lock()
	defer m.mu.Unlock()
	roi := in.ROI
	if roi == 0 {
		roi = 2.0
	}
	p := models.Promotion{
		ID: fmt.Sprintf("TPM-%03d", 25+len(m.promos)), Name: in.Name, SKU: in.SKU,
		ROI: roi, Status: "active", Outlets: in.Outlets,
	}
	m.promos = append([]models.Promotion{p}, m.promos...)
	return p
}

func (m *memoryState) createDispatch(in models.DispatchInput) models.Dispatch {
	m.mu.Lock()
	defer m.mu.Unlock()
	d := models.Dispatch{
		ID: fmt.Sprintf("DXP-%d", 2814+len(m.dispatches)), TruckID: in.TruckID, Driver: in.Driver,
		OrderIDs: in.OrderIDs, Status: "planned", ETA: in.ETA, UpdatedAt: now(),
	}
	if d.ETA == "" {
		d.ETA = "< 6h"
	}
	m.dispatches = append([]models.Dispatch{d}, m.dispatches...)
	return d
}

func (m *memoryState) runReport(in models.ReportRunInput) models.ReportRun {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = "Custom report"
	}
	return models.ReportRun{
		JobID: newUUID(), Name: name, Status: "queued",
		RowCount: 22, Message: "Report queued for generation and email delivery",
	}
}

func (m *memoryState) exportPage(page, format string) models.ExportPayload {
	if format == "" {
		format = "json"
	}
	rows := m.exportRows(page)
	return models.ExportPayload{
		Page: page, Format: format, GeneratedAt: now(),
		RowCount: len(rows), Rows: rows,
	}
}

func (m *memoryState) exportRows(page string) []map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	switch page {
	case "outlets":
		rows := make([]map[string]any, 0, len(m.outlets))
		for _, o := range m.outlets {
			rows = append(rows, map[string]any{
				"id": o.ID, "name": o.Name, "channel": o.Channel, "status": o.Status,
			})
		}
		return rows
	case "orders":
		rows := make([]map[string]any, 0, len(m.orders))
		for _, o := range m.orders {
			rows = append(rows, map[string]any{
				"id": o.ID, "outlet": o.OutletName, "status": o.Status, "amountUgx": o.AmountUGX,
			})
		}
		return rows
	case "invoices":
		rows := make([]map[string]any, 0, len(m.invoices))
		for _, inv := range m.invoices {
			rows = append(rows, map[string]any{
				"id": inv.ID, "distributor": inv.Distributor, "amountUgx": inv.AmountUGX, "status": inv.Status,
			})
		}
		return rows
	case "network":
		rows := make([]map[string]any, 0, len(m.distributors))
		for _, d := range m.distributors {
			rows = append(rows, map[string]any{
				"id": d.ID, "name": d.Name, "region": d.Region, "revenueUgx": d.RevenueUGX,
			})
		}
		return rows
	default:
		return []map[string]any{{"page": page, "note": "export snapshot"}}
	}
}
