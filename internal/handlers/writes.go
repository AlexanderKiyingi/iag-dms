package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/iag/dms/backend/internal/models"
	"github.com/iag/dms/backend/internal/store"
	"github.com/alvor-technologies/iag-platform-go/apierr"
)

func (h *API) PatchOutlet(c *gin.Context) {
	var patch models.OutletPatch
	if err := c.ShouldBindJSON(&patch); err != nil {
		badRequest(c, "invalid body")
		return
	}
	if patch.Name == "" && patch.Address == "" && patch.Channel == "" && patch.BeatID == "" &&
		patch.Status == "" && patch.Score == "" && patch.Frequency == "" {
		badRequest(c, "at least one field required")
		return
	}
	item, err := h.Repo.PatchOutlet(c.Param("id"), patch)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			notFound(c)
			return
		}
		apierr.JSONStatus(c, http.StatusInternalServerError, "update failed")
		return
	}
	h.publish(c, "dms.outlet.updated", gin.H{"id": item.ID, "status": item.Status})
	h.recordAudit(c, "PatchOutlet", store.AuditDetail("outlet", item.ID, "updated"))
	c.JSON(http.StatusOK, item)
}

func (h *API) GetInvoice(c *gin.Context) {
	id := c.Param("id")
	if h.Finance != nil && h.Finance.Enabled() {
		if inv, err := h.Finance.GetInvoice(c.Request.Context(), id); err == nil {
			h.recordAudit(c, "GetInvoice", store.AuditDetail("invoice", inv.No, "finance upstream"))
			c.JSON(http.StatusOK, models.Invoice{
				ID: inv.No, Distributor: inv.Customer, DistributorID: inv.Customer,
				AmountUGX: inv.Total, Status: inv.Status,
			})
			return
		}
	}
	item, err := h.Repo.GetInvoice(id)
	if err != nil {
		notFound(c)
		return
	}
	h.recordAudit(c, "GetInvoice", store.AuditDetail("invoice", item.ID, "read"))
	c.JSON(http.StatusOK, item)
}

func (h *API) ListVisitReports(c *gin.Context) {
	items, total := h.Repo.ListVisitReports(listOpts(c))
	paginated(c, items, total)
}

func (h *API) CompleteCheckIn(c *gin.Context) {
	item, err := h.Repo.CompleteCheckIn(c.Param("id"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			notFound(c)
			return
		}
		apierr.JSONStatus(c, http.StatusInternalServerError, "complete failed")
		return
	}
	h.publish(c, "dms.checkin.completed", gin.H{"id": item.ID, "repId": item.RepID})
	h.recordAudit(c, "CompleteCheckIn", store.AuditDetail("check-in", item.ID, "completed"))
	c.JSON(http.StatusOK, item)
}

func (h *API) CreateClaim(c *gin.Context) {
	var in models.ClaimInput
	if err := bindJSONCoerced(c, &in); err != nil || in.OutletID == "" || in.Type == "" {
		badRequest(c, "outletId and type are required")
		return
	}
	cl := h.Repo.CreateClaim(in)
	h.publish(c, "dms.claim.created", gin.H{"id": cl.ID, "type": cl.Type})
	h.recordAudit(c, "CreateClaim", store.AuditDetail("claim", cl.ID, "created"))
	c.JSON(http.StatusCreated, cl)
}

func (h *API) CreatePromotion(c *gin.Context) {
	var in models.PromotionInput
	if err := bindJSONCoerced(c, &in); err != nil || in.Name == "" {
		badRequest(c, "name is required")
		return
	}
	p := h.Repo.CreatePromotion(in)
	h.publish(c, "dms.promotion.created", gin.H{"id": p.ID, "name": p.Name})
	h.recordAudit(c, "CreatePromotion", store.AuditDetail("promotion", p.ID, "created"))
	c.JSON(http.StatusCreated, p)
}

func (h *API) CreateDispatch(c *gin.Context) {
	var in models.DispatchInput
	if err := c.ShouldBindJSON(&in); err != nil || in.TruckID == "" || len(in.OrderIDs) == 0 {
		badRequest(c, "truckId and orderIds are required")
		return
	}
	d := h.Repo.CreateDispatch(in)
	h.publish(c, "dms.dispatch.created", gin.H{"id": d.ID, "truckId": d.TruckID})
	h.recordAudit(c, "CreateDispatch", store.AuditDetail("dispatch", d.ID, "created"))
	c.JSON(http.StatusCreated, d)
}

// ---- Generic patches -------------------------------------------------------

func (h *API) PatchPromotion(c *gin.Context) {
	var patch models.PromotionPatch
	if err := bindJSONCoerced(c, &patch); err != nil {
		badRequest(c, "invalid body")
		return
	}
	item, err := h.Repo.PatchPromotion(c.Param("id"), patch)
	if err != nil {
		writeStoreErr(c, err, "update failed")
		return
	}
	h.publish(c, "dms.promotion.updated", gin.H{"id": item.ID, "status": item.Status})
	h.recordAudit(c, "PatchPromotion", store.AuditDetail("promotion", item.ID, "updated"))
	c.JSON(http.StatusOK, item)
}

func (h *API) PatchClaim(c *gin.Context) {
	var patch models.ClaimPatch
	if err := bindJSONCoerced(c, &patch); err != nil {
		badRequest(c, "invalid body")
		return
	}
	item, err := h.Repo.PatchClaim(c.Param("id"), patch)
	if err != nil {
		writeStoreErr(c, err, "update failed")
		return
	}
	h.publish(c, "dms.claim.updated", gin.H{"id": item.ID, "status": item.Status})
	h.recordAudit(c, "PatchClaim", store.AuditDetail("claim", item.ID, "updated"))
	c.JSON(http.StatusOK, item)
}

