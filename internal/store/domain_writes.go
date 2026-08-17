package store

import (
	"context"
	"fmt"
	"time"

	"github.com/iag/dms/backend/internal/models"
)

// ============================================================================
// Journey assignments
// ============================================================================

func (r *Repository) ListJourneyAssignments(repID, date string) []models.JourneyAssignment {
	if r.pool != nil {
		return r.pgListJourneyAssignments(r.bg(), repID, date)
	}
	return r.mem.listJourneyAssignments(repID, date)
}

func (r *Repository) CreateJourneyAssignment(in models.JourneyAssignmentInput) (models.JourneyAssignment, error) {
	if r.pool != nil {
		return r.pgCreateJourneyAssignment(r.bg(), in)
	}
	return r.mem.createJourneyAssignment(in)
}

func (r *Repository) PatchJourneyAssignment(id string, patch models.JourneyAssignmentPatch) (models.JourneyAssignment, error) {
	if r.pool != nil {
		return r.pgPatchJourneyAssignment(r.bg(), id, patch)
	}
	return r.mem.patchJourneyAssignment(id, patch)
}

func (r *Repository) DeleteJourneyAssignment(id string) error {
	if r.pool != nil {
		return r.pgDelete(r.bg(), "dms_journey_assignments", id)
	}
	return r.mem.deleteJourneyAssignment(id)
}

func (r *Repository) pgListJourneyAssignments(ctx context.Context, repID, date string) []models.JourneyAssignment {
	rows, err := r.pool.Query(ctx, `
		SELECT id, rep_id, to_char(date,'YYYY-MM-DD'), beat_id, seq, status
		FROM dms_journey_assignments
		WHERE ($1 = '' OR rep_id = $1) AND ($2 = '' OR date = $2::date)
		ORDER BY date, seq`, repID, date)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []models.JourneyAssignment
	for rows.Next() {
		var a models.JourneyAssignment
		if rows.Scan(&a.ID, &a.RepID, &a.Date, &a.BeatID, &a.Seq, &a.Status) == nil {
			out = append(out, a)
		}
	}
	return out
}

