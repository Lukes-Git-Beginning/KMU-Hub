package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authv1 "github.com/kmuhub/kmuhub/proto/auth/v1"
)

// --- HandleGetMyPermissions / HandleGetUserPermissions ---

func TestHandleGetMyPermissions_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleGetMyPermissions)
}

func TestHandleGetUserPermissions_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/admin/users/550e8400-e29b-41d4-a716-446655440000/permissions", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetUserPermissions(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetUserPermissions_InvalidUUID(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/admin/users/bad-id/permissions", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetUserPermissions(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// TestToEffectivePermissionsBody_WireShape pins the one thing the frontend
// cannot recover from: capabilities travels as a map keyed by the capability
// key, while the proto carries a repeated list. The mapping is also where the
// camelCase isSystem lives — the frontend type spells it that way, so a
// well-meant "correction" to snake_case would silently mark every preset as
// editable.
func TestToEffectivePermissionsBody_WireShape(t *testing.T) {
	body := toEffectivePermissionsBody(&authv1.GetEffectivePermissionsResponse{
		Roles: []*authv1.EffectiveRole{
			{Id: "11111111-1111-1111-1111-111111111111", Name: "manager", IsSystem: true, Color: "hsl(217 91% 60%)"},
		},
		Capabilities: []*authv1.EffectiveCapability{
			{Key: "work:task:edit", Scope: "team", Sources: []string{"11111111-1111-1111-1111-111111111111"}},
			{Key: "crm:contact:read", Scope: "all", Sources: []string{"11111111-1111-1111-1111-111111111111"}},
		},
	})

	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	want := `{"permissions":{"roles":[{"id":"11111111-1111-1111-1111-111111111111","name":"manager","isSystem":true,"color":"hsl(217 91% 60%)"}],` +
		`"capabilities":{"crm:contact:read":{"scope":"all","sources":["11111111-1111-1111-1111-111111111111"]},` +
		`"work:task:edit":{"scope":"team","sources":["11111111-1111-1111-1111-111111111111"]}}}}`
	if string(raw) != want {
		t.Errorf("wire shape drifted\n got: %s\nwant: %s", raw, want)
	}
}

// TestToEffectivePermissionsBody_EmptyIsContainerNotNull guards the answer for
// an account without a single role. The frontend iterates roles and looks keys
// up in capabilities unconditionally; null for either would break the RBAC
// surfaces on exactly the accounts an admin is most likely to inspect.
func TestToEffectivePermissionsBody_EmptyIsContainerNotNull(t *testing.T) {
	raw, err := json.Marshal(toEffectivePermissionsBody(&authv1.GetEffectivePermissionsResponse{}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	want := `{"permissions":{"roles":[],"capabilities":{}}}`
	if string(raw) != want {
		t.Errorf("empty response = %s, want %s", raw, want)
	}
}

// TestToEffectivePermissionsBody_NilSourcesMarshalAsArray covers the proto3
// quirk behind the previous test: an omitted repeated field arrives as nil, and
// nil marshals to null, not [].
func TestToEffectivePermissionsBody_NilSourcesMarshalAsArray(t *testing.T) {
	raw, err := json.Marshal(toEffectivePermissionsBody(&authv1.GetEffectivePermissionsResponse{
		Capabilities: []*authv1.EffectiveCapability{{Key: "work:task:read", Scope: "own"}},
	}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	want := `{"permissions":{"roles":[],"capabilities":{"work:task:read":{"scope":"own","sources":[]}}}}`
	if string(raw) != want {
		t.Errorf("nil sources = %s, want %s", raw, want)
	}
}

// TestToEffectivePermissionsBody_OverridesWireShape pins the two R-6 fields
// against rbac-types.ts. An allow override carries the "override" sentinel as
// the last entry of sources — it is not a role id and resolves against no
// role, which is exactly how EffectivePermissionsView tells a hand-granted
// right from an inherited one. deniedByOverride keeps the scope the roles
// would have given so the row can be shown struck through.
func TestToEffectivePermissionsBody_OverridesWireShape(t *testing.T) {
	raw, err := json.Marshal(toEffectivePermissionsBody(&authv1.GetEffectivePermissionsResponse{
		Capabilities: []*authv1.EffectiveCapability{
			{Key: "work:task:edit", Scope: "own", Sources: []string{"11111111-1111-1111-1111-111111111111", "override"}},
		},
		HasOverrides: true,
		DeniedByOverride: []*authv1.DeniedCapability{
			{Key: "crm:contact:delete", RoleScope: "all", Sources: []string{"11111111-1111-1111-1111-111111111111"}},
		},
	}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	want := `{"permissions":{"roles":[],"capabilities":{"work:task:edit":{"scope":"own","sources":["11111111-1111-1111-1111-111111111111","override"]}},` +
		`"hasOverrides":true,` +
		`"deniedByOverride":[{"key":"crm:contact:delete","roleScope":"all","sources":["11111111-1111-1111-1111-111111111111"]}]}}`
	if string(raw) != want {
		t.Errorf("override wire shape drifted\n got: %s\nwant: %s", raw, want)
	}
}

// TestToEffectivePermissionsBody_OmitsOverrideFieldsWhenThereAreNone is the
// regression guard for every account that carries no override, and for
// ?base=1: both fields drop out of the JSON entirely, so the answer stays
// byte-identical to the one this route gave before R-6 existed. The frontend
// defaults them (`hasOverrides = false`, `deniedByOverride = []`).
func TestToEffectivePermissionsBody_OmitsOverrideFieldsWhenThereAreNone(t *testing.T) {
	raw, err := json.Marshal(toEffectivePermissionsBody(&authv1.GetEffectivePermissionsResponse{
		Capabilities: []*authv1.EffectiveCapability{{Key: "work:task:read", Scope: "own", Sources: []string{"r"}}},
	}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if strings.Contains(string(raw), "hasOverrides") || strings.Contains(string(raw), "deniedByOverride") {
		t.Errorf("override fields must be omitted when empty, got: %s", raw)
	}
}

// --- HandleListRoles ---

func TestHandleListRoles_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListRoles)
}

// TestToRoleBody_WireShape pins the null sentinel: a system preset carries an
// empty tenant_id and preset_id on the wire (proto3 has no string null), and
// the gateway must render those as JSON null, not "" — the frontend's
// isSystem / basedOn logic keys off null, not an empty string.
func TestToRoleBody_WireShape(t *testing.T) {
	raw, err := json.Marshal(toRoleBody(&authv1.Role{
		Id:              "11111111-1111-1111-1111-111111111111",
		Name:            "admin",
		Description:     "Full system access",
		TenantId:        "",
		PresetId:        "",
		IsSystem:        true,
		Color:           "hsl(0 72% 51%)",
		MemberCount:     3,
		CapabilityCount: 282,
	}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	want := `{"id":"11111111-1111-1111-1111-111111111111","name":"admin","description":"Full system access",` +
		`"tenantId":null,"basedOn":null,"isSystem":true,"color":"hsl(0 72% 51%)","memberCount":3,"capabilityCount":282}`
	if string(raw) != want {
		t.Errorf("wire shape drifted\n got: %s\nwant: %s", raw, want)
	}
}

// TestToRoleBody_CustomRoleCarriesTenantAndBasedOn is the counterpart: a
// tenant-owned custom role has both ids populated on the wire.
func TestToRoleBody_CustomRoleCarriesTenantAndBasedOn(t *testing.T) {
	raw, err := json.Marshal(toRoleBody(&authv1.Role{
		Id:       "22222222-2222-2222-2222-222222222222",
		Name:     "Buchhaltung",
		TenantId: "33333333-3333-3333-3333-333333333333",
		PresetId: "11111111-1111-1111-1111-111111111111",
		IsSystem: false,
	}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	want := `{"id":"22222222-2222-2222-2222-222222222222","name":"Buchhaltung","description":"",` +
		`"tenantId":"33333333-3333-3333-3333-333333333333","basedOn":"11111111-1111-1111-1111-111111111111",` +
		`"isSystem":false,"color":"","memberCount":0,"capabilityCount":0}`
	if string(raw) != want {
		t.Errorf("wire shape drifted\n got: %s\nwant: %s", raw, want)
	}
}

// TestRolesBody_EmptyIsContainerNotNull guards a tenant with zero visible
// roles (should not happen once presets seed, but the frontend iterates roles
// unconditionally either way).
func TestRolesBody_EmptyIsContainerNotNull(t *testing.T) {
	body := rolesBody{Roles: []roleBody{}}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	want := `{"roles":[]}`
	if string(raw) != want {
		t.Errorf("empty roles = %s, want %s", raw, want)
	}
}

// --- HandleCreateRole ---

func TestHandleCreateRole_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleCreateRole)
}

// TestRoleResponseBody_WrapsTheEntity pins the single-entity wrapper: the
// frontend's createRole reads `resp.role`, so a bare role would arrive as
// undefined and the builder would render an empty card after a successful
// create.
func TestRoleResponseBody_WrapsTheEntity(t *testing.T) {
	raw, err := json.Marshal(roleResponseBody{Role: toRoleBody(&authv1.Role{
		Id:              "44444444-4444-4444-4444-444444444444",
		Name:            "Buchhaltung",
		TenantId:        "33333333-3333-3333-3333-333333333333",
		PresetId:        "11111111-1111-1111-1111-111111111111",
		Color:           "hsl(217 91% 60%)",
		CapabilityCount: 11,
	})})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	want := `{"role":{"id":"44444444-4444-4444-4444-444444444444","name":"Buchhaltung","description":"",` +
		`"tenantId":"33333333-3333-3333-3333-333333333333","basedOn":"11111111-1111-1111-1111-111111111111",` +
		`"isSystem":false,"color":"hsl(217 91% 60%)","memberCount":0,"capabilityCount":11}}`
	if string(raw) != want {
		t.Errorf("wire shape drifted\n got: %s\nwant: %s", raw, want)
	}
}

// --- HandleUpdateRole / HandleDeleteRole ---

func TestHandleUpdateRole_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleUpdateRole)
}

func TestHandleDeleteRole_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleDeleteRole)
}

// --- HandleGetRolePermissions / HandleSetRolePermissions ---

func TestHandleGetRolePermissions_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleGetRolePermissions)
}

func TestHandleSetRolePermissions_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleSetRolePermissions)
}

// TestToRoleGrantsBody_EmptyIsContainerNotNull guards a role with zero
// grants: the frontend's fetchRolePermissions does `resp?.grants ?? {}`, but
// Object.entries on a JSON null still throws before that fallback helps, so
// the map must marshal as {} on the wire, not null.
func TestToRoleGrantsBody_EmptyIsContainerNotNull(t *testing.T) {
	raw, err := json.Marshal(rolePermissionsResponseBody{
		RoleID: "11111111-1111-1111-1111-111111111111",
		Grants: toRoleGrantsBody(nil),
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	want := `{"roleId":"11111111-1111-1111-1111-111111111111","grants":{}}`
	if string(raw) != want {
		t.Errorf("empty grants = %s, want %s", raw, want)
	}
}

// --- HandleAssignRole / HandleRemoveRole ---

func TestHandleAssignRole_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleAssignRole)
}

func TestHandleRemoveRole_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleRemoveRole)
}

