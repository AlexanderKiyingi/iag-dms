package models

import "strings"

type RoleSpec struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Full   string `json:"full"`
	Pages  any    `json:"pages"`
	Modals any    `json:"modals"`
}

var PageTitles = map[string]string{
	"overview": "Distribution Tower", "network": "Distributor Network", "outlets": "Outlets & Universe",
	"orders": "Secondary Orders", "routes": "Routes & Beats", "checkin": "Field Check-In",
	"journey": "Journey Planner", "field": "Field Force", "execution": "Retail Execution",
	"promo": "Trade Promotions", "claims": "Claims & Returns", "dispatch": "Dispatch & Fleet",
	"stock": "Distributor Stock", "stockwh": "Stock & Warehouse", "finance": "Finance Hub",
	"invoices": "Invoice Studio", "pricing": "Pricing Templates", "reports": "Reports Studio",
	"kpi": "KPI & Incentives", "analytics": "Analytics Studio", "forecast": "AI Demand Forecast",
}

var Roles = map[string]RoleSpec{
	"md": {
		ID: "md", Label: "MD · all access", Full: "Managing Director",
		Pages: "*", Modals: "*",
	},
	"head_ops": {
		ID: "head_ops", Label: "Head of Operations", Full: "Head of Operations",
		Pages: []string{
			"overview", "network", "outlets", "orders", "routes", "checkin", "journey", "field",
			"execution", "promo", "claims", "dispatch", "stock", "stockwh", "finance", "invoices",
			"pricing", "reports", "kpi", "analytics", "forecast",
		},
		Modals: "*",
	},
	"field_manager": {
		ID: "field_manager", Label: "Field Manager", Full: "Field Operations Manager",
		Pages: []string{
			"overview", "outlets", "routes", "checkin", "journey", "field", "execution", "kpi",
		},
		Modals: []string{"mCheckin", "mVisitReport", "mAddOutlet"},
	},
	"distributor_mgr": {
		ID: "distributor_mgr", Label: "Distributor Manager", Full: "Distributor Manager",
		Pages: []string{
			"overview", "network", "outlets", "orders", "stock", "finance", "invoices", "dispatch",
		},
		Modals: []string{"mNewInvoice", "mAddOutlet"},
	},
	"field_rep": {
		ID: "field_rep", Label: "Field Rep", Full: "Field Representative",
		Pages: []string{"checkin", "journey", "outlets"},
		Modals: []string{"mCheckin", "mVisitReport"},
	},
}

func RoleFromGroups(groups []string, isSuperuser bool) string {
	if isSuperuser {
		return "md"
	}
	for _, g := range groups {
		switch strings.ToLower(strings.ReplaceAll(strings.TrimSpace(g), "-", "_")) {
		case "md", "managing_director":
			return "md"
		case "head_ops", "head_of_operations", "head_operations":
			return "head_ops"
		case "field_manager", "field_ops_manager":
			return "field_manager"
		case "distributor_mgr", "distributor_manager":
			return "distributor_mgr"
		case "field_rep", "field_representative", "sales_rep":
			return "field_rep"
		}
	}
	return "field_rep"
}

func CanMutateRole(role string, perms []string) bool {
	if role == "md" || role == "head_ops" {
		return true
	}
	for _, p := range perms {
		if p == "*" || strings.HasPrefix(p, "dms.manage_") {
			return true
		}
	}
	return false
}

func CanManageAdmin(perms []string, isStaff, isSuperuser bool) bool {
	if isSuperuser || isStaff {
		return true
	}
	for _, p := range perms {
		if p == "dms.admin.read" || p == "dms.admin.update" || p == "*" {
			return true
		}
	}
	return false
}
