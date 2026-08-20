package handlers

import (
	"fmt"
	"html"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/iag/dms/backend/internal/events"
	"github.com/iag/dms/backend/internal/models"
	"github.com/iag/dms/backend/internal/store"
)

// SubmitInvoiceEFRIS fiscalises an invoice through iag-finance's URA EFRIS
// integration and records the returned receipt. When finance is not wired it
// degrades to a local simulated receipt so the flow stays exercisable in dev.
func (h *API) SubmitInvoiceEFRIS(c *gin.Context) {
	id := c.Param("id")
	inv, err := h.Repo.GetInvoice(id)
	if err != nil {
		notFound(c)
		return
	}
	status, receipt := "submitted", ""
	if h.Finance != nil && h.Finance.Enabled() {
		res, ferr := h.Finance.SubmitEFRIS(c.Request.Context(), inv.ID)
		if ferr != nil {
			apierrStatus(c, http.StatusBadGateway, "EFRIS submission failed: "+ferr.Error())
			return
		}
		status, receipt = res.Status, res.URAReceipt
	} else {
		receipt = "SIM-" + inv.ID // simulate mode (no finance upstream)
	}
	updated, err := h.Repo.SetInvoiceEFRIS(id, status, receipt)
	if err != nil {
		writeStoreErr(c, err, "persist failed")
		return
	}
	h.publish(c, "dms.invoice.fiscalised", gin.H{"id": id, "status": status, "receipt": receipt})
	h.recordAudit(c, "SubmitInvoiceEFRIS", store.AuditDetail("invoice", id, "fiscalised "+status))
	c.JSON(http.StatusOK, updated)
}

// GetInvoiceDocument renders a self-contained HTML invoice carrying the EFRIS
// fiscal stamp (the URA receipt is the digital signature). It records the
// document URL on the invoice for the frontend.
func (h *API) GetInvoiceDocument(c *gin.Context) {
	id := c.Param("id")
	inv, err := h.Repo.GetInvoice(id)
	if err != nil {
		notFound(c)
		return
	}
	docURL := h.Cfg.PublicAPIURL + h.Cfg.GatewayAPIPrefix + "/v1/invoices/" + id + "/document"
	if inv.DocumentURL != docURL {
		_ = h.Repo.SetInvoiceDocument(id, docURL)
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(renderInvoiceHTML(inv)))
}

// SendInvoice emails the invoice to the distributor (or a supplied recipient)
// via the notifications service using the seeded invoice-ready template.
func (h *API) SendInvoice(c *gin.Context) {
	id := c.Param("id")
	inv, err := h.Repo.GetInvoice(id)
	if err != nil {
		notFound(c)
		return
	}
	var body struct {
		Recipient string `json:"recipient"`
	}
	_ = c.ShouldBindJSON(&body)
	recipient := body.Recipient
	if recipient == "" {
		recipient = events.DefaultNotifyRecipient()
	}
	if recipient == "" {
		badRequest(c, "recipient is required (no default configured)")
		return
	}
	docURL := h.Cfg.PublicAPIURL + h.Cfg.GatewayAPIPrefix + "/v1/invoices/" + id + "/document"
	if h.Events != nil && h.Events.Enabled() {
		// "invoice-ready-email" is the seeded template name; the file key in the
		// notifications repo is "invoice-ready" but it is stored under the
		// -email suffix (see templates/email/embed.go seedNames). Passing the
		// file key resolves to nothing and the dispatch fails template-not-found.
		h.Events.PublishAlert(c.Request.Context(), "email", recipient, "invoice-ready-email", map[string]string{
			"Name":          inv.Distributor,
			"InvoiceNumber": inv.ID,
			"Amount":        fmt.Sprintf("UGX %.0f", inv.AmountUGX),
			"DueDate":       inv.DueDate.Format("2006-01-02"),
			"InvoiceURL":    docURL,
		}, inv.ID)
	}
	h.recordAudit(c, "SendInvoice", store.AuditDetail("invoice", id, "emailed to "+recipient))
	c.JSON(http.StatusAccepted, gin.H{"status": "queued", "recipient": recipient, "documentUrl": docURL})
}

func renderInvoiceHTML(inv models.Invoice) string {
	fiscal := "Not fiscalised"
	if inv.URAReceipt != "" {
		fiscal = fmt.Sprintf("URA EFRIS receipt: %s (%s)", html.EscapeString(inv.URAReceipt), html.EscapeString(inv.EFRISStatus))
	}
	return fmt.Sprintf(`<!doctype html>
<html><head><meta charset="utf-8"><title>Invoice %[1]s</title>
<style>body{font-family:system-ui,sans-serif;max-width:720px;margin:40px auto;color:#111}
h1{margin:0 0 4px}.muted{color:#666}.row{display:flex;justify-content:space-between;margin:6px 0}
.total{font-size:1.4rem;font-weight:700;border-top:2px solid #111;padding-top:8px;margin-top:16px}
.fiscal{margin-top:28px;padding:12px 16px;border:1px dashed #888;border-radius:8px;background:#fafafa;font-family:ui-monospace,monospace}
</style></head><body>
<h1>Tax Invoice %[1]s</h1>
<div class="muted">Generated %[6]s</div>
<div class="row"><span>Bill to</span><strong>%[2]s</strong></div>
<div class="row"><span>Distributor ID</span><span>%[3]s</span></div>
<div class="row"><span>Due date</span><span>%[4]s</span></div>
<div class="row"><span>Status</span><span>%[5]s</span></div>
<div class="row total"><span>Total</span><span>UGX %[7]s</span></div>
<div class="fiscal">%[8]s</div>
</body></html>`,
		html.EscapeString(inv.ID),
		html.EscapeString(inv.Distributor),
		html.EscapeString(inv.DistributorID),
		inv.DueDate.Format("2006-01-02"),
		html.EscapeString(inv.Status),
		time.Now().UTC().Format("2006-01-02 15:04 MST"),
		fmt.Sprintf("%.0f", inv.AmountUGX),
		fiscal,
	)
}