// TestHandleRemoveRole_InvalidRoleUUID covers the parameter the route gained
// in wave 1b: the role now travels in the path, so a malformed one has to be
// caught here rather than reaching the service as an empty id.
func TestHandleRemoveRole_InvalidRoleUUID(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/users/550e8400-e29b-41d4-a716-446655440000/roles/bad-id", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	req = withChiURLParam(req, "roleId", "not-a-uuid")
	routes.HandleRemoveRole(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid roleId")
}

// TestToUserRolesBody_EmptyIsContainerNotNull guards the answer for an account
// left without any role: removeUserRole does `resp?.roles ?? []`, but the
// frontend also feeds the value straight into .map/.includes on the summary
// view, where a JSON null throws before that fallback can help.
func TestToUserRolesBody_EmptyIsContainerNotNull(t *testing.T) {
	raw, err := json.Marshal(toUserRolesBody(nil))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	want := `{"roles":[]}`
	if string(raw) != want {
		t.Errorf("empty roles = %s, want %s", raw, want)
	}
}

// TestToUserRolesBody_WireShape pins the plain id list AssignRoleInput's
// counterpart carries — role ids, not names: the frontend matches them against
// the ids it got from GET /admin/roles.
func TestToUserRolesBody_WireShape(t *testing.T) {
	raw, err := json.Marshal(toUserRolesBody([]string{
		"11111111-1111-1111-1111-111111111111",
		"22222222-2222-2222-2222-222222222222",
	}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	want := `{"roles":["11111111-1111-1111-1111-111111111111","22222222-2222-2222-2222-222222222222"]}`
	if string(raw) != want {
		t.Errorf("wire shape drifted\n got: %s\nwant: %s", raw, want)
	}
}

// TestToRoleGrantsBody_WireShape pins the map-keyed-by-capability-key shape:
// RoleGrants in rbac-types.ts is Record<string, {scope}>, the proto's
// repeated RoleGrant is the wire form the gateway must fold into it.
func TestToRoleGrantsBody_WireShape(t *testing.T) {
	raw, err := json.Marshal(rolePermissionsResponseBody{
		RoleID: "11111111-1111-1111-1111-111111111111",
		Grants: toRoleGrantsBody([]*authv1.RoleGrant{
			{Key: "crm:contact:read", Scope: "all"},
			{Key: "work:task:edit", Scope: "team"},
		}),
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	want := `{"roleId":"11111111-1111-1111-1111-111111111111",` +
		`"grants":{"crm:contact:read":{"scope":"all"},"work:task:edit":{"scope":"team"}}}`
	if string(raw) != want {
		t.Errorf("wire shape drifted\n got: %s\nwant: %s", raw, want)
	}
}

// --- HandleGetUserOverrides / HandleSetUserOverrides / HandleClearUserOverrides ---

func TestHandleGetUserOverrides_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleGetUserOverrides)
}

func TestHandleSetUserOverrides_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleSetUserOverrides)
}

func TestHandleClearUserOverrides_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleClearUserOverrides)
}

