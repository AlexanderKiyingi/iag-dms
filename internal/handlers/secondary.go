package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/iag/dms/backend/internal/auth"
	"github.com/iag/dms/backend/internal/models"
	"github.com/iag/dms/backend/internal/store"
)

// Secondary-sales desk: van loads, secondary invoices, collections, outlet
// returns, van closes, schemes, rep targets, damage claims. The rules live in
// store/secondary.go; these handlers bind, call, map errors and publish.

func docFilter(c *gin.Context) store.DocFilter {
	o := listOpts(c)
	return store.DocFilter{
		Status: o.Status, OutletID: o.OutletID, RefID: c.Query("refId"),
		Limit: o.Limit, Offset: o.Offset,
	}
}

// secondaryErr maps rule failures: 400 for a bad payload, 409 for a write the
// desk's rules refuse, 404 for a missing row.
func secondaryErr(c *gin.Context, err error, failMsg string) {
	switch {
	case errors.Is(err, store.ErrInvalidInput):
		badRequest(c, err.Error())
	case errors.Is(err, store.ErrConflict):
		conflict(c, err.Error())
	default:
		writeStoreErr(c, err, failMsg)
	}
}

func bindDoc[T any](c *gin.Context) (T, bool) {
	var in T
	if err := bindJSONCoerced(c, &in); err != nil {
		badRequest(c, "invalid body")
		return in, false
	}
	return in, true
}

// ---- van loads --------------------------------------------------------------

func (h *API) ListVanLoads(c *gin.Context) {
	items, total := h.Repo.ListVanLoads(docFilter(c))
	paginated(c, items, total)
}

