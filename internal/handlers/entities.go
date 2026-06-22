package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/iag/dms/backend/internal/events"
	"github.com/iag/dms/backend/internal/financeclient"
	"github.com/iag/dms/backend/internal/models"
	"github.com/iag/dms/backend/internal/store"
	"github.com/alvor-technologies/iag-platform-go/apierr"
)

func (h *API) ListDistributors(c *gin.Context) {
	items, total := h.Repo.ListDistributors(listOpts(c))
	paginated(c, items, total)
}

func (h *API) GetDistributor(c *gin.Context) {
	item, err := h.Repo.GetDistributor(c.Param("id"))
	if err != nil {
		notFound(c)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *API) ListOutlets(c *gin.Context) {
	items, total := h.Repo.ListOutlets(listOpts(c))
	paginated(c, items, total)
}

func (h *API) OutletStats(c *gin.Context) {
	c.JSON(http.StatusOK, h.Repo.OutletStats())
}

func (h *API) GetOutlet(c *gin.Context) {
	item, err := h.Repo.GetOutlet(c.Param("id"))
	if err != nil {
		notFound(c)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *API) CreateOutlet(c *gin.Context) {
	var in models.OutletInput
	if err := bindJSONCoerced(c, &in); err != nil || in.Name == "" || in.Channel == "" || in.DistributorID == "" {
		badRequest(c, "name, channel, and distributorId are required")
		return
	}
	out, err := h.Repo.CreateOutlet(in)
	if err != nil {
		if errors.Is(err, store.ErrInvalidInput) {
			badRequest(c, err.Error())
			return
		}
		apierr.JSONStatus(c, http.StatusInternalServerError, "create outlet failed")
		return
	}
	h.publish(c, "dms.outlet.created", gin.H{"id": out.ID, "name": out.Name})
	h.recordAudit(c, "CreateOutlet", store.AuditDetail("outlet", out.ID, "created"))
	c.JSON(http.StatusCreated, out)
}

func (h *API) ListOrders(c *gin.Context) {
	items, total := h.Repo.ListOrders(listOpts(c))
	paginated(c, items, total)
}

func (h *API) OrdersBoard(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"columns": h.Repo.OrdersBoard(), "stats": h.Repo.OrdersStats()})
}

func (h *API) GetOrder(c *gin.Context) {
	item, err := h.Repo.GetOrder(c.Param("id"))
	if err != nil {
		notFound(c)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *API) CreateOrder(c *gin.Context) {
	var in models.OrderInput
	if err := bindJSONCoerced(c, &in); err != nil || in.OutletID == "" {
		badRequest(c, "outletId is required")
		return
	}
	o := h.Repo.CreateOrder(in)
	h.publish(c, "dms.order.created", gin.H{"id": o.ID, "status": o.Status})
	// Notify the ops/sales desk when a new distribution order is placed.
	if recipient := events.DefaultNotifyRecipient(); recipient != "" && h.Events != nil {
		h.Events.PublishAlert(c.Request.Context(), "", recipient, "dms.alert", map[string]string{
			"Title": fmt.Sprintf("New distribution order: %v", o.ID),
			"Body":  fmt.Sprintf("Order %v was created (status %v).", o.ID, o.Status),
		}, fmt.Sprintf("dms-order-%v", o.ID))
	}
	h.recordAudit(c, "CreateOrder", store.AuditDetail("order", o.ID, "created"))
	c.JSON(http.StatusCreated, o)
}

func (h *API) PatchOrderStatus(c *gin.Context) {
	var body struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Status == "" {
		badRequest(c, "status is required")
		return
	}
	item, err := h.Repo.UpdateOrderStatus(c.Param("id"), body.Status)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			notFound(c)
			return
		}
		apierr.JSONStatus(c, http.StatusInternalServerError, "update failed")
		return
	}
	h.publish(c, "dms.order.status_changed", gin.H{"id": item.ID, "status": item.Status})
	h.recordAudit(c, "PatchOrderStatus", store.AuditDetail("order", item.ID, "status → "+item.Status))
	c.JSON(http.StatusOK, item)
}

func (h *API) ListBeats(c *gin.Context) {
	items, total := h.Repo.ListBeats(listOpts(c))
	paginated(c, items, total)
}

func (h *API) RoutesStats(c *gin.Context) {
	c.JSON(http.StatusOK, h.Repo.RoutesStats())
}

