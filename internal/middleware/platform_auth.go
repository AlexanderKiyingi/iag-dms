package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/alvor-technologies/iag-platform-go/authclient"
	"github.com/iag/dms/backend/internal/ctxkeys"
	"github.com/iag/dms/backend/internal/platformauth"
)

const (
	HeaderUserID        = "X-IAG-User-Id"
	HeaderEmail         = "X-IAG-Email"
	HeaderGroups        = "X-IAG-Groups"
	HeaderRoles         = "X-IAG-Roles"
	HeaderPermissions   = "X-IAG-Permissions"
	HeaderIsSuperuser   = "X-IAG-Is-Superuser"
	HeaderIsStaff       = "X-IAG-Is-Staff"
	HeaderGatewaySecret = "X-IAG-Gateway-Secret"
)

type PlatformAuth struct {
	authMode      string
	gatewaySecret string
	verifier      *platformauth.Verifier
}

func NewPlatformAuth(mode string, gatewaySecret string, verifier *platformauth.Verifier) *PlatformAuth {
	return &PlatformAuth{authMode: mode, gatewaySecret: gatewaySecret, verifier: verifier}
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
		switch m.authMode {
		case "gateway":
			m.fromGateway(c)
		case "jwt":
			m.fromJWT(c)
		case "none":
			c.Next()
		default:
			c.Next()
		}
	}
}

func (m *PlatformAuth) fromGateway(c *gin.Context) {
	if m.gatewaySecret != "" && c.GetHeader(HeaderGatewaySecret) != m.gatewaySecret {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	sub := c.GetHeader(HeaderUserID)
	if sub == "" {
		c.Next()
		return
	}
	userID, err := uuid.Parse(sub)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
		return
	}
	groups := splitHeaderList(c.GetHeader(HeaderGroups))
	if len(groups) == 0 {
		groups = splitHeaderList(c.GetHeader(HeaderRoles))
	}
	claims := &authclient.Claims{
		Email:       c.GetHeader(HeaderEmail),
		IsSuperuser: strings.EqualFold(c.GetHeader(HeaderIsSuperuser), "true"),
		IsStaff:     strings.EqualFold(c.GetHeader(HeaderIsStaff), "true"),
		Groups:      groups,
		Permissions: splitHeaderList(c.GetHeader(HeaderPermissions)),
	}
	claims.Subject = sub
	setPrincipal(c, userID, claims)
	c.Next()
}

func (m *PlatformAuth) fromJWT(c *gin.Context) {
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
	setPrincipal(c, uid, claims)
	c.Next()
}

func setPrincipal(c *gin.Context, userID uuid.UUID, claims *authclient.Claims) {
	c.Set(ctxkeys.UserID, userID)
	c.Set(ctxkeys.Claims, claims)
	c.Set(ctxkeys.Permissions, claims.Permissions)
}

func splitHeaderList(value string) []string {
	if value == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(value, ",") {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
