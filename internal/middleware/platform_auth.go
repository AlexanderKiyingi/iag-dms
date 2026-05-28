// Package middleware implements Bearer+aud authentication for inbound DMS
// requests. The gateway-header trust path (X-IAG-* + GATEWAY_INTERNAL_SECRET)
// has been removed — every request must carry a verifiable JWT with
// aud=iag.dms.
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/iag/dms/backend/internal/ctxkeys"
	"github.com/iag/dms/backend/internal/platformauth"
)

type PlatformAuth struct {
	verifier *platformauth.Verifier
}

func NewPlatformAuth(verifier *platformauth.Verifier) *PlatformAuth {
	return &PlatformAuth{verifier: verifier}
}

func isPublicProbePath(path string) bool {
	switch path {
	case "/health", "/healthz", "/ready", "/", "/index.html":
		return true
	default:
		return strings.HasPrefix(path, "/assets/")
	}
}

func (m *PlatformAuth) AttachPrincipal() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isPublicProbePath(c.Request.URL.Path) {
			c.Next()
			return
		}
		if m.verifier == nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "jwt verifier not configured"})
			return
		}
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		claims, err := m.verifier.Verify(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		uid, _ := uuid.Parse(claims.Subject)
		c.Set(ctxkeys.UserID, uid)
		c.Set(ctxkeys.Claims, claims)
		c.Set(ctxkeys.Permissions, claims.Permissions)
		c.Next()
	}
}
