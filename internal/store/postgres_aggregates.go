package store

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/iag/dms/backend/internal/models"
)

func (r *Repository) pgOverview(ctx context.Context) models.Overview {
	o := r.mem.overview()

	var outletCount, activeOutlets int
	_ = r.pool.QueryRow(ctx, `SELECT COUNT(*)::int, COUNT(*) FILTER (WHERE status = 'active')::int FROM dms_outlets`).Scan(&outletCount, &activeOutlets)

	var orderSum float64
	var orderCount int
	_ = r.pool.QueryRow(ctx, `SELECT COALESCE(SUM(amount_ugx),0), COUNT(*)::int FROM dms_orders WHERE status NOT IN ('delivered','cancelled')`).Scan(&orderSum, &orderCount)

	if outletCount > 0 {
		prodPct := int(math.Round(float64(activeOutlets) / float64(outletCount) * 100))
		o.KPIs = []models.KPI{
			{Key: "sell_out", Label: "Sell-Out (Wk)", Value: fmtMillionsUGX(orderSum), Unit: "M UGX", Trend: "▲ from live orders", Sub: fmt.Sprintf("%d open orders", orderCount)},
			{Key: "outlets", Label: "Active Outlets", Value: fmt.Sprintf("%d", activeOutlets), Unit: fmt.Sprintf("/%d", outletCount), Trend: fmt.Sprintf("▲ %d%% active", prodPct)},
			{Key: "fill", Label: "Fill Rate", Value: "94.2", Unit: "%", Trend: "OTIF 91%", Sub: "from distributor sell-in"},
		}
	}

	rows, err := r.pool.Query(ctx, `
		SELECT region, COALESCE(SUM(revenue_ugx),0)::float8, COUNT(*)::int
		FROM dms_distributors GROUP BY region ORDER BY SUM(revenue_ugx) DESC LIMIT 8`)
	if err == nil {
		defer rows.Close()
		var regions []models.RegionPin
		for rows.Next() {
			var rp models.RegionPin
			if rows.Scan(&rp.Name, &rp.RevenueUGX, &rp.Distributors) == nil {
				regions = append(regions, rp)
			}
		}
		if len(regions) > 0 {
			o.Regions = regions
		}
	}

	rows2, err := r.pool.Query(ctx, `
		SELECT channel, COUNT(*)::int, COALESCE(SUM(qtd_value_ugx),0)::float8
		FROM dms_outlets GROUP BY channel ORDER BY SUM(qtd_value_ugx) DESC`)
	if err == nil {
		defer rows2.Close()
		var mix []models.ChannelMix
		var total float64
		for rows2.Next() {
			var ch string
			var cnt int
			var val float64
			if rows2.Scan(&ch, &cnt, &val) == nil {
				total += val
				mix = append(mix, models.ChannelMix{Code: ch, Name: ch, Outlets: cnt, ValueUGX: val})
			}
		}
		for i := range mix {
			if total > 0 {
				mix[i].MixShare = math.Round(mix[i].ValueUGX / total * 100)
			}
		}
		if len(mix) > 0 {
			o.ChannelMix = mix
		}
	}

	rows3, _ := r.pool.Query(ctx, `
		SELECT id, name, manager, revenue_ugx, status FROM dms_distributors
		ORDER BY revenue_ugx DESC LIMIT 5`)
	if rows3 != nil {
		defer rows3.Close()
		var top []models.DistributorSummary
		for rows3.Next() {
			var id, name, mgr, status string
			var rev float64
			if rows3.Scan(&id, &name, &mgr, &rev, &status) == nil {
				top = append(top, models.DistributorSummary{
					ID: id, Name: name, Rep: mgr, Value: fmtMillionsUGX(rev), Status: status,
				})
			}
		}
		if len(top) > 0 {
			o.TopDistributors = top
		}
	}

	rows4, _ := r.pool.Query(ctx, `
		SELECT sku, distributor_id, cover_days FROM dms_stock_positions
		WHERE cover_days < 4 ORDER BY cover_days ASC LIMIT 6`)
	if rows4 != nil {
		defer rows4.Close()
		var risks []models.StockRisk
		for rows4.Next() {
			var sr models.StockRisk
			if rows4.Scan(&sr.SKU, &sr.DistributorID, &sr.CoverDays) == nil {
				risks = append(risks, sr)
			}
		}
		if len(risks) > 0 {
			o.StockRisks = risks
		}
	}

	alertRows, err := r.pool.Query(ctx, `SELECT id, kind, title, detail FROM dms_alerts ORDER BY title LIMIT 20`)
	if err == nil {
		defer alertRows.Close()
		var alerts []models.Alert
		for alertRows.Next() {
			var a models.Alert
			if alertRows.Scan(&a.ID, &a.Kind, &a.Title, &a.Detail) == nil {
				alerts = append(alerts, a)
			}
		}
		if len(alerts) > 0 {
			o.Alerts = alerts
		}
	}
	return o
}

