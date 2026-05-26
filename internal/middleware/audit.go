package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/iag/dms/backend/internal/store"
)

func RequestAudit(repo *store.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		if repo == nil {
			return
		}
		path := c.Request.URL.Path
		if path == "/health" || path == "/healthz" || path == "/ready" ||
			path == "/" || path == "/index.html" || strings.HasPrefix(path, "/assets/") {
			return
		}
		duration := int(time.Since(start).Milliseconds())
		_ = repo.LogAPIRequest(
			c.Request.Context(),
			c.Request.Method,
			path,
			c.Writer.Status(),
			ActorName(c),
			duration,
			c.ClientIP(),
		)
	}
}
