package models

type PermissionDescriptor struct {
	Name        string
	Description string
}

func PermissionDescriptors() []PermissionDescriptor {
	return []PermissionDescriptor{
		{Name: "dms.view_overview", Description: "View distribution tower"},
		{Name: "dms.view_outlets", Description: "View outlet universe"},
		{Name: "dms.manage_outlets", Description: "Create and update outlets"},
		{Name: "dms.view_orders", Description: "View secondary orders"},
		{Name: "dms.manage_orders", Description: "Create and update orders"},
		{Name: "dms.field_checkin", Description: "Field check-in and visit reports"},
		{Name: "dms.view_finance", Description: "View finance and invoices"},
		{Name: "dms.manage_invoices", Description: "Create invoices"},
		{Name: "dms.manage_claims", Description: "Create claims"},
		{Name: "dms.manage_promotions", Description: "Create trade promotions"},
		{Name: "dms.manage_dispatch", Description: "Create dispatch trips"},
		{Name: "dms.run_reports", Description: "Run and export reports"},
		{Name: "dms.insights.read", Description: "View distribution signals and analytics"},
		{Name: "dms.audit.read", Description: "View audit log"},
		{Name: "dms.audit.create", Description: "Append audit entries"},
		{Name: "dms.admin.read", Description: "View admin monitoring"},
		{Name: "dms.admin.update", Description: "Admin operational actions"},
	}
}

func PermissionCatalogData() map[string]any {
	keys := []string{
		"dms.view_overview", "dms.view_outlets", "dms.manage_outlets",
		"dms.view_orders", "dms.manage_orders", "dms.field_checkin",
		"dms.view_finance", "dms.manage_invoices", "dms.manage_claims",
		"dms.manage_promotions", "dms.manage_dispatch", "dms.run_reports",
		"dms.insights.read", "dms.audit.read", "dms.audit.create",
		"dms.admin.read", "dms.admin.update",
	}
	roleLabels := make(map[string]string, len(Roles))
	for id, spec := range Roles {
		roleLabels[id] = spec.Label
	}
	builtin := make([]string, 0, len(Roles))
	for id := range Roles {
		builtin = append(builtin, id)
	}
	return map[string]any{
		"modules":      []string{"distribution", "field", "logistics", "finance", "intelligence", "admin"},
		"allKeys":      keys,
		"roleLabels":   roleLabels,
		"builtinRoles": builtin,
	}
}

func BuiltinRolesPermissions() map[string]any {
	out := make(map[string]any, len(Roles))
	for id, spec := range Roles {
		out[id] = map[string]any{
			"label": spec.Label, "full": spec.Full, "pages": spec.Pages, "modals": spec.Modals,
		}
	}
	return map[string]any{"roles": out}
}