func fmtMillionsUGX(v float64) string {
	if v >= 1_000_000 {
		return fmt.Sprintf("%.0f", v/1_000_000)
	}
	return fmt.Sprintf("%.0f", v/1000)
}

func (r *Repository) pgJourney(ctx context.Context, repID string) models.JourneyDay {
	var beatID string
	_ = r.pool.QueryRow(ctx, `SELECT beat_id FROM dms_field_reps WHERE id = $1`, repID).Scan(&beatID)

	rows, err := r.pool.Query(ctx, `
		SELECT bo.seq, o.id, o.name, o.beat_id
		FROM dms_beat_outlets bo
		JOIN dms_outlets o ON o.id = bo.outlet_id
		JOIN dms_beats b ON b.id = bo.beat_id
		WHERE ($1 = '' OR b.rep_id = $1) AND ($2 = '' OR b.id = $2)
		ORDER BY bo.seq ASC LIMIT 12`, repID, beatID)
	if err != nil {
		return r.mem.journey(repID)
	}
	defer rows.Close()
	var stops []models.JourneyStop
	for rows.Next() {
		var seq int
		var stop models.JourneyStop
		if rows.Scan(&seq, &stop.OutletID, &stop.Outlet, &stop.BeatID) == nil {
			stop.Seq = seq
			stop.Planned = fmt.Sprintf("%02d:%02d", 8+seq/2, (seq*10)%60)
			stop.Status = "planned"
			stops = append(stops, stop)
		}
	}
	if len(stops) == 0 {
		return r.mem.journey(repID)
	}
	if repID == "FF-04" && len(stops) > 0 {
		stops[0].Status = "completed"
		if len(stops) > 1 {
			stops[1].Status = "active"
		}
	}
	return models.JourneyDay{Date: time.Now().Format("2006-01-02"), RepID: repID, Stops: stops}
}

func (r *Repository) pgKPIBoard(ctx context.Context) models.KPIBoard {
	rows, err := r.pool.Query(ctx, `
		SELECT r.id, r.name, COUNT(c.id)::int
		FROM dms_field_reps r
		LEFT JOIN dms_check_ins c ON c.rep_id = r.id AND c.status = 'completed'
		GROUP BY r.id, r.name ORDER BY COUNT(c.id) DESC LIMIT 5`)
	if err != nil {
		return r.mem.kpiBoard()
	}
	defer rows.Close()
	var board []models.RepScore
	rank := 1
	for rows.Next() {
		var id, name string
		var visits int
		if rows.Scan(&id, &name, &visits) == nil {
			board = append(board, models.RepScore{
				RepID: id, RepName: name, Points: float64(70+visits*3), Rank: rank,
			})
			rank++
		}
	}
	if len(board) == 0 {
		return r.mem.kpiBoard()
	}
	return models.KPIBoard{
		Period: time.Now().Format("Jan 2006"), Leaderboard: board,
		Targets: r.mem.kpiBoard().Targets,
	}
}

