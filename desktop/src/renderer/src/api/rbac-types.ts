/**
 * RBAC — app-owned API contract types (R-1 Fundament).
 *
 * The FE consumes these shapes; the MSW mock conforms today, the real backend
 * (Luke's track 🔒, backend-gaps §RBAC) conforms tomorrow. Cut so Luke's
 * existing schema (roles / permissions(resource, action) / role_permissions /
 * user_roles, migration 000002) can serve it 1:1 once `roles.tenant_id`,
 * `based_on` and `GET /me/permissions` land.
 *
 * Capability key format is the backend's `resource:action` composition, where
 * the resource itself may be two-part: `work:task:edit` ⇒ resource `work:task`,
 * action `edit`. Level-1 visibility uses the `<module>:module:view` convention.
 * The curated key catalogue lives in `.planning/rbac-block/CAPABILITY-KATALOG.md`.
 */

/** Data scope of a granted capability (union resolves to the widest). */
export type CapabilityScope = 'own' | 'team' | 'all'

/** Scope width order for union resolution: all > team > own. */
export const SCOPE_ORDER: Record<CapabilityScope, number> = { own: 0, team: 1, all: 2 }

/**
 * One resolved grant of the effective permission set.
 * `sources` carries the role ids the grant derives from — the "Effektive
 * Rechte" provenance no competitor ships (KONZEPT §3, Multi-Rollen-Union).
 */
export interface CapabilityGrant {
  scope: CapabilityScope
  /** Role ids (Role.id) this grant originates from — union provenance. */
  sources: string[]
}

/**
 * A role. `tenantId === null` ⇒ immutable system preset (Zentria-maintained);
 * set ⇒ tenant custom role (R-2 editor). `basedOn` points at the preset a
 * custom role was cloned from.
 */
export interface Role {
  id: string
  /** Technical name (English, unique per tenant). System presets are mapped to i18n labels by id in the FE. */
  name: string
  description: string
  tenantId: string | null
  basedOn: string | null
  isSystem: boolean
  /** HSL accent used for role dots/avatars across admin surfaces. */
  color: string
  /** Number of accounts currently holding this role. */
  memberCount: number
  /** Number of capability grants the role carries (summary for list views). */
  capabilityCount: number
}

/** Effective permissions of the authenticated account (multi-role union, resolved server-side). */
export interface EffectivePermissions {
  /** Roles the account holds (assignment order). */
  roles: Array<Pick<Role, 'id' | 'name' | 'isSystem' | 'color'>>
  /** Flat map: full capability key (`resource:action`) → grant. Missing key = denied (default-deny). */
  capabilities: Record<string, CapabilityGrant>
}

export interface EffectivePermissionsResponse {
  permissions: EffectivePermissions
}

export interface RolesResponse {
  roles: Role[]
}

// ── R-2 role builder (CRUD + per-role grants + assignment) ──────────────────

/**
 * The editable grant map of a single role: capability key → scope. Missing
 * key = not granted. This is what `PUT /admin/roles/{id}/permissions` writes —
 * Luke's `role_permissions` rows map 1:1 (scope lands as a new column there,
 * backend-gaps §RBAC).
 */
export type RoleGrants = Record<string, { scope: CapabilityScope }>

export interface RolePermissionsResponse {
  roleId: string
  grants: RoleGrants
}

/** Create = always a clone: grants start as a copy of `basedOn`'s grants. */
export interface CreateRoleInput {
  name: string
  description: string
  color: string
  /** Role id (preset or custom) the new role clones from. */
  basedOn: string
}

export interface UpdateRoleInput {
  name?: string
  description?: string
  color?: string
}

export interface UpdateRolePermissionsInput {
  grants: RoleGrants
}

export interface RoleResponse {
  role: Role
}

/** `POST /users/{id}/roles` body (route exists in Luke's route_auth.go). */
export interface AssignRoleInput {
  roleId: string
}

/** Role ids an account holds after an assignment mutation. */
export interface UserRolesResponse {
  roles: string[]
}
