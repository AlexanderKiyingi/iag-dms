package models

import "time"

type AuditEntry struct {
	ID       string    `json:"id"`
	LoggedAt time.Time `json:"logged_at"`
	UserName string    `json:"user_name"`
	Action   string    `json:"action"`
	Detail   string    `json:"detail"`
}

type PermissionCheckInput struct {
	Keys []string `json:"keys"`
}

type PermissionCheckResult struct {
	Allowed map[string]bool `json:"allowed"`
}

type PermissionContext struct {
	Role           string   `json:"role"`
	Email          string   `json:"email"`
	Name           string   `json:"name"`
	Permissions    []string `json:"permissions"`
	CanMutate      bool     `json:"canMutate"`
	CanManageAdmin bool     `json:"canManageAdmin"`
	IsStaff        bool     `json:"isStaff"`
}