// TestToOverridesBody_EmptyIsContainerNotNull: an account with no deviation is
// the normal case, and fetchUserOverrides does `resp?.overrides ?? {}` — but
// Object.entries on a JSON null throws before that fallback ever helps, so the
// map has to marshal as {}.
func TestToOverridesBody_EmptyIsContainerNotNull(t *testing.T) {
	raw, err := json.Marshal(userOverridesResponseBody{
		UserID:    "22222222-2222-2222-2222-222222222222",
		Overrides: toOverridesBody(nil),
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	want := `{"userId":"22222222-2222-2222-2222-222222222222","overrides":{}}`
	if string(raw) != want {
		t.Errorf("wire shape drifted\n got: %s\nwant: %s", raw, want)
	}
}

// TestToOverridesBody_WireShape pins the shape against UserOverridesResponse
// in rbac-types.ts: the capability key is the map key, never a field, and both
// mode and scope travel even for a deny (the frontend type carries them both).
func TestToOverridesBody_WireShape(t *testing.T) {
	raw, err := json.Marshal(userOverridesResponseBody{
		UserID: "22222222-2222-2222-2222-222222222222",
		Overrides: toOverridesBody([]*authv1.CapabilityOverride{
			{Key: "rapporte:report:create", Mode: "deny", Scope: "all"},
			{Key: "work:project:edit", Mode: "allow", Scope: "team"},
		}),
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	want := `{"userId":"22222222-2222-2222-2222-222222222222","overrides":{` +
		`"rapporte:report:create":{"mode":"deny","scope":"all"},` +
		`"work:project:edit":{"mode":"allow","scope":"team"}}}`
	if string(raw) != want {
		t.Errorf("wire shape drifted\n got: %s\nwant: %s", raw, want)
	}
}
