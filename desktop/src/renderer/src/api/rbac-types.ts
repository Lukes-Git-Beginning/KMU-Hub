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

/** Provenance sentinel for a grant that a per-user override contributed (R-6). */
export const OVERRIDE_SOURCE = 'override' as const

/**
 * One resolved grant of the effective permission set.
 * `sources` carries the role ids the grant derives from — the "Effektive
 * Rechte" provenance no competitor ships (KONZEPT §3, Multi-Rollen-Union).
 * An `allow` override adds the `OVERRIDE_SOURCE` sentinel to `sources`.
 */
export interface CapabilityGrant {
  scope: CapabilityScope
  /** Role ids (Role.id) this grant originates from — union provenance. */
  sources: string[]
}

/**
 * A key the role union WOULD have granted but a per-user `deny` override
 * removed (R-6). Carried alongside the resolved capabilities so the effective
 * view can render it struck-through with "persönlich entzogen" provenance
 * instead of silently dropping it (the market gap — see R6-RECHERCHE §1).
 */
export interface DeniedByOverride {
  /** Full capability key (`resource:action`). */
  key: string
  /** Scope the role union would have granted (for the struck-through label). */
  roleScope: CapabilityScope
  /** Role ids that granted it before the override — kept for provenance. */
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
  /** Whether this account carries any per-user override (drives the "Angepasst" badge). R-6. */
  hasOverrides?: boolean
  /** Keys the roles granted but a deny override removed — for the struck-through effective view. R-6. */
  deniedByOverride?: DeniedByOverride[]
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

// ── R-6 per-user permission overrides ───────────────────────────────────────

/** An override either grants a key (with a scope) or revokes it entirely. */
export type OverrideMode = 'allow' | 'deny'

/**
 * A single per-user override. `deny` removes the key from the effective set
 * regardless of the roles; `allow` sets/raises it to `scope`. The override
 * wins per key over the entire role union (R6-Briefing §0).
 */
export interface CapabilityOverride {
  mode: OverrideMode
  /** Target scope for `allow`; ignored (but stored) for `deny`. */
  scope: CapabilityScope
}

/** A user's override map: capability key → override. Missing key = follows roles live. */
export type UserOverrides = Record<string, CapabilityOverride>

/** `PUT /admin/users/{id}/overrides` body — replaces the whole map. */
export interface UpdateUserOverridesInput {
  overrides: UserOverrides
}

export interface UserOverridesResponse {
  userId: string
  overrides: UserOverrides
}
