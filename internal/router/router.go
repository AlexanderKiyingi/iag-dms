package router

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	platformmw "github.com/alvor-technologies/iag-platform-go/middleware"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/iag/dms/backend/internal/auth"
	"github.com/iag/dms/backend/internal/config"
	"github.com/iag/dms/backend/internal/events"
	"github.com/iag/dms/backend/internal/handlers"
	"github.com/iag/dms/backend/internal/middleware"
	"github.com/iag/dms/backend/internal/store"
)

type Options struct {
	Cfg          config.Config
	Repo         *store.Repository
	PlatformAuth *middleware.PlatformAuth
	Events       *events.Bus
}

func New(opts Options) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(otelgin.Middleware(opts.Cfg.ServiceName))
	r.Use(platformmw.RequestID())
	r.Use(securityHeaders())
	r.Use(corsMiddleware(opts.Cfg.CORSOrigin))

	r.Static("/assets", "./assets")
	r.StaticFile("/", "./index.html")
	r.StaticFile("/index.html", "./index.html")

	api := &handlers.API{Repo: opts.Repo, Cfg: opts.Cfg, Events: opts.Events}

	r.GET("/healthz", api.Health)
	r.GET("/health", api.Health)
	r.GET("/ready", api.Ready)

	v1 := r.Group("/v1")
	if opts.PlatformAuth != nil {
		v1.Use(opts.PlatformAuth.AttachPrincipal())
	}
	if opts.Cfg.StrictRBAC() {
		v1.Use(auth.StrictRBAC())
	}
	v1.Use(middleware.RequestAudit(opts.Repo))

	registerPlatformRoutes(v1, api)
	registerDomainRoutes(v1, api)
	registerAuditRoutes(v1, api)
	registerAdminRoutes(v1, api)
	registerIntelligenceRoutes(v1, api)

	return r
}

func registerPlatformRoutes(v1 *gin.RouterGroup, api *handlers.API) {
	v1.GET("/bootstrap", auth.RequirePerm("dms.view_overview"), api.Bootstrap)
	v1.GET("/auth/session", auth.RequirePerm("dms.view_overview"), api.Session)
	v1.GET("/permissions/catalog", auth.RequirePerm("dms.view_overview"), api.PermissionsCatalog)
	v1.GET("/permissions/builtin", auth.RequirePerm("dms.view_overview"), api.PermissionsBuiltin)
	v1.POST("/permissions/check", auth.RequirePerm("dms.view_overview"), api.PermissionsCheck)
	v1.GET("/permissions/me", auth.RequirePerm("dms.view_overview"), api.PermissionsMe)
	v1.GET("/lookups/:kind", auth.RequirePerm("dms.view_overview"), api.Lookups)
	v1.GET("/search", auth.RequirePerm("dms.view_overview"), api.Search)
	v1.GET("/notifications", auth.RequirePerm("dms.view_overview"), api.Notifications)
	v1.GET("/platform/status", auth.RequireStaff(), api.PlatformStatus)
}

func registerDomainRoutes(v1 *gin.RouterGroup, api *handlers.API) {
	v1.GET("/overview", auth.RequirePerm("dms.view_overview"), api.Overview)

	v1.GET("/distributors", auth.RequirePerm("dms.view_overview"), api.ListDistributors)
	v1.GET("/distributors/:id", auth.RequirePerm("dms.view_overview"), api.GetDistributor)

	v1.GET("/outlets/stats", auth.RequirePerm("dms.view_outlets"), api.OutletStats)
	v1.GET("/outlets", auth.RequirePerm("dms.view_outlets"), api.ListOutlets)
	v1.POST("/outlets", auth.RequirePerm("dms.manage_outlets"), api.CreateOutlet)
	v1.GET("/outlets/:id", auth.RequirePerm("dms.view_outlets"), api.GetOutlet)
	v1.PATCH("/outlets/:id", auth.RequirePerm("dms.manage_outlets"), api.PatchOutlet)

	v1.GET("/orders/stats", auth.RequirePerm("dms.view_orders"), func(c *gin.Context) {
		c.JSON(http.StatusOK, api.Repo.OrdersStats())
	})
	v1.GET("/orders/board", auth.RequirePerm("dms.view_orders"), api.OrdersBoard)
	v1.GET("/orders", auth.RequirePerm("dms.view_orders"), api.ListOrders)
	v1.POST("/orders", auth.RequirePerm("dms.manage_orders"), api.CreateOrder)
	v1.GET("/orders/:id", auth.RequirePerm("dms.view_orders"), api.GetOrder)
	v1.PATCH("/orders/:id/status", auth.RequirePerm("dms.manage_orders"), api.PatchOrderStatus)

	v1.GET("/routes/stats", auth.RequirePerm("dms.view_overview"), api.RoutesStats)
	v1.GET("/beats", auth.RequirePerm("dms.view_overview"), api.ListBeats)
	v1.GET("/beats/:id", auth.RequirePerm("dms.view_overview"), api.GetBeat)

	v1.GET("/field/reps", auth.RequirePerm("dms.field_checkin"), api.ListReps)
	v1.GET("/field/check-ins/stats", auth.RequirePerm("dms.field_checkin"), api.CheckInStats)
	v1.GET("/field/check-ins", auth.RequirePerm("dms.field_checkin"), api.ListCheckIns)
	v1.POST("/field/check-ins", auth.RequirePerm("dms.field_checkin"), api.CreateCheckIn)
	v1.PATCH("/field/check-ins/:id", auth.RequirePerm("dms.field_checkin"), api.CompleteCheckIn)
	v1.GET("/field/visit-reports", auth.RequirePerm("dms.field_checkin"), api.ListVisitReports)
	v1.POST("/field/visit-reports", auth.RequirePerm("dms.field_checkin"), api.CreateVisitReport)
	v1.GET("/field/journey", auth.RequirePerm("dms.field_checkin"), api.Journey)
	v1.GET("/field/execution", auth.RequirePerm("dms.field_checkin"), api.ListExecution)

	v1.GET("/promotions", auth.RequirePerm("dms.view_overview"), api.ListPromotions)
	v1.POST("/promotions", auth.RequirePerm("dms.manage_promotions"), api.CreatePromotion)
	v1.GET("/claims", auth.RequirePerm("dms.view_overview"), api.ListClaims)
	v1.POST("/claims", auth.RequirePerm("dms.manage_claims"), api.CreateClaim)
	v1.GET("/dispatch", auth.RequirePerm("dms.view_orders"), api.ListDispatches)
	v1.POST("/dispatch", auth.RequirePerm("dms.manage_dispatch"), api.CreateDispatch)

	v1.GET("/stock/distributor", auth.RequirePerm("dms.view_overview"), api.ListStock)
	v1.GET("/stock/skus", auth.RequirePerm("dms.view_overview"), api.ListSKUs)

	v1.GET("/finance/summary", auth.RequirePerm("dms.view_finance"), api.FinanceSummary)
	v1.GET("/invoices", auth.RequirePerm("dms.view_finance"), api.ListInvoices)
	v1.GET("/invoices/:id", auth.RequirePerm("dms.view_finance"), api.GetInvoice)
	v1.POST("/invoices", auth.RequirePerm("dms.manage_invoices"), api.CreateInvoice)

	v1.GET("/pricing/templates", auth.RequirePerm("dms.view_finance"), api.ListPricing)
	v1.GET("/reports/templates", auth.RequirePerm("dms.run_reports"), api.ListReports)
	v1.POST("/reports/run", auth.RequirePerm("dms.run_reports"), api.RunReport)
	v1.POST("/exports/:page", auth.RequirePerm("dms.run_reports"), api.ExportPage)

	v1.GET("/kpi/board", auth.RequirePerm("dms.insights.read"), api.KPIBoard)
	v1.GET("/analytics/summary", auth.RequirePerm("dms.insights.read"), api.Analytics)
	v1.GET("/forecast", auth.RequirePerm("dms.insights.read"), api.Forecast)
}

