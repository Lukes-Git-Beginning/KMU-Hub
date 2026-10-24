package gateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// The system roles member and manager hold settings:write and calendars:admin
// for their OWN preferences and calendars. The routes below are tenant-wide,
// so the gateway (or, where noted, the service) must not let those keys alone
// through.
//
// The permission sets are the relevant slice of what the seeded system roles
// carry (role_permissions, migration 000256 onward): member/manager have the
// coarse keys, only admin (and it_admin) has settings:tenant:manage.
var (
	memberSettingsPerms = []string{"settings:read", "settings:write", "calendars:read", "calendars:write", "calendars:admin", "settings:personal:manage"}
	adminSettingsPerms  = append([]string{"settings:tenant:manage"}, memberSettingsPerms...)
)

// TestTenantWideSettingsGuards_MemberVsAdmin pins which tenant-wide writes a
// member token is stopped at the gateway, and that the personal settings
// routes the same key protects stay open.
//
// The registry is empty, so a request that passes every guard dies in the
// handler with 503 (service not registered) — the status is spelled out so a
// mistyped path (404) cannot pass as "not forbidden".
func TestTenantWideSettingsGuards_MemberVsAdmin(t *testing.T) {
	videoRouter := chi.NewRouter()
	NewVideoRoutes(emptyRegistry(), "", "").RegisterRoutes(videoRouter, guardTestAuth)

	calendarRouter := chi.NewRouter()
	NewCalendarRoutes(emptyRegistry()).RegisterRoutes(calendarRouter, guardTestAuth)

	settingsRouter := chi.NewRouter()
	newModuleAccessRoutes(emptyRegistry()).RegisterRoutes(settingsRouter, guardTestAuth)

	const (
		allowed = http.StatusServiceUnavailable
		denied  = http.StatusForbidden
	)

	tests := []struct {
		name       string
		router     chi.Router
		method     string
		path       string
		perms      []string
		wantStatus int
	}{
		// Tenant-wide, narrowed at the gateway.
		{"presence config, member denied", videoRouter, "PUT", "/api/v1/video/presence/config", memberSettingsPerms, denied},
		{"presence config, admin allowed", videoRouter, "PUT", "/api/v1/video/presence/config", adminSettingsPerms, allowed},
		{"presence config, tenant key without settings:write denied", videoRouter, "PUT", "/api/v1/video/presence/config", []string{"settings:tenant:manage"}, denied},
		{"presence config read, member allowed", videoRouter, "GET", "/api/v1/video/presence/config", memberSettingsPerms, allowed},
		{"holiday seed, member denied", calendarRouter, "POST", "/api/v1/calendar/holidays/seed", memberSettingsPerms, denied},
		{"holiday seed, admin allowed", calendarRouter, "POST", "/api/v1/calendar/holidays/seed", adminSettingsPerms, allowed},
		{"holiday list, member allowed", calendarRouter, "GET", "/api/v1/calendar/holidays", memberSettingsPerms, allowed},

		// Personal settings: the same settings:write key, must stay open for member.
		{"user settings write, member allowed", settingsRouter, "PUT", "/api/v1/settings/work/user", memberSettingsPerms, allowed},
		{"user preferences write, member allowed", settingsRouter, "PUT", "/api/v1/users/preferences/", memberSettingsPerms, allowed},

		// Branding and tenant settings pass the gateway on settings:write on
		// purpose: the settings service admits admin OR the module-lead of that
		// module (callerMayWriteTenantSettings), and a module-lead is usually a
		// member. A member who is neither is rejected there — proven against a
		// real Postgres in internal/settings
		// (TestTenantWrites_MemberAndManagerRolesRejectedAdminAllowed).
		{"branding write, member reaches the service", settingsRouter, "PUT", "/api/v1/admin/branding/", memberSettingsPerms, allowed},
		{"tenant settings write, member reaches the service", settingsRouter, "PUT", "/api/v1/settings/work/tenant", memberSettingsPerms, allowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(`{}`))
			req = withAuth(req, "user-123", testTenantID)
			req = withPermissions(req, tt.perms...)
			rec := httptest.NewRecorder()
			tt.router.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("%s %s with %v: got %d, want %d", tt.method, tt.path, tt.perms, rec.Code, tt.wantStatus)
			}
		})
	}
}
