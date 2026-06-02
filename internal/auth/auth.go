// Package auth implements per-route permission checks. The platform
// middleware verifies the inbound JWT and attaches claims; this package only
// inspects those claims.
package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/iag/dms/backend/internal/middleware"
	"github.com/alvor-technologies/iag-platform-go/apierr"
)

const strictRBACKey = "strict_rbac"

// StrictRBAC enables fail-closed permission checks when JWT permission lists
// are empty (production default).
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

func HasPerm(c *gin.Context, codename string) bool {
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
		if _, ok := middleware.Claims(c); !ok {
			apierr.Unauthorized(c, "authentication required")
			return
		}
		if !HasPerm(c, codename) {
			apierr.WriteWith(c, http.StatusForbidden, apierr.CodeForbidden, "permission denied: "+codename, gin.H{"required_permission": codename})
			return
		}
		c.Next()
	}
}

func RequireAnyPerm(codenames ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := middleware.Claims(c); !ok {
			apierr.Unauthorized(c, "authentication required")
			return
		}
		for _, codename := range codenames {
			if HasPerm(c, codename) {
				c.Next()
				return
			}
		}
		apierr.Forbidden(c, "permission denied")
	}
}

func RequireStaff() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := middleware.Claims(c)
		if !ok || claims == nil {
			apierr.Unauthorized(c, "authentication required")
			return
		}
		if !claims.IsStaff && !claims.IsSuperuser {
			apierr.Forbidden(c, "staff access required")
			return
		}
		c.Next()
	}
}

func ActorName(c *gin.Context) string {
	return middleware.ActorName(c)
}