func registerAuditRoutes(v1 *gin.RouterGroup, api *handlers.API) {
	v1.GET("/audit", auth.RequirePerm("dms.audit.read"), api.ListAudit)
	v1.POST("/audit", auth.RequirePerm("dms.audit.create"), api.AppendAuditEntry)
	v1.GET("/audit/:id", auth.RequirePerm("dms.audit.read"), api.GetAuditEntry)
}

func registerAdminRoutes(v1 *gin.RouterGroup, api *handlers.API) {
	admin := v1.Group("/admin")
	admin.Use(auth.RequireStaff())
	{
		admin.GET("/audit", auth.RequirePerm("dms.audit.read"), api.ListAudit)
		admin.GET("/audit-logs", auth.RequirePerm("dms.admin.read"), api.AdminAuditLogs)
		admin.GET("/monitoring/summary", auth.RequirePerm("dms.admin.read"), api.AdminMonitoringSummary)
		admin.GET("/monitoring/activity", auth.RequirePerm("dms.admin.read"), api.AdminMonitoringActivity)
	}
}

func registerIntelligenceRoutes(v1 *gin.RouterGroup, api *handlers.API) {
	v1.GET("/insights/signals", auth.RequirePerm("dms.insights.read"), api.InsightsSignals)
}

// securityHeaders emits the platform-wide baseline browser security header
// set. The Content-Security-Policy uses the permissive shape because this
// service mounts a static SPA (/, /index.html, /assets/*) — connect-src /
// img-src / style-src / font-src are allow-listed to 'self' (plus data: for
// inline images and fonts) so the bundled frontend can fetch its own assets.
//
// HSTS is gated on TLS termination (direct TLS or trusted X-Forwarded-Proto)
// so plain-HTTP dev environments (http://localhost) do not lock browsers
// into HTTPS for the developer's whole domain.
func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'; connect-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; font-src 'self' data:; script-src 'self'")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=(), interest-cohort=()")
		c.Header("X-XSS-Protection", "1; mode=block")
		if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}
		c.Next()
	}
}

// cors helpers below

func corsMiddleware(allowed string) gin.HandlerFunc {
	allowAny := allowed == "" || allowed == "*"
	allowedOrigins := splitAllowedOrigins(allowed)
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if allowAny || (origin != "" && originAllowed(origin, allowedOrigins)) {
			if origin != "" {
				c.Header("Access-Control-Allow-Origin", origin)
			} else if allowAny {
				c.Header("Access-Control-Allow-Origin", "*")
			}
			c.Header("Vary", "Origin")
		}
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, If-Match, X-Requested-With, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "ETag, X-Request-ID")
		c.Header("Access-Control-Max-Age", "86400")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func splitAllowedOrigins(allowed string) []string {
	if allowed == "" || allowed == "*" {
		return nil
	}
	parts := strings.Split(allowed, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func originAllowed(origin string, allowed []string) bool {
	for _, candidate := range allowed {
		if origin == candidate {
			return true
		}
	}
	return false
}
