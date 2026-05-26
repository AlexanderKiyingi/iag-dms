package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/iag/dms/backend/internal/models"
)

type apiAuditRow struct {
	Method     string
	Path       string
	StatusCode int
	UserName   string
	DurationMs int
	ClientIP   string
	LoggedAt   time.Time
}

func (m *memoryState) appendAudit(action, detail, userName string) models.AuditEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	e := models.AuditEntry{
		ID: fmt.Sprintf("AUD-%d", 1000+len(m.auditEntries)), Action: action, Detail: detail,
		UserName: userName, LoggedAt: now(),
	}
	m.auditEntries = append([]models.AuditEntry{e}, m.auditEntries...)
	return e
}

func (m *memoryState) listAudit(limit int) ([]models.AuditEntry, int) {
	m.mu.RLock()
	defer m.mu.Unlock()
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	total := len(m.auditEntries)
	if limit > total {
		limit = total
	}
	return append([]models.AuditEntry(nil), m.auditEntries[:limit]...), total
}

func (m *memoryState) getAudit(id string) (models.AuditEntry, error) {
	m.mu.RLock()
	defer m.mu.Unlock()
	for _, e := range m.auditEntries {
		if e.ID == id {
			return e, nil
		}
	}
	return models.AuditEntry{}, ErrNotFound
}

func (m *memoryState) logAPIRequest(method, path string, status int, user string, dur int, ip string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.apiAudit = append([]apiAuditRow{{
		Method: method, Path: path, StatusCode: status, UserName: user,
		DurationMs: dur, ClientIP: ip, LoggedAt: now(),
	}}, m.apiAudit...)
	if len(m.apiAudit) > 500 {
		m.apiAudit = m.apiAudit[:500]
	}
}

func (m *memoryState) listAPIAudit(limit int) ([]map[string]any, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	total := len(m.apiAudit)
	if limit > total {
		limit = total
	}
	out := make([]map[string]any, 0, limit)
	for i := 0; i < limit; i++ {
		r := m.apiAudit[i]
		out = append(out, map[string]any{
			"method": r.Method, "path": r.Path, "status": r.StatusCode,
			"user": r.UserName, "duration_ms": r.DurationMs, "logged_at": r.LoggedAt,
		})
	}
	return out, total
}

func (m *memoryState) monitoringSummary(busEnabled bool) map[string]any {
	m.mu.RLock()
	defer m.mu.Unlock()
	var total, errors int
	var sumDur int
	cutoff := now().Add(-24 * time.Hour)
	for _, r := range m.apiAudit {
		if r.LoggedAt.Before(cutoff) {
			continue
		}
		total++
		sumDur += r.DurationMs
		if r.StatusCode >= 400 {
			errors++
		}
	}
	avg := 0.0
	if total > 0 {
		avg = float64(sumDur) / float64(total)
	}
	openClaims := 0
	for _, c := range m.claims {
		if c.Status == "open" {
			openClaims++
		}
	}
	return map[string]any{
		"requests_24h": total, "errors_24h": errors, "avg_duration_ms": avg,
		"open_claims": openClaims, "outlets": len(m.outlets), "orders": len(m.orders),
		"event_bus_enabled": busEnabled, "store": "memory",
	}
}

func (m *memoryState) listSignals(limit int) []map[string]any {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	m.mu.RLock()
	defer m.mu.Unlock()
	if len(m.signals) > 0 {
		if limit > len(m.signals) {
			limit = len(m.signals)
		}
		return append([]map[string]any(nil), m.signals[:limit]...)
	}
	return defaultDMSSignals()
}

func defaultDMSSignals() []map[string]any {
	return []map[string]any{
		{"id": "SIG-001", "kind": "stock", "entity": "D-001", "signal": "Stock-out risk · Bugisu AA", "strength": "high", "action": "Replen within 48h"},
		{"id": "SIG-002", "kind": "invoice", "entity": "INV-2418", "signal": "Invoice due today", "strength": "high", "action": "Collect or extend terms"},
		{"id": "SIG-003", "kind": "field", "entity": "FF-04", "signal": "Beat ahead of SLA", "strength": "medium", "action": "Review journey adherence"},
	}
}

