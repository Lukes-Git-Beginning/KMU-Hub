package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