func (h *API) GetVanLoad(c *gin.Context) {
	item, err := h.Repo.GetVanLoad(c.Param("id"))
	if err != nil {
		notFound(c)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *API) CreateVanLoad(c *gin.Context) {
	in, ok := bindDoc[models.VanLoad](c)
	if !ok {
		return
	}
	v, err := h.Repo.CreateVanLoad(in)
	if err != nil {
		secondaryErr(c, err, "create failed")
		return
	}
	h.publish(c, "dms.van_load.created", gin.H{"id": v.ID, "vehicleId": v.VehicleID, "quantity": v.Quantity})
	h.recordAudit(c, "CreateVanLoad", store.AuditDetail("van-load", v.ID, "created"))
	c.JSON(http.StatusCreated, v)
}

func (h *API) UpdateVanLoad(c *gin.Context) {
	in, ok := bindDoc[models.VanLoad](c)
	if !ok {
		return
	}
	v, err := h.Repo.UpdateVanLoad(c.Param("id"), in)
	if err != nil {
		secondaryErr(c, err, "update failed")
		return
	}
	h.recordAudit(c, "UpdateVanLoad", store.AuditDetail("van-load", v.ID, "updated"))
	c.JSON(http.StatusOK, v)
}

func (h *API) DeleteVanLoad(c *gin.Context) {
	h.deleteSecondary(c, "van-load", h.Repo.DeleteVanLoad)
}

// ---- secondary invoices -----------------------------------------------------

func (h *API) ListSecondaryInvoices(c *gin.Context) {
	items, total := h.Repo.ListSecondaryInvoices(docFilter(c))
	paginated(c, items, total)
}

func (h *API) GetSecondaryInvoice(c *gin.Context) {
	item, err := h.Repo.GetSecondaryInvoice(c.Param("id"))
	if err != nil {
		notFound(c)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *API) invoiceInput(c *gin.Context) (store.SecondaryInvoiceInput, bool) {
	in, ok := bindDoc[models.SecondaryInvoice](c)
	if !ok {
		return store.SecondaryInvoiceInput{}, false
	}
	return store.SecondaryInvoiceInput{
		SecondaryInvoice: in,
		Actor:            auth.ActorName(c),
		CanOverride:      auth.HasPerm(c, "dms.override_credit"),
	}, true
}

func (h *API) CreateSecondaryInvoice(c *gin.Context) {
	in, ok := h.invoiceInput(c)
	if !ok {
		return
	}
	inv, err := h.Repo.CreateSecondaryInvoice(in)
	if err != nil {
		secondaryErr(c, err, "create failed")
		return
	}
	// The Finance writer books AR/revenue from this event — this service
	// never posts a journal itself.
	h.publish(c, "distribution.invoice.posted", gin.H{
		"id": inv.ID, "outletId": inv.OutletID, "amountUgx": inv.AmountUGX, "taxUgx": inv.TaxUGX,
		"status": inv.Status, "saleKind": inv.SaleKind,
	})
	if inv.CreditOverride {
		h.recordAudit(c, "CreditOverride", store.AuditDetail("secondary-invoice", inv.ID, "override by "+inv.OverrideBy+": "+inv.OverrideReason))
	}
	h.recordAudit(c, "CreateSecondaryInvoice", store.AuditDetail("secondary-invoice", inv.ID, "created"))
	c.JSON(http.StatusCreated, inv)
}

func (h *API) UpdateSecondaryInvoice(c *gin.Context) {
	in, ok := h.invoiceInput(c)
	if !ok {
		return
	}
	inv, err := h.Repo.UpdateSecondaryInvoice(c.Param("id"), in)
	if err != nil {
		secondaryErr(c, err, "update failed")
		return
	}
	h.publish(c, "distribution.invoice.updated", gin.H{"id": inv.ID, "outletId": inv.OutletID, "status": inv.Status, "outstandingUgx": inv.OutstandingUGX})
	h.recordAudit(c, "UpdateSecondaryInvoice", store.AuditDetail("secondary-invoice", inv.ID, "updated → "+inv.Status))
	c.JSON(http.StatusOK, inv)
}

func (h *API) DeleteSecondaryInvoice(c *gin.Context) {
	h.deleteSecondary(c, "secondary-invoice", h.Repo.DeleteSecondaryInvoice)
}

// ---- collections ------------------------------------------------------------

func (h *API) ListCollections(c *gin.Context) {
	items, total := h.Repo.ListCollections(docFilter(c))
	paginated(c, items, total)
}

func (h *API) CreateCollection(c *gin.Context) {
	in, ok := bindDoc[models.Collection](c)
	if !ok {
		return
	}
	col, err := h.Repo.CreateCollection(in)
	if err != nil {
		secondaryErr(c, err, "create failed")
		return
	}
	h.publish(c, "distribution.payment.collected", gin.H{
		"id": col.ID, "invoiceId": col.InvoiceID, "outletId": col.OutletID, "amountUgx": col.AmountUGX, "method": col.Method,
	})
	h.recordAudit(c, "CreateCollection", store.AuditDetail("collection", col.ID, "collected against "+col.InvoiceID))
	c.JSON(http.StatusCreated, col)
}

func (h *API) UpdateCollection(c *gin.Context) {
	in, ok := bindDoc[models.Collection](c)
	if !ok {
		return
	}
	col, err := h.Repo.UpdateCollection(c.Param("id"), in)
	if err != nil {
		secondaryErr(c, err, "update failed")
		return
	}
	h.recordAudit(c, "UpdateCollection", store.AuditDetail("collection", col.ID, "updated → "+col.Status))
	c.JSON(http.StatusOK, col)
}

func (h *API) DeleteCollection(c *gin.Context) {
	h.deleteSecondary(c, "collection", h.Repo.DeleteCollection)
}

// ---- outlet returns ---------------------------------------------------------

func (h *API) ListOutletReturns(c *gin.Context) {
	items, total := h.Repo.ListOutletReturns(docFilter(c))
	paginated(c, items, total)
}

func (h *API) CreateOutletReturn(c *gin.Context) {
	in, ok := bindDoc[models.OutletReturn](c)
	if !ok {
		return
	}
	ret, err := h.Repo.CreateOutletReturn(in)
	if err != nil {
		secondaryErr(c, err, "create failed")
		return
	}
	// Warehouse books the receipt or write-off from this event.
	h.publish(c, "distribution.return.received", gin.H{
		"id": ret.ID, "outletId": ret.OutletID, "itemId": ret.ItemID, "quantity": ret.Quantity,
		"reason": ret.Reason, "disposition": ret.InventoryDisposition,
	})
	h.recordAudit(c, "CreateOutletReturn", store.AuditDetail("outlet-return", ret.ID, ret.Reason+" → "+ret.InventoryDisposition))
	c.JSON(http.StatusCreated, ret)
}

func (h *API) UpdateOutletReturn(c *gin.Context) {
	in, ok := bindDoc[models.OutletReturn](c)
	if !ok {
		return
	}
	ret, err := h.Repo.UpdateOutletReturn(c.Param("id"), in)
	if err != nil {
		secondaryErr(c, err, "update failed")
		return
	}
	h.recordAudit(c, "UpdateOutletReturn", store.AuditDetail("outlet-return", ret.ID, "updated → "+ret.Status))
	c.JSON(http.StatusOK, ret)
}

func (h *API) DeleteOutletReturn(c *gin.Context) {
	h.deleteSecondary(c, "outlet-return", h.Repo.DeleteOutletReturn)
}

// ---- van reconciliations ----------------------------------------------------

func (h *API) ListVanRecons(c *gin.Context) {
	items, total := h.Repo.ListVanRecons(docFilter(c))
	paginated(c, items, total)
}

func (h *API) CreateVanRecon(c *gin.Context) {
	in, ok := bindDoc[models.VanRecon](c)
	if !ok {
		return
	}
	rec, err := h.Repo.CreateVanRecon(in)
	if err != nil {
		secondaryErr(c, err, "create failed")
		return
	}
	h.publish(c, "distribution.van.closed", gin.H{"id": rec.ID, "vanLoadId": rec.VanLoadID, "status": rec.Status, "stockVariance": rec.StockVariance, "cashVariance": rec.CashVariance})
	h.recordAudit(c, "CreateVanRecon", store.AuditDetail("van-recon", rec.ID, rec.Status))
	c.JSON(http.StatusCreated, rec)
}

func (h *API) UpdateVanRecon(c *gin.Context) {
	in, ok := bindDoc[models.VanRecon](c)
	if !ok {
		return
	}
	rec, err := h.Repo.UpdateVanRecon(c.Param("id"), in)
	if err != nil {
		secondaryErr(c, err, "update failed")
		return
	}
	h.recordAudit(c, "UpdateVanRecon", store.AuditDetail("van-recon", rec.ID, "updated → "+rec.Status))
	c.JSON(http.StatusOK, rec)
}

func (h *API) DeleteVanRecon(c *gin.Context) {
	h.deleteSecondary(c, "van-recon", h.Repo.DeleteVanRecon)
}

// ---- schemes ----------------------------------------------------------------

func (h *API) ListSchemes(c *gin.Context) {
	items, total := h.Repo.ListSchemes(docFilter(c))
	paginated(c, items, total)
}

func (h *API) CreateScheme(c *gin.Context) {
	in, ok := bindDoc[models.Scheme](c)
	if !ok {
		return
	}
	sch, err := h.Repo.CreateScheme(in)
	if err != nil {
		secondaryErr(c, err, "create failed")
		return
	}
	h.recordAudit(c, "CreateScheme", store.AuditDetail("scheme", sch.ID, "created"))
	c.JSON(http.StatusCreated, sch)
}

func (h *API) UpdateScheme(c *gin.Context) {
	in, ok := bindDoc[models.Scheme](c)
	if !ok {
		return
	}
	sch, err := h.Repo.UpdateScheme(c.Param("id"), in)
	if err != nil {
		secondaryErr(c, err, "update failed")
		return
	}
	h.recordAudit(c, "UpdateScheme", store.AuditDetail("scheme", sch.ID, "updated"))
	c.JSON(http.StatusOK, sch)
}

func (h *API) DeleteScheme(c *gin.Context) {
	h.deleteSecondary(c, "scheme", h.Repo.DeleteScheme)
}

// ---- rep targets ------------------------------------------------------------

func (h *API) ListRepTargets(c *gin.Context) {
	items, total := h.Repo.ListRepTargets(docFilter(c))
	paginated(c, items, total)
}

func (h *API) CreateRepTarget(c *gin.Context) {
	in, ok := bindDoc[models.RepTarget](c)
	if !ok {
		return
	}
	t, err := h.Repo.CreateRepTarget(in)
	if err != nil {
		secondaryErr(c, err, "create failed")
		return
	}
	h.recordAudit(c, "CreateRepTarget", store.AuditDetail("rep-target", t.ID, "created"))
	c.JSON(http.StatusCreated, t)
}

func (h *API) UpdateRepTarget(c *gin.Context) {
	in, ok := bindDoc[models.RepTarget](c)
	if !ok {
		return
	}
	t, err := h.Repo.UpdateRepTarget(c.Param("id"), in)
	if err != nil {
		secondaryErr(c, err, "update failed")
		return
	}
	h.recordAudit(c, "UpdateRepTarget", store.AuditDetail("rep-target", t.ID, "updated"))
	c.JSON(http.StatusOK, t)
}

func (h *API) DeleteRepTarget(c *gin.Context) {
	h.deleteSecondary(c, "rep-target", h.Repo.DeleteRepTarget)
}

// ---- damage claims ----------------------------------------------------------

func (h *API) ListDamageClaims(c *gin.Context) {
	items, total := h.Repo.ListDamageClaims(docFilter(c))
	paginated(c, items, total)
}

func (h *API) CreateDamageClaim(c *gin.Context) {
	in, ok := bindDoc[models.DamageClaim](c)
	if !ok {
		return
	}
	cl, err := h.Repo.CreateDamageClaim(in)
	if err != nil {
		secondaryErr(c, err, "create failed")
		return
	}
	// Fleet picks this up as a damage claim against the vehicle/driver.
	h.publish(c, "distribution.damage_claim.raised", gin.H{"id": cl.ID, "vanId": cl.VanID, "driver": cl.Driver, "itemId": cl.ItemID, "quantity": cl.Quantity})
	h.recordAudit(c, "CreateDamageClaim", store.AuditDetail("damage-claim", cl.ID, "raised"))
	c.JSON(http.StatusCreated, cl)
}

func (h *API) UpdateDamageClaim(c *gin.Context) {
	in, ok := bindDoc[models.DamageClaim](c)
	if !ok {
		return
	}
	cl, err := h.Repo.UpdateDamageClaim(c.Param("id"), in)
	if err != nil {
		secondaryErr(c, err, "update failed")
		return
	}
	h.recordAudit(c, "UpdateDamageClaim", store.AuditDetail("damage-claim", cl.ID, "updated → "+cl.Status))
	c.JSON(http.StatusOK, cl)
}

func (h *API) DeleteDamageClaim(c *gin.Context) {
	h.deleteSecondary(c, "damage-claim", h.Repo.DeleteDamageClaim)
}

// deleteSecondary is deleteEntity with the rules' 409 mapped.
func (h *API) deleteSecondary(c *gin.Context, entity string, del func(string) error) {
	id := c.Param("id")
	if err := del(id); err != nil {
		secondaryErr(c, err, "delete failed")
		return
	}
	h.publish(c, "distribution."+entity+".deleted", gin.H{"id": id})
	h.recordAudit(c, "Delete "+entity, store.AuditDetail(entity, id, "deleted"))
	c.Status(http.StatusNoContent)
}
