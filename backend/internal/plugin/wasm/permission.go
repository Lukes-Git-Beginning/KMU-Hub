package wasm

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/plugin/repository"
)

// PermissionEnforcer checks plugin permissions before host API calls
type PermissionEnforcer struct {
	repo *repository.PermissionRepository
}

// NewPermissionEnforcer creates a new permission enforcer
func NewPermissionEnforcer(repo *repository.PermissionRepository) *PermissionEnforcer {
	return &PermissionEnforcer{repo: repo}
}

// Check verifies that a plugin installation has the required permission
func (e *PermissionEnforcer) Check(ctx context.Context, installationID uuid.UUID, permission string) error {
	has, err := e.repo.HasPermission(ctx, installationID, permission)
	if err != nil {
		slog.Warn("permission check failed",
			"installation_id", installationID,
			"permission", permission,
			"error", err,
		)
		return fmt.Errorf("permission check failed: %w", err)
	}
	if !has {
		return fmt.Errorf("plugin does not have permission: %s", permission)
	}
	return nil
}

// RequiredPermission maps host API function names to required permissions
var RequiredPermission = map[string]string{
	"crm_get_contact":   "crm:read",
	"crm_list_contacts": "crm:read",
	"crm_get_company":   "crm:read",
	"work_get_task":     "work:read",
	"work_create_task":  "work:write",
	"biz_get_invoice":   "biz:read",
	"custom_field_get":  "crm:read",
	"custom_field_set":  "crm:write",
	"events_subscribe":  "events:read",
}

// PermissionForFunction returns the required permission for a host function
func PermissionForFunction(funcName string) string {
	if perm, ok := RequiredPermission[funcName]; ok {
		return perm
	}
	// KV store, config, and logging don't require special permissions
	if strings.HasPrefix(funcName, "kv_") || strings.HasPrefix(funcName, "log_") || funcName == "config_get" {
		return ""
	}
	return ""
}
