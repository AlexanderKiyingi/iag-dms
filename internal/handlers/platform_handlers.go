package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/iag/dms/backend/internal/auth"
	"github.com/iag/dms/backend/internal/middleware"
	"github.com/iag/dms/backend/internal/models"
	"github.com/alvor-technologies/iag-platform-go/apierr"
)

func (h *API) sessionFromContext(c *gin.Context) map[string]any {
	role := "field_rep"
	email := "dev@iag.local"
	name := "Developer"
	perms := []string{}
	if claims, ok := middleware.Claims(c); ok && claims != nil {
		email = claims.Email
		name = claims.Name
		if name == "" {
			name = email
		}
		role = models.RoleFromGroups(claims.Groups, claims.IsSuperuser)
		perms = claims.Permissions
	}
	spec, _ := models.Roles[role]
	return map[string]any{
		"email": email, "name": name, "role": role,
		"role_label": spec.Label, "role_full": spec.Full,
		"pages": spec.Pages, "modals": spec.Modals, "permissions": perms,
	}
}

func (h *API) permissionContext(c *gin.Context) models.PermissionContext {
	sess := h.sessionFromContext(c)
	role, _ := sess["role"].(string)
	email, _ := sess["email"].(string)
	name, _ := sess["name"].(string)
	perms, _ := sess["permissions"].([]string)
	isStaff, isSuper := false, false
	if claims, ok := middleware.Claims(c); ok && claims != nil {
		isStaff = claims.IsStaff
		isSuper = claims.IsSuperuser
		if len(perms) == 0 {
			perms = claims.Permissions
		}
	}
	return models.PermissionContext{
		Role: role, Email: email, Name: name, Permissions: perms,
		CanMutate:      models.CanMutateRole(role, perms) || isSuper,
		CanManageAdmin: models.CanManageAdmin(perms, isStaff, isSuper),
		IsStaff:        isStaff || isSuper,
	}
}

func (h *API) Bootstrap(c *gin.Context) {
	sess := h.sessionFromContext(c)
	c.JSON(http.StatusOK, gin.H{
		"service":     h.Cfg.ServiceName,
		"version":     "0.1.0",
		"api_prefix":  h.Cfg.GatewayAPIPrefix + "/v1",
		"public_api":  h.Cfg.PublicAPIURL,
		"session":     sess,
		"permissions": h.permissionContext(c),
		"pages":       h.Repo.Pages(),
		"page_titles": models.PageTitles,
		"roles":       models.Roles,
		"sync_status": "connected",
		"modules":     []string{"distribution", "field", "logistics", "finance", "intelligence"},
	})
}

func (h *API) Session(c *gin.Context) {
	c.JSON(http.StatusOK, h.sessionFromContext(c))
}

func (h *API) PermissionsCatalog(c *gin.Context) {
	c.JSON(http.StatusOK, models.PermissionCatalogData())
}

func (h *API) PermissionsBuiltin(c *gin.Context) {
	c.JSON(http.StatusOK, models.BuiltinRolesPermissions())
}

func (h *API) PermissionsCheck(c *gin.Context) {
	var in models.PermissionCheckInput
	if err := c.ShouldBindJSON(&in); err != nil {
		badRequest(c, "invalid body")
		return
	}
	allowed := make(map[string]bool, len(in.Keys))
	for _, key := range in.Keys {
		allowed[key] = auth.HasPerm(c, key)
	}
	c.JSON(http.StatusOK, models.PermissionCheckResult{Allowed: allowed})
}

func (h *API) PermissionsMe(c *gin.Context) {
	c.JSON(http.StatusOK, h.permissionContext(c))
}

func (h *API) Lookups(c *gin.Context) {
	items, err := h.Repo.Lookups(c.Request.Context(), c.Param("kind"))
	if err != nil {
		apierr.JSONStatus(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *API) recordAudit(c *gin.Context, action, detail string) {
	if h.Repo == nil {
		return
	}
	_, _ = h.Repo.AppendAudit(c.Request.Context(), action, detail, auth.ActorName(c))
}