func (r *Repository) pgCreateJourneyAssignment(ctx context.Context, in models.JourneyAssignmentInput) (models.JourneyAssignment, error) {
	id, _ := r.pgNextID(ctx, "JNY")
	a := models.JourneyAssignment{
		ID: id, RepID: in.RepID, Date: in.Date, BeatID: in.BeatID, Seq: in.Seq, Status: "planned",
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO dms_journey_assignments (id, rep_id, date, beat_id, seq, status)
		VALUES ($1,$2,$3::date,$4,$5,$6)`,
		a.ID, a.RepID, a.Date, a.BeatID, a.Seq, a.Status)
	return a, err
}

func (r *Repository) pgPatchJourneyAssignment(ctx context.Context, id string, patch models.JourneyAssignmentPatch) (models.JourneyAssignment, error) {
	var a models.JourneyAssignment
	err := r.pool.QueryRow(ctx, `
		SELECT id, rep_id, to_char(date,'YYYY-MM-DD'), beat_id, seq, status
		FROM dms_journey_assignments WHERE id=$1`, id).
		Scan(&a.ID, &a.RepID, &a.Date, &a.BeatID, &a.Seq, &a.Status)
	if err != nil {
		return a, ErrNotFound
	}
	if patch.BeatID != "" {
		a.BeatID = patch.BeatID
	}
	if patch.Date != "" {
		a.Date = patch.Date
	}
	if patch.Status != "" {
		a.Status = patch.Status
	}
	if patch.Seq != nil {
		a.Seq = *patch.Seq
	}
	_, err = r.pool.Exec(ctx, `
		UPDATE dms_journey_assignments SET beat_id=$2, date=$3::date, seq=$4, status=$5 WHERE id=$1`,
		a.ID, a.BeatID, a.Date, a.Seq, a.Status)
	return a, err
}

func (m *memoryState) listJourneyAssignments(repID, date string) []models.JourneyAssignment {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []models.JourneyAssignment
	for _, a := range m.journeyAssignments {
		if repID != "" && a.RepID != repID {
			continue
		}
		if date != "" && a.Date != date {
			continue
		}
		out = append(out, a)
	}
	return out
}

func (m *memoryState) createJourneyAssignment(in models.JourneyAssignmentInput) (models.JourneyAssignment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a := models.JourneyAssignment{
		ID:     fmt.Sprintf("JNY-%05d", 1+len(m.journeyAssignments)),
		RepID:  in.RepID, Date: in.Date, BeatID: in.BeatID, Seq: in.Seq, Status: "planned",
	}
	m.journeyAssignments = append(m.journeyAssignments, a)
	return a, nil
}

func (m *memoryState) patchJourneyAssignment(id string, patch models.JourneyAssignmentPatch) (models.JourneyAssignment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, a := range m.journeyAssignments {
		if a.ID != id {
			continue
		}
		if patch.BeatID != "" {
			m.journeyAssignments[i].BeatID = patch.BeatID
		}
		if patch.Date != "" {
			m.journeyAssignments[i].Date = patch.Date
		}
		if patch.Status != "" {
			m.journeyAssignments[i].Status = patch.Status
		}
		if patch.Seq != nil {
			m.journeyAssignments[i].Seq = *patch.Seq
		}
		return m.journeyAssignments[i], nil
	}
	return models.JourneyAssignment{}, ErrNotFound
}

func (m *memoryState) deleteJourneyAssignment(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return deleteByID(&m.journeyAssignments, id, func(a models.JourneyAssignment) string { return a.ID })
}

// ============================================================================
// Pricing lifecycle
// ============================================================================

func (r *Repository) CreatePricing(in models.PricingInput, actor string) (models.PricingTemplate, error) {
	if r.pool != nil {
		return r.pgCreatePricing(r.bg(), in, actor)
	}
	return r.mem.createPricing(in, actor)
}

func (r *Repository) PatchPricing(id string, patch models.PricingPatch, actor string) (models.PricingTemplate, error) {
	if r.pool != nil {
		return r.pgPatchPricing(r.bg(), id, patch, actor)
	}
	return r.mem.patchPricing(id, patch, actor)
}

func (r *Repository) ApprovePricing(id, actor string) (models.PricingTemplate, error) {
	if r.pool != nil {
		return r.pgApprovePricing(r.bg(), id, actor)
	}
	return r.mem.approvePricing(id, actor)
}

func (r *Repository) ListPricingVersions(templateID string) []models.PricingVersion {
	if r.pool != nil {
		return r.pgListPricingVersions(r.bg(), templateID)
	}
	return r.mem.listPricingVersions(templateID)
}

func nextVersion(prev string) string {
	// v3 -> v4; anything unparseable -> v1
	var n int
	if _, err := fmt.Sscanf(prev, "v%d", &n); err != nil || n <= 0 {
		return "v1"
	}
	return fmt.Sprintf("v%d", n+1)
}

func (r *Repository) pgCreatePricing(ctx context.Context, in models.PricingInput, actor string) (models.PricingTemplate, error) {
	id, _ := r.pgNextID(ctx, "PRC")
	currency := in.Currency
	if currency == "" {
		currency = "UGX"
	}
	p := models.PricingTemplate{ID: id, Name: in.Name, Channel: in.Channel, Version: "v1", Currency: currency, Status: "draft"}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO dms_pricing_templates (id, name, channel, version, currency, status, created_by, updated_at)
		VALUES ($1,$2,$3,$4,$5,'draft',$6,NOW())`,
		p.ID, p.Name, p.Channel, p.Version, p.Currency, actor)
	if err == nil {
		r.pgInsertPricingVersion(ctx, p.ID, p.Version, actor)
	}
	return p, err
}

func (r *Repository) pgPatchPricing(ctx context.Context, id string, patch models.PricingPatch, actor string) (models.PricingTemplate, error) {
	var p models.PricingTemplate
	err := r.pool.QueryRow(ctx, `SELECT id, name, channel, version, currency, COALESCE(status,'draft') FROM dms_pricing_templates WHERE id=$1`, id).
		Scan(&p.ID, &p.Name, &p.Channel, &p.Version, &p.Currency, &p.Status)
	if err != nil {
		return p, ErrNotFound
	}
	if patch.Name != "" {
		p.Name = patch.Name
	}
	if patch.Channel != "" {
		p.Channel = patch.Channel
	}
	if patch.Currency != "" {
		p.Currency = patch.Currency
	}
	if patch.Status != "" {
		p.Status = patch.Status
	}
	// An edit bumps the version and re-enters draft for re-approval.
	p.Version = nextVersion(p.Version)
	if patch.Status == "" {
		p.Status = "draft"
	}
	_, err = r.pool.Exec(ctx, `UPDATE dms_pricing_templates SET name=$2, channel=$3, currency=$4, version=$5, status=$6, updated_at=NOW() WHERE id=$1`,
		p.ID, p.Name, p.Channel, p.Currency, p.Version, p.Status)
	if err == nil {
		r.pgInsertPricingVersion(ctx, p.ID, p.Version, actor)
	}
	return p, err
}

