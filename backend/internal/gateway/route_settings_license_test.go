package gateway

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kmuhub/kmuhub/internal/featureflag"
	settingsv1 "github.com/kmuhub/kmuhub/proto/settings/v1"
)

// The deployment-wide flag is an upper bound on a tenant's activation: a stored
// active=true for a module this deployment does not run must not reach the UI as
// active, or the licence tab shows a module the app has no routes for.
func TestToTenantModuleJSON_FlagMasksStoredActivation(t *testing.T) {
	flags := featureflag.NewRegistry().Load(func(key string) string {
		if key == "COSMI_MODULE_HELPDESK_ENABLED" {
			return "true"
		}
		return "false"
	})
	sr := NewSettingsRoutes(nil, flags)

	helpdesk := sr.toTenantModuleJSON(&settingsv1.TenantModule{
		ModuleId: "helpdesk", Group: "industry", Active: true, AssignedSeats: 5,
		FlagKey: "modules.helpdesk",
	})
	assert.True(t, helpdesk.Active)
	assert.Equal(t, int32(5), helpdesk.AssignedSeats)

	// Same stored state, flag off — masked to inactive, and its seats with it.
	wiki := sr.toTenantModuleJSON(&settingsv1.TenantModule{
		ModuleId: "wiki", Group: "tools", Active: true, AssignedSeats: 9,
		FlagKey: "modules.wiki",
	})
	assert.False(t, wiki.Active)
	assert.Zero(t, wiki.AssignedSeats)

	// No flag: ships in every deployment, activation decides alone.
	crm := sr.toTenantModuleJSON(&settingsv1.TenantModule{
		ModuleId: "crm", Group: "core", Active: true, AssignedSeats: 14,
	})
	assert.True(t, crm.Active)
	assert.Equal(t, int32(14), crm.AssignedSeats)
}

// moduleAvailable resolves the flag from the catalogue, not from the response —
// the PATCH handler has to answer the question before it has ever called the
// settings service.
func TestModuleAvailable_UsesCatalogue(t *testing.T) {
	flags := featureflag.NewRegistry().Load(func(key string) string {
		if key == "COSMI_MODULE_BUCHHALTUNG_ENABLED" {
			return "true"
		}
		return "false"
	})
	sr := NewSettingsRoutes(nil, flags)

	assert.True(t, sr.moduleAvailable("finance"), "flag on")
	assert.False(t, sr.moduleAvailable("berichte"), "flag off")
	assert.True(t, sr.moduleAvailable("calendar"), "no flag at all")
	assert.True(t, sr.moduleAvailable("not-a-module"), "unknown ids are rejected earlier, not here")
}
