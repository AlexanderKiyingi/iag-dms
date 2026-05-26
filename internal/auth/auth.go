package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/iag/dms/backend/internal/middleware"
)

const (
	AuthModeNone  = "auth_mode_none"
	strictRBACKey = "strict_rbac"
)

// StrictRBAC enables fail-closed permission checks when JWT permission lists are empty.
func StrictRBAC() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(strictRBACKey, true)
		c.Next()
	}
}

func isStrictRBAC(c *gin.Context) bool {
	v, ok := c.Get(strictRBACKey)
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}

func SetAuthModeNone(c *gin.Context) {
	c.Set(AuthModeNone, true)
}

func isAuthDisabled(c *gin.Context) bool {
	if v, ok := c.Get(AuthModeNone); ok {
		if b, ok := v.(bool); ok && b {
			return true
		}
	}
	return false
}

func HasPerm(c *gin.Context, codename string) bool {
	if isAuthDisabled(c) {
		return true
	}
	claims, ok := middleware.Claims(c)
	if !ok || claims == nil {
		return false
	}
	if claims.IsSuperuser || claims.IsStaff {
		return true
	}
	if claims.HasPermission(codename) {
		return true
	}
	perms := claims.Permissions
	if len(perms) == 0 {
		return !isStrictRBAC(c)
	}
	for _, p := range perms {
		if p == "*" || p == codename {
			return true
		}
	}
	return false
}

func RequirePerm(codename string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isAuthDisabled(c) {
			c.Next()
			return
		}
		if _, ok := middleware.Claims(c); !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		if !HasPerm(c, codename) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permission denied", "permission": codename})
			return
		}
		c.Next()
	}
}

func RequireAnyPerm(codenames ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isAuthDisabled(c) {
			c.Next()
			return
		}
		if _, ok := middleware.Claims(c); !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		for _, codename := range codenames {
			if HasPerm(c, codename) {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permission denied"})
	}
}

func RequireStaff() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isAuthDisabled(c) {
			c.Next()
			return
		}
		claims, ok := middleware.Claims(c)
		if !ok || claims == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		if !claims.IsStaff && !claims.IsSuperuser {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "staff access required"})
			return
		}
		c.Next()
	}
}

func ActorName(c *gin.Context) string {
	return middleware.ActorName(c)
}
