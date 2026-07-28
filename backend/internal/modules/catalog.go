// Package modules holds the catalogue of Cosmi modules a tenant can book.
//
// It sits below both the settings service (which persists per-tenant activation)
// and the gateway (which masks an activation the deployment-wide feature flag
// does not cover), so the module ids, their grouping and their flag keys are
// defined exactly once. The ids mirror ModuleId in desktop/src/renderer/src/lib/
// pricing.ts — a mismatch there shows up as a module the admin hub cannot render.
package modules

// Group is the UI grouping of a module in the licence tab (ModuleGroupId in
// desktop/src/renderer/src/api/admin-types.ts).
type Group string

const (
	GroupCore     Group = "core"
	GroupComm     Group = "comm"
	GroupTeam     Group = "team"
	GroupIndustry Group = "industry"
	GroupTools    Group = "tools"
)

// Module is one bookable Cosmi module.
type Module struct {
	// ID is the module identifier shared with the frontend.
	ID string
	// Group is the UI grouping.
	Group Group
	// FlagKey is the deployment-wide feature flag that gates the module, or ""
	// for a module that ships in every deployment. The flag is an upper bound:
	// a tenant cannot activate what the deployment does not run.
	FlagKey string
}

// Catalog lists every bookable module in display order.
var Catalog = []Module{
	// Kern
	{ID: "crm", Group: GroupCore},
	{ID: "tasks", Group: GroupCore},
	{ID: "finance", Group: GroupCore, FlagKey: "modules.buchhaltung"},
	{ID: "calendar", Group: GroupCore},
	{ID: "documents", Group: GroupCore},
	// Kommunikation
	{ID: "chat", Group: GroupComm},
	{ID: "meetings", Group: GroupComm, FlagKey: "modules.video"},
	{ID: "mail", Group: GroupComm},
	{ID: "dialer", Group: GroupComm},
	// Team
	{ID: "team", Group: GroupTeam},
	{ID: "zeiterfassung", Group: GroupTeam},
	{ID: "schichten", Group: GroupTeam, FlagKey: "modules.schichten"},
	// Branche / Projekte
	{ID: "projects", Group: GroupIndustry},
	{ID: "vertraege", Group: GroupIndustry, FlagKey: "modules.vertraege"},
	{ID: "helpdesk", Group: GroupIndustry, FlagKey: "modules.helpdesk"},
	{ID: "inventar", Group: GroupIndustry, FlagKey: "modules.inventar"},
	{ID: "einkauf", Group: GroupIndustry, FlagKey: "modules.einkauf"},
	{ID: "fuhrpark", Group: GroupIndustry, FlagKey: "modules.fuhrpark"},
	{ID: "produktion", Group: GroupIndustry, FlagKey: "modules.produktion"},
	{ID: "vermietung", Group: GroupIndustry, FlagKey: "modules.vermietung"},
	// Tools
	{ID: "berichte", Group: GroupTools, FlagKey: "modules.berichte"},
	{ID: "formulare", Group: GroupTools, FlagKey: "modules.formulare"},
	{ID: "wiki", Group: GroupTools, FlagKey: "modules.wiki"},
	{ID: "rapporte", Group: GroupTools, FlagKey: "modules.rapporte"},
}

// byID indexes Catalog for lookups.
var byID = func() map[string]Module {
	m := make(map[string]Module, len(Catalog))
	for _, mod := range Catalog {
		m[mod.ID] = mod
	}
	return m
}()

// Get returns the catalogue entry for id.
func Get(id string) (Module, bool) {
	mod, ok := byID[id]
	return mod, ok
}

// FlagKey returns the feature flag gating the module, or "" when the module is
// unknown or ships unconditionally. Callers treat "" as "no flag to check".
func FlagKey(id string) string {
	return byID[id].FlagKey
}