func (r *Repository) AppendAudit(ctx context.Context, action, detail, userName string) (models.AuditEntry, error) {
	if r.pool != nil {
		return r.pgAppendAudit(ctx, action, detail, userName)
	}
	return r.mem.appendAudit(action, detail, userName), nil
}

func (r *Repository) ListAudit(ctx context.Context, limit int) ([]models.AuditEntry, int, error) {
	if r.pool != nil {
		return r.pgListAudit(ctx, limit)
	}
	items, total := r.mem.listAudit(limit)
	return items, total, nil
}

func (r *Repository) GetAudit(ctx context.Context, id string) (models.AuditEntry, error) {
	if r.pool != nil {
		return r.pgGetAudit(ctx, id)
	}
	return r.mem.getAudit(id)
}

func (r *Repository) LogAPIRequest(ctx context.Context, method, path string, statusCode int, userName string, durationMs int, clientIP string) error {
	if r.pool != nil {
		return r.pgLogAPIRequest(ctx, method, path, statusCode, userName, durationMs, clientIP)
	}
	r.mem.logAPIRequest(method, path, statusCode, userName, durationMs, clientIP)
	return nil
}

func (r *Repository) ListAPIAuditLogs(ctx context.Context, limit int) ([]map[string]any, int, error) {
	if r.pool != nil {
		return r.pgListAPIAuditLogs(ctx, limit)
	}
	items, total := r.mem.listAPIAudit(limit)
	return items, total, nil
}

func (r *Repository) MonitoringSummaryWithBus(ctx context.Context, busEnabled bool) (map[string]any, error) {
	if r.pool != nil {
		return r.pgMonitoringSummary(ctx, busEnabled)
	}
	return r.mem.monitoringSummary(busEnabled), nil
}

func (r *Repository) ListSignals(ctx context.Context, limit int) ([]map[string]any, error) {
	if r.pool != nil {
		return r.pgListSignals(ctx, limit)
	}
	return r.mem.listSignals(limit), nil
}

func (r *Repository) Lookups(ctx context.Context, kind string) ([]map[string]any, error) {
	switch kind {
	case "distributors":
		items, _ := r.ListDistributors(ListOpts{Limit: 200})
		out := make([]map[string]any, 0, len(items))
		for _, d := range items {
			out = append(out, map[string]any{"id": d.ID, "label": d.Name, "region": d.Region})
		}
		return out, nil
	case "outlets":
		items, _ := r.ListOutlets(ListOpts{Limit: 200})
		out := make([]map[string]any, 0, len(items))
		for _, o := range items {
			out = append(out, map[string]any{"id": o.ID, "label": o.Name, "channel": o.Channel})
		}
		return out, nil
	case "reps":
		items, _ := r.ListReps(ListOpts{Limit: 100})
		out := make([]map[string]any, 0, len(items))
		for _, rep := range items {
			out = append(out, map[string]any{"id": rep.ID, "label": rep.Name, "region": rep.Region})
		}
		return out, nil
	case "beats":
		items, _ := r.ListBeats(ListOpts{Limit: 100})
		out := make([]map[string]any, 0, len(items))
		for _, b := range items {
			out = append(out, map[string]any{"id": b.ID, "label": b.Name, "repId": b.RepID})
		}
		return out, nil
	case "skus":
		items, _ := r.ListSKUs(ListOpts{Limit: 100})
		out := make([]map[string]any, 0, len(items))
		for _, s := range items {
			out = append(out, map[string]any{"id": s.Code, "label": s.Name})
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unknown lookup kind: %s", kind)
	}
}

func AuditDetail(entityType, entityID, verb string) string {
	return fmt.Sprintf("%s %s %s", verb, entityType, entityID)
}

func (r *Repository) pgAppendAudit(ctx context.Context, action, detail, userName string) (models.AuditEntry, error) {
	id, err := r.pgNextID(ctx, "AUD")
	if err != nil {
		id = "AUD-" + uuid.NewString()[:8]
	}
	now := time.Now().UTC()
	_, err = r.pool.Exec(ctx, `
		INSERT INTO dms_audit_entries (id, logged_at, user_name, action, detail) VALUES ($1,$2,$3,$4,$5)`,
		id, now, userName, action, detail)
	if err != nil {
		return models.AuditEntry{}, err
	}
	return models.AuditEntry{ID: id, LoggedAt: now, UserName: userName, Action: action, Detail: detail}, nil
}

func (r *Repository) pgListAudit(ctx context.Context, limit int) ([]models.AuditEntry, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM dms_audit_entries`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, logged_at, user_name, action, detail FROM dms_audit_entries ORDER BY logged_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []models.AuditEntry
	for rows.Next() {
		var e models.AuditEntry
		if rows.Scan(&e.ID, &e.LoggedAt, &e.UserName, &e.Action, &e.Detail) == nil {
			out = append(out, e)
		}
	}
	return out, total, rows.Err()
}

func (r *Repository) pgGetAudit(ctx context.Context, id string) (models.AuditEntry, error) {
	var e models.AuditEntry
	err := r.pool.QueryRow(ctx, `
		SELECT id, logged_at, user_name, action, detail FROM dms_audit_entries WHERE id = $1`, id).
		Scan(&e.ID, &e.LoggedAt, &e.UserName, &e.Action, &e.Detail)
	if err == pgx.ErrNoRows {
		return e, ErrNotFound
	}
	return e, err
}

func (r *Repository) pgLogAPIRequest(ctx context.Context, method, path string, statusCode int, userName string, durationMs int, clientIP string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO dms_api_audit (method, path, status_code, user_name, duration_ms, client_ip)
		VALUES ($1,$2,$3,$4,$5,$6)`, method, path, statusCode, userName, durationMs, clientIP)
	return err
}