func (h *API) PatchDispatch(c *gin.Context) {
	var patch models.DispatchPatch
	if err := bindJSONCoerced(c, &patch); err != nil {
		badRequest(c, "invalid body")
		return
	}
	item, err := h.Repo.PatchDispatch(c.Param("id"), patch)
	if err != nil {
		writeStoreErr(c, err, "update failed")
		return
	}
	h.publish(c, "dms.dispatch.updated", gin.H{"id": item.ID, "status": item.Status})
	h.recordAudit(c, "PatchDispatch", store.AuditDetail("dispatch", item.ID, "updated"))
	c.JSON(http.StatusOK, item)
}

func (h *API) PatchInvoice(c *gin.Context) {
	var patch models.InvoicePatch
	if err := bindJSONCoerced(c, &patch); err != nil {
		badRequest(c, "invalid body")
		return
	}
	item, err := h.Repo.PatchInvoice(c.Param("id"), patch)
	if err != nil {
		writeStoreErr(c, err, "update failed")
		return
	}
	h.publish(c, "dms.invoice.updated", gin.H{"id": item.ID, "status": item.Status})
	h.recordAudit(c, "PatchInvoice", store.AuditDetail("invoice", item.ID, "updated"))
	c.JSON(http.StatusOK, item)
}

// ---- Deletes ---------------------------------------------------------------

func (h *API) DeleteOutlet(c *gin.Context) {
	h.deleteEntity(c, "outlet", h.Repo.DeleteOutlet)
}

func (h *API) DeleteOrder(c *gin.Context) {
	id := c.Param("id")
	if ord, err := h.Repo.GetOrder(id); err == nil && strings.EqualFold(ord.Status, "delivered") {
		conflict(c, "delivered orders cannot be deleted")
		return
	}
	h.deleteEntity(c, "order", h.Repo.DeleteOrder)
}

func (h *API) DeleteCheckIn(c *gin.Context) {
	h.deleteEntity(c, "check-in", h.Repo.DeleteCheckIn)
}

func (h *API) DeletePromotion(c *gin.Context) {
	h.deleteEntity(c, "promotion", h.Repo.DeletePromotion)
}

func (h *API) DeleteClaim(c *gin.Context) {
	h.deleteEntity(c, "claim", h.Repo.DeleteClaim)
}

func (h *API) DeleteDispatch(c *gin.Context) {
	h.deleteEntity(c, "dispatch", h.Repo.DeleteDispatch)
}

func (h *API) DeleteInvoice(c *gin.Context) {
	id := c.Param("id")
	if inv, err := h.Repo.GetInvoice(id); err == nil && strings.EqualFold(inv.Status, "paid") {
		conflict(c, "paid invoices cannot be deleted")
		return
	}
	h.deleteEntity(c, "invoice", h.Repo.DeleteInvoice)
}

// deleteEntity runs a repository delete, emits a dms.<entity>.deleted event and
// audit row, and returns 204 (or 404 when the id is unknown).
func (h *API) deleteEntity(c *gin.Context, entity string, del func(string) error) {
	id := c.Param("id")
	if err := del(id); err != nil {
		writeStoreErr(c, err, "delete failed")
		return
	}
	h.publish(c, "dms."+entity+".deleted", gin.H{"id": id})
	h.recordAudit(c, "Delete "+entity, store.AuditDetail(entity, id, "deleted"))
	c.Status(http.StatusNoContent)
}

// writeStoreErr maps a store error to the right HTTP status.
func writeStoreErr(c *gin.Context, err error, failMsg string) {
	if errors.Is(err, store.ErrNotFound) {
		notFound(c)
		return
	}
	apierr.JSONStatus(c, http.StatusInternalServerError, failMsg)
}

func conflict(c *gin.Context, msg string) {
	apierr.JSONStatus(c, http.StatusConflict, msg)
}

func (h *API) RunReport(c *gin.Context) {
	var in models.ReportRunInput
	if err := c.ShouldBindJSON(&in); err != nil {
		badRequest(c, "invalid body")
		return
	}
	if strings.TrimSpace(in.Name) == "" && strings.TrimSpace(in.TemplateID) == "" {
		badRequest(c, "name or templateId is required")
		return
	}
	run := h.Repo.RunReport(in)
	// Deliver by email when a recipient was supplied (report studio "schedule
	// delivery"): the notifications service renders and sends it.
	if strings.TrimSpace(in.EmailTo) != "" && h.Events != nil && h.Events.Enabled() {
		h.Events.PublishAlert(c.Request.Context(), "email", in.EmailTo, "dms.alert", map[string]string{
			"Title": "Report ready: " + run.Name,
			"Body":  fmt.Sprintf("Your report \"%s\" generated %d rows and is ready.", run.Name, run.RowCount),
		}, run.JobID)
		run.Message = "Report generated and emailed to " + in.EmailTo
	}
	status := http.StatusOK
	if run.Status == "queued" {
		status = http.StatusAccepted
	}
	c.JSON(status, run)
}

func (h *API) ExportPage(c *gin.Context) {
	page := c.Param("page")
	if page == "" {
		badRequest(c, "page is required")
		return
	}
	var body models.ExportInput
	_ = c.ShouldBindJSON(&body)
	payload := h.Repo.ExportPage(page, body.Format)
	c.JSON(http.StatusOK, payload)
}