func (r *Repository) pgApprovePricing(ctx context.Context, id, actor string) (models.PricingTemplate, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE dms_pricing_templates SET status='approved', approved_by=$2, updated_at=NOW() WHERE id=$1`, id, actor)
	if err != nil {
		return models.PricingTemplate{}, err
	}
	if tag.RowsAffected() == 0 {
		return models.PricingTemplate{}, ErrNotFound
	}
	var p models.PricingTemplate
	_ = r.pool.QueryRow(ctx, `SELECT id, name, channel, version, currency, status FROM dms_pricing_templates WHERE id=$1`, id).
		Scan(&p.ID, &p.Name, &p.Channel, &p.Version, &p.Currency, &p.Status)
	return p, nil
}

func (r *Repository) pgInsertPricingVersion(ctx context.Context, templateID, version, actor string) {
	vid, _ := r.pgNextID(ctx, "PRV")
	_, _ = r.pool.Exec(ctx, `INSERT INTO dms_pricing_versions (id, template_id, version, created_by) VALUES ($1,$2,$3,$4)`,
		vid, templateID, version, actor)
}

func (r *Repository) pgListPricingVersions(ctx context.Context, templateID string) []models.PricingVersion {
	rows, err := r.pool.Query(ctx, `SELECT id, template_id, version, created_by, created_at FROM dms_pricing_versions WHERE template_id=$1 ORDER BY created_at DESC`, templateID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []models.PricingVersion
	for rows.Next() {
		var v models.PricingVersion
		if rows.Scan(&v.ID, &v.TemplateID, &v.Version, &v.CreatedBy, &v.CreatedAt) == nil {
			out = append(out, v)
		}
	}
	return out
}

func (m *memoryState) createPricing(in models.PricingInput, actor string) (models.PricingTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	currency := in.Currency
	if currency == "" {
		currency = "UGX"
	}
	p := models.PricingTemplate{
		ID: fmt.Sprintf("PRC-%03d", 1+len(m.pricing)), Name: in.Name, Channel: in.Channel,
		Version: "v1", Currency: currency, Status: "draft",
	}
	m.pricing = append(m.pricing, p)
	m.pricingVersions = append(m.pricingVersions, models.PricingVersion{
		ID: fmt.Sprintf("PRV-%05d", 1+len(m.pricingVersions)), TemplateID: p.ID, Version: p.Version, CreatedBy: actor, CreatedAt: now(),
	})
	return p, nil
}

func (m *memoryState) patchPricing(id string, patch models.PricingPatch, actor string) (models.PricingTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, p := range m.pricing {
		if p.ID != id {
			continue
		}
		if patch.Name != "" {
			m.pricing[i].Name = patch.Name
		}
		if patch.Channel != "" {
			m.pricing[i].Channel = patch.Channel
		}
		if patch.Currency != "" {
			m.pricing[i].Currency = patch.Currency
		}
		m.pricing[i].Version = nextVersion(m.pricing[i].Version)
		if patch.Status != "" {
			m.pricing[i].Status = patch.Status
		} else {
			m.pricing[i].Status = "draft"
		}
		m.pricingVersions = append(m.pricingVersions, models.PricingVersion{
			ID: fmt.Sprintf("PRV-%05d", 1+len(m.pricingVersions)), TemplateID: id, Version: m.pricing[i].Version, CreatedBy: actor, CreatedAt: now(),
		})
		return m.pricing[i], nil
	}
	return models.PricingTemplate{}, ErrNotFound
}

func (m *memoryState) approvePricing(id, actor string) (models.PricingTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, p := range m.pricing {
		if p.ID == id {
			m.pricing[i].Status = "approved"
			return m.pricing[i], nil
		}
	}
	return models.PricingTemplate{}, ErrNotFound
}

func (m *memoryState) listPricingVersions(templateID string) []models.PricingVersion {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []models.PricingVersion
	for _, v := range m.pricingVersions {
		if v.TemplateID == templateID {
			out = append(out, v)
		}
	}
	return out
}

// ============================================================================
// Report schedules
// ============================================================================

func (r *Repository) ListReportSchedules() []models.ReportSchedule {
	if r.pool != nil {
		return r.pgListReportSchedules(r.bg())
	}
	return r.mem.listReportSchedules()
}

func (r *Repository) CreateReportSchedule(in models.ReportScheduleInput) (models.ReportSchedule, error) {
	if r.pool != nil {
		return r.pgCreateReportSchedule(r.bg(), in)
	}
	return r.mem.createReportSchedule(in)
}

func (r *Repository) DeleteReportSchedule(id string) error {
	if r.pool != nil {
		return r.pgDelete(r.bg(), "dms_report_schedules", id)
	}
	return r.mem.deleteReportSchedule(id)
}

// DueReportSchedules returns active schedules whose next_run_at has passed and
// advances their next_run_at by 24h. Postgres-only; memory mode returns nil.
func (r *Repository) DueReportSchedules() []models.ReportSchedule {
	if r.pool == nil {
		return nil
	}
	ctx := r.bg()
	// Claim and return in one statement. Selecting the due rows and advancing
	// them in a second query leaves a window where another replica of this
	// service runs the same SELECT and gets the same rows, and every schedule
	// caught in that window is delivered twice — a duplicate report emailed or
	// WhatsApp'd to a real recipient.
	//
	// FOR UPDATE SKIP LOCKED locks each candidate row for this transaction and
	// steps over rows another instance already holds, so a schedule is returned
	// to exactly one caller. RETURNING hands back only what this caller claimed.
	rows, err := r.pool.Query(ctx, `
		UPDATE dms_report_schedules
		SET last_run_at = NOW(), next_run_at = NOW() + INTERVAL '1 day'
		WHERE id IN (
			SELECT id FROM dms_report_schedules
			WHERE active AND next_run_at <= NOW()
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, template_id, name, cron, channel, recipient, active`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var due []models.ReportSchedule
	for rows.Next() {
		var s models.ReportSchedule
		if rows.Scan(&s.ID, &s.TemplateID, &s.Name, &s.Cron, &s.Channel, &s.Recipient, &s.Active) == nil {
			due = append(due, s)
		}
	}
	return due
}

func scanSchedule(rows interface {
	Scan(dest ...any) error
}) (models.ReportSchedule, error) {
	var s models.ReportSchedule
	var last, next *time.Time
	err := rows.Scan(&s.ID, &s.TemplateID, &s.Name, &s.Cron, &s.Channel, &s.Recipient, &s.Active, &last, &next)
	s.LastRunAt, s.NextRunAt = last, next
	return s, err
}

func (r *Repository) pgListReportSchedules(ctx context.Context) []models.ReportSchedule {
	rows, err := r.pool.Query(ctx, `
		SELECT id, template_id, name, cron, channel, recipient, active, last_run_at, next_run_at
		FROM dms_report_schedules ORDER BY created_at DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []models.ReportSchedule
	for rows.Next() {
		if s, err := scanSchedule(rows); err == nil {
			out = append(out, s)
		}
	}
	return out
}

func (r *Repository) pgCreateReportSchedule(ctx context.Context, in models.ReportScheduleInput) (models.ReportSchedule, error) {
	id, _ := r.pgNextID(ctx, "SCH")
	channel := in.Channel
	if channel == "" {
		channel = "email"
	}
	s := models.ReportSchedule{
		ID: id, TemplateID: in.TemplateID, Name: in.Name, Cron: in.Cron,
		Channel: channel, Recipient: in.Recipient, Active: true,
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO dms_report_schedules (id, template_id, name, cron, channel, recipient, active)
		VALUES ($1,$2,$3,$4,$5,$6,true)`,
		s.ID, s.TemplateID, s.Name, s.Cron, s.Channel, s.Recipient)
	return s, err
}

func (m *memoryState) listReportSchedules() []models.ReportSchedule {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]models.ReportSchedule(nil), m.reportSchedules...)
}

func (m *memoryState) createReportSchedule(in models.ReportScheduleInput) (models.ReportSchedule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	channel := in.Channel
	if channel == "" {
		channel = "email"
	}
	s := models.ReportSchedule{
		ID: fmt.Sprintf("SCH-%05d", 1+len(m.reportSchedules)), TemplateID: in.TemplateID,
		Name: in.Name, Cron: in.Cron, Channel: channel, Recipient: in.Recipient, Active: true,
	}
	m.reportSchedules = append(m.reportSchedules, s)
	return s, nil
}

func (m *memoryState) deleteReportSchedule(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return deleteByID(&m.reportSchedules, id, func(s models.ReportSchedule) string { return s.ID })
}
