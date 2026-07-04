package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/iag/dms/backend/internal/store"
)

// SendMessage delivers an in-app message to a recipient's notifications inbox
// via the notifications service (channel in_app, alert-in-app template). The
// recipient reads it through GET /inbox and the realtime stream.
func (h *API) SendMessage(c *gin.Context) {
	var body struct {
		To      string `json:"to"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.To == "" || body.Body == "" {
		badRequest(c, "to and body are required")
		return
	}
	subject := body.Subject
	if subject == "" {
		subject = "New message"
	}
	if h.Events == nil || !h.Events.Enabled() {
		apierrStatus(c, http.StatusServiceUnavailable, "messaging is unavailable (event bus disabled)")
		return
	}
	h.Events.PublishAlert(c.Request.Context(), "in_app", body.To, "alert-in-app", map[string]string{
		"Title":   subject,
		"Message": body.Body,
	}, body.To)
	h.recordAudit(c, "SendMessage", store.AuditDetail("message", body.To, "sent"))
	c.JSON(http.StatusAccepted, gin.H{"status": "queued", "to": body.To})
}