func (h *API) GetBeat(c *gin.Context) {
	item, err := h.Repo.GetBeat(c.Param("id"))
	if err != nil {
		notFound(c)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *API) ListReps(c *gin.Context) {
	items, total := h.Repo.ListReps(listOpts(c))
	paginated(c, items, total)
}

func (h *API) ListCheckIns(c *gin.Context) {
	items, total := h.Repo.ListCheckIns(listOpts(c))
	paginated(c, items, total)
}

func (h *API) CheckInStats(c *gin.Context) {
	c.JSON(http.StatusOK, h.Repo.CheckInStats())
}

func (h *API) CreateCheckIn(c *gin.Context) {
	var in models.CheckInInput
	if err := bindJSONCoerced(c, &in); err != nil || in.RepID == "" || in.OutletID == "" {
		badRequest(c, "repId and outletId are required")
		return
	}
	ci := h.Repo.CreateCheckIn(in)
	h.publish(c, "dms.checkin.created", gin.H{"id": ci.ID, "repId": ci.RepID})
	h.recordAudit(c, "CreateCheckIn", store.AuditDetail("check-in", ci.ID, "created"))
	c.JSON(http.StatusCreated, ci)
}

func (h *API) CreateVisitReport(c *gin.Context) {
	var in models.VisitReportInput
	if err := bindJSONCoerced(c, &in); err != nil || in.RepID == "" || in.OutletID == "" {
		badRequest(c, "repId and outletId are required")
		return
	}
	v := h.Repo.CreateVisitReport(in)
	h.publish(c, "dms.visit.reported", gin.H{"id": v.ID, "outcome": v.Outcome})
	h.recordAudit(c, "CreateVisitReport", store.AuditDetail("visit-report", v.ID, "created"))
	c.JSON(http.StatusCreated, v)
}

func (h *API) Journey(c *gin.Context) {
	repID := c.DefaultQuery("repId", "FF-04")
	c.JSON(http.StatusOK, h.Repo.Journey(repID))
}

func (h *API) ListPromotions(c *gin.Context) {
	items, total := h.Repo.ListPromotions(listOpts(c))
	paginated(c, items, total)
}

func (h *API) ListClaims(c *gin.Context) {
	items, total := h.Repo.ListClaims(listOpts(c))
	paginated(c, items, total)
}

func (h *API) ListDispatches(c *gin.Context) {
	items, total := h.Repo.ListDispatches(listOpts(c))
	paginated(c, items, total)
}

func (h *API) ListStock(c *gin.Context) {
	items, total := h.Repo.ListStock(listOpts(c))
	paginated(c, items, total)
}

func (h *API) ListSKUs(c *gin.Context) {
	items, total := h.Repo.ListSKUs(listOpts(c))
	paginated(c, items, total)
}

func (h *API) ListInvoices(c *gin.Context) {
	if h.Finance != nil && h.Finance.Enabled() {
		items, err := h.Finance.ListInvoices(c.Request.Context(), listOpts(c).Limit)
		if err == nil {
			out := make([]models.Invoice, 0, len(items))
			for _, it := range items {
				due, _ := time.Parse("2006-01-02", it.Due)
			out = append(out, models.Invoice{
					ID:            it.No,
					DistributorID: it.Customer,
					Distributor:   it.Customer,
					AmountUGX:     it.Balance,
					Status:        it.Status,
					DueDate:       due,
				})
			}
			paginated(c, out, len(out))
			return
		}
	}
	items, total := h.Repo.ListInvoices(listOpts(c))
	paginated(c, items, total)
}

func (h *API) CreateInvoice(c *gin.Context) {
	var in models.InvoiceInput
	if err := bindJSONCoerced(c, &in); err != nil || in.DistributorID == "" {
		badRequest(c, "distributorId is required")
		return
	}
	inv := h.Repo.CreateInvoice(in)

	// Finance is the system of record for invoices. When wired, push the
	// invoice upstream so the number we return — and every subsequent
	// read-through (ListInvoices/GetInvoice/FinanceSummary) — stays
	// consistent with the finance ledger. A local copy is still kept so the
	// service degrades gracefully if finance is unreachable.
	if h.Finance != nil && h.Finance.Enabled() {
		req := financeclient.CreateInvoiceRequest{
			Customer: in.DistributorID,
			Total:    in.AmountUGX,
			Status:   "open",
		}
		if !in.DueDate.IsZero() {
			req.Due = in.DueDate.Format("2006-01-02")
		}
		if created, err := h.Finance.CreateInvoice(c.Request.Context(), req); err == nil {
			inv.ID = created.No
			if created.Status != "" {
				inv.Status = created.Status
			}
		} else {
			slog.Warn("finance invoice create failed; kept local only", "err", err, "distributorId", in.DistributorID)
		}
	}

	h.publish(c, "dms.invoice.created", gin.H{"id": inv.ID})
	h.recordAudit(c, "CreateInvoice", store.AuditDetail("invoice", inv.ID, "created"))
	c.JSON(http.StatusCreated, inv)
}

func (h *API) FinanceSummary(c *gin.Context) {
	if h.Finance != nil && h.Finance.Enabled() {
		if s, err := h.Finance.Summary(c.Request.Context()); err == nil {
			// Finance owns AR/overdue/collected. It does not expose DSO, so
			// derive that from the local invoice ledger rather than reporting 0.
			c.JSON(http.StatusOK, models.FinanceSummary{
				ARBalanceUGX: parseAmountUGX(s.ARBalance),
				OverdueUGX:   parseAmountUGX(s.Overdue),
				CollectedUGX: parseAmountUGX(s.Collected),
				DSODays:      h.Repo.FinanceSummary().DSODays,
			})
			return
		}
	}
	c.JSON(http.StatusOK, h.Repo.FinanceSummary())
}

func parseAmountUGX(raw string) float64 {
	var v float64
	_, _ = fmt.Sscanf(raw, "%f", &v)
	return v
}

func (h *API) ListPricing(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"items": h.Repo.ListPricing()})
}

func (h *API) ListReports(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"items": h.Repo.ListReports()})
}

func (h *API) ListExecution(c *gin.Context) {
	items, total := h.Repo.ListExecution(listOpts(c))
	paginated(c, items, total)
}

func (h *API) KPIBoard(c *gin.Context) {
	c.JSON(http.StatusOK, h.Repo.KPIBoard())
}

func (h *API) Analytics(c *gin.Context) {
	c.JSON(http.StatusOK, h.Repo.Analytics())
}

func (h *API) Forecast(c *gin.Context) {
	c.JSON(http.StatusOK, h.Repo.Forecast(c.Query("sku")))
}

func (h *API) publish(c *gin.Context, eventType string, data any) {
	if h.Events == nil || !h.Events.Enabled() {
		return
	}
	_ = h.Events.Publish(c.Request.Context(), eventType, data)
}

func (h *API) Notifications(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"items": h.Repo.Overview().Alerts})
}