func (r *Repository) pgListAPIAuditLogs(ctx context.Context, limit int) ([]map[string]any, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM dms_api_audit`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT method, path, status_code, user_name, duration_ms, logged_at
		FROM dms_api_audit ORDER BY logged_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var method, path, user string
		var status, dur int
		var at time.Time
		if rows.Scan(&method, &path, &status, &user, &dur, &at) == nil {
			out = append(out, map[string]any{
				"method": method, "path": path, "status": status,
				"user": user, "duration_ms": dur, "logged_at": at,
			})
		}
	}
	return out, total, rows.Err()
}

func (r *Repository) pgMonitoringSummary(ctx context.Context, busEnabled bool) (map[string]any, error) {
	var total24h, errors24h, outlets, orders, openClaims int
	var avgMs float64
	err := r.pool.QueryRow(ctx, `
		SELECT
			COUNT(*)::int,
			COUNT(*) FILTER (WHERE status_code >= 400)::int,
			COALESCE(AVG(duration_ms), 0)
		FROM dms_api_audit WHERE logged_at >= NOW() - INTERVAL '24 hours'`).Scan(&total24h, &errors24h, &avgMs)
	if err != nil {
		return nil, err
	}
	_ = r.pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM dms_outlets`).Scan(&outlets)
	_ = r.pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM dms_orders`).Scan(&orders)
	_ = r.pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM dms_claims WHERE status = 'open'`).Scan(&openClaims)
	return map[string]any{
		"requests_24h": total24h, "errors_24h": errors24h, "avg_duration_ms": avgMs,
		"open_claims": openClaims, "outlets": outlets, "orders": orders,
		"event_bus_enabled": busEnabled, "store": "postgres",
	}, nil
}

func (r *Repository) pgListSignals(ctx context.Context, limit int) ([]map[string]any, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, kind, entity_id, entity_name, signal_type, strength, action_hint, observed_at
		FROM dms_signals ORDER BY observed_at DESC LIMIT $1`, limit)
	if err != nil {
		return defaultDMSSignals(), nil
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id, kind, entID, entName, sig, strength, action string
		var at time.Time
		if rows.Scan(&id, &kind, &entID, &entName, &sig, &strength, &action, &at) == nil {
			out = append(out, map[string]any{
				"id": id, "kind": kind, "entity_id": entID, "entity": entName,
				"signal": sig, "strength": strength, "action": action, "observed_at": at,
			})
		}
	}
	if len(out) == 0 {
		return defaultDMSSignals(), nil
	}
	return out, rows.Err()
}