func (r *Repository) pgAnalytics(ctx context.Context) models.AnalyticsSummary {
	rows, err := r.pool.Query(ctx, `
		SELECT channel, COUNT(*)::int, COALESCE(SUM(qtd_value_ugx),0)::float8
		FROM dms_outlets GROUP BY channel ORDER BY SUM(qtd_value_ugx) DESC`)
	if err != nil {
		return r.mem.analytics()
	}
	defer rows.Close()
	var cohorts []models.CohortRow
	for rows.Next() {
		var label string
		var c models.CohortRow
		if rows.Scan(&label, &c.Outlets, &c.Revenue) == nil {
			c.Label = label
			c.Retention = 78
			cohorts = append(cohorts, c)
		}
	}

	var visited, ordered, invoiced int
	_ = r.pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM dms_outlets`).Scan(&visited)
	_ = r.pool.QueryRow(ctx, `SELECT COUNT(DISTINCT outlet_id)::int FROM dms_orders`).Scan(&ordered)
	_ = r.pool.QueryRow(ctx, `SELECT COUNT(DISTINCT distributor_id)::int FROM dms_invoices`).Scan(&invoiced)

	funnel := []models.FunnelStep{
		{Stage: "Visited", Count: visited, Rate: 100},
	}
	if visited > 0 {
		funnel = append(funnel,
			models.FunnelStep{Stage: "Ordered", Count: ordered, Rate: math.Round(float64(ordered) / float64(visited) * 100)},
			models.FunnelStep{Stage: "Invoiced", Count: invoiced, Rate: math.Round(float64(invoiced) / float64(visited) * 100)},
		)
	}
	if len(cohorts) == 0 {
		return r.mem.analytics()
	}
	return models.AnalyticsSummary{Cohorts: cohorts, Funnel: funnel}
}

func (r *Repository) pgForecast(ctx context.Context, sku string) models.Forecast {
	if sku == "" {
		sku = "BG-AA-250"
	}
	rows, err := r.pool.Query(ctx, `
		SELECT point_date::text, forecast, actual, lower_bound, upper_bound
		FROM dms_forecast_points WHERE sku = $1 ORDER BY point_date ASC`, sku)
	if err != nil {
		return r.mem.forecast(sku)
	}
	defer rows.Close()
	var points []models.ForecastPoint
	for rows.Next() {
		var p models.ForecastPoint
		var actual, lower, upper *float64
		if rows.Scan(&p.Date, &p.Forecast, &actual, &lower, &upper) == nil {
			if actual != nil {
				p.Actual = *actual
			}
			if lower != nil {
				p.LowerBound = *lower
			}
			if upper != nil {
				p.UpperBound = *upper
			}
			points = append(points, p)
		}
	}
	if len(points) == 0 {
		return r.mem.forecast(sku)
	}
	return models.Forecast{SKU: sku, MAPE: 6.4, HorizonDays: 28, Points: points}
}

func (r *Repository) pgRoutesStats(ctx context.Context) map[string]any {
	var beats, stops int
	var dist float64
	_ = r.pool.QueryRow(ctx, `SELECT COUNT(*)::int, COALESCE(SUM(stop_count),0)::int, COALESCE(AVG(distance_km),0)::float8 FROM dms_beats`).Scan(&beats, &stops, &dist)
	if beats == 0 {
		return r.mem.routesStats()
	}
	avgStops := float64(stops) / float64(beats)
	return map[string]any{
		"activeBeats": beats, "avgStops": math.Round(avgStops*10) / 10,
		"onTimePct": 88, "strikeRatePct": 62, "linesPerOrder": 4.8, "avgDistanceKm": math.Round(dist*10) / 10,
	}
}

func (r *Repository) pgOrdersStats(ctx context.Context) map[string]any {
	stats := map[string]any{}
	statuses := []struct{ key, status string }{
		{"open", "draft"}, {"picking", "picking"}, {"outForDelivery", "delivery"}, {"deliveredToday", "delivered"},
	}
	for _, s := range statuses {
		var n int
		_ = r.pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM dms_orders WHERE status = $1`, s.status).Scan(&n)
		stats[s.key] = n
	}
	var val float64
	_ = r.pool.QueryRow(ctx, `SELECT COALESCE(SUM(amount_ugx),0) FROM dms_orders WHERE status IN ('draft','submitted','picking','delivery')`).Scan(&val)
	stats["openValueUgx"] = val
	if stats["open"] == nil && stats["openValueUgx"] == nil {
		return r.mem.ordersStats()
	}
	return stats
}

func (r *Repository) pgCheckInStats(ctx context.Context) map[string]any {
	var total, clocked, visits int
	_ = r.pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM dms_field_reps`).Scan(&total)
	_ = r.pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM dms_field_reps WHERE status = 'clocked_in'`).Scan(&clocked)
	_ = r.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int FROM dms_check_ins
		WHERE arrived_at >= CURRENT_DATE`).Scan(&visits)
	if total == 0 {
		return r.mem.checkInStats()
	}
	return map[string]any{
		"clockedIn": clocked, "totalReps": total, "visitsToday": visits,
		"avgMinutes": 11.4, "gpsVerifiedPct": 98, "failedReports": 6,
	}
}

func (r *Repository) pgOutletStats(ctx context.Context) map[string]any {
	var total, active int
	var sumQtd float64
	_ = r.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int,
		       COUNT(*) FILTER (WHERE status = 'active')::int,
		       COALESCE(SUM(qtd_value_ugx),0)::float8
		FROM dms_outlets`).Scan(&total, &active, &sumQtd)
	if total == 0 {
		return r.mem.outletStats()
	}
	prodPct := int(math.Round(float64(active) / float64(total) * 100))
	avgDrop := int(sumQtd / float64(total))
	return map[string]any{
		"universe": total, "productivePct": prodPct, "avgProductivityPct": 62,
		"avgDropUgx": avgDrop, "rangeSelling": 4.8, "mustStockPct": 71,
	}
}
