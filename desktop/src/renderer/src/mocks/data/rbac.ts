/**
 * RBAC seed (R-1) — the 7 system preset roles, their curated capability
 * grants, and the demo account→roles assignment.
 *
 * Grants follow `.planning/rbac-block/CAPABILITY-KATALOG.md`: level 1
 * (`<module>:module:view`) is seeded for ALL modules; level 2/3 fine
 * capabilities are seeded for the core modules (work, documents, crm,
 * finance, team, wiki, settings/admin/security). Remaining modules get their
 * fine grants curated per R-3 batch — until then they carry visibility only.
 *
 * Deliberately NO wildcard grant for admin: every capability is explicit, so
 * new modules stay default-deny for every role until seeded here (and later
 * in Luke's permission seed migrations).
 */
import type { CapabilityGrant, CapabilityScope, Role } from '@/api/rbac-types'
import { SCOPE_ORDER } from '@/api/rbac-types'
import type { RoleId } from '@/config/roles'
import { MODULE_KEYS, moduleViewKey, type ModuleKey } from '@/config/capabilities'
import type { User } from '@/stores/auth'
import { CURRENT_USER, IDS } from './shared-ids'

// ── Grant builders ──────────────────────────────────────────────────────────

/** capability key → scope of the grant. */
type GrantSpec = Record<string, CapabilityScope>

const keys = (list: string[], scope: CapabilityScope = 'all'): GrantSpec =>
  Object.fromEntries(list.map((k) => [k, scope]))

const view = (modules: readonly ModuleKey[]): GrantSpec =>
  keys(modules.map((m) => moduleViewKey(m)))

/** Later specs override earlier ones (used for scope overrides per role). */
const merge = (...specs: GrantSpec[]): GrantSpec => Object.assign({}, ...specs)

// ── Module groups (level 1) ─────────────────────────────────────────────────

const STANDARD_MODULES: ModuleKey[] = [
  'dashboard', 'work', 'kommunikation', 'crm', 'kalender', 'zeiterfassung',
  'documents', 'wiki', 'mail', 'video', 'team', 'notifications', 'settings',
]

const INDUSTRY_MODULES: ModuleKey[] = [
  'inventar', 'schichten', 'einkauf', 'helpdesk', 'fuhrpark', 'produktion',
  'berichte', 'vertraege', 'formulare', 'vermietung', 'rapporte', 'dialer',
  'automatisierung',
]

// ── Core-module fine capabilities (catalogue keys) ──────────────────────────

const WORK_ALL = [
  'work:task:read', 'work:task:create', 'work:task:edit', 'work:task:delete',
  'work:task:be_assigned', 'work:task:comment',
  'work:project:read', 'work:project:create', 'work:project:edit', 'work:project:delete',
  'work:project:manage_members', 'work:time:log', 'work:board:export',
]

const DOCUMENTS_ALL = [
  'documents:file:read', 'documents:file:download', 'documents:file:upload',
  'documents:file:edit', 'documents:file:delete',
  'documents:share:manage', 'documents:share_link:create',
  'documents:version:restore', 'documents:template:manage',
]

const CRM_ALL = [
  'crm:contact:read', 'crm:contact:create', 'crm:contact:edit', 'crm:contact:delete',
  'crm:contact:export', 'crm:deal:read', 'crm:deal:create', 'crm:deal:edit',
  'crm:pipeline:manage', 'crm:import:run', 'crm:advisory:read', 'crm:advisory:write',
  'crm:segment:override',
]

const FINANCE_ALL = [
  'finance:invoice:read', 'finance:invoice:create', 'finance:invoice:edit',
  'finance:invoice:delete', 'finance:invoice:send', 'finance:dunning:run',
  'finance:quote:read', 'finance:quote:create', 'finance:quote:send',
  'finance:amounts:view', 'finance:export:run', 'finance:incoming:review',
  'finance:incoming:book', 'finance:settings:manage',
]

const TEAM_HR_ALL = [
  'team:employee:read', 'team:employee:create', 'team:employee:edit', 'team:employee:deactivate',
  'team:data_personal:view', 'team:data_personal:edit',
  'team:data_job:view', 'team:data_job:edit',
  'team:salary:view', 'team:salary:edit',
  'team:documents:view', 'team:documents:edit',
  'team:absence:read', 'team:absence:approve',
  'team:role:assign', 'team:training:manage',
  'team:payroll:view', 'team:payroll:run',
  'team:corrections:manage', 'team:onboarding:manage',
]

const WIKI_ALL = [
  'wiki:article:read', 'wiki:article:create', 'wiki:article:edit', 'wiki:article:delete',
  'wiki:article:publish', 'wiki:share_token:create', 'wiki:template:manage',
]

const ADMIN_SECURITY_ALL = [
  'settings:personal:manage', 'settings:tenant:manage',
  'admin:module:view', 'admin:user:read', 'admin:user:invite', 'admin:user:deactivate',
  'admin:role:read', 'admin:role:create', 'admin:role:edit', 'admin:role:delete', 'admin:role:assign',
  'admin:license:manage', 'admin:branding:manage', 'admin:integrations:manage',
  'admin:company:manage', 'admin:modules:manage', 'admin:it:manage', 'admin:ai:manage',
  'admin:impersonate:run', 'mail:settings:manage',
  'security:module:view', 'security:audit:read', 'security:policy:manage', 'security:gdpr:execute',
]

// ── The 7 system presets ────────────────────────────────────────────────────

interface RoleDef {
  /** Technical English name (mirrors what Luke's `roles.name` column will hold). */
  name: string
  description: string
  color: string
  grants: GrantSpec
}

export const ROLE_DEFS: Record<RoleId, RoleDef> = {
  // Vollzugriff — every capability, explicitly (no wildcard).
  admin: {
    name: 'Full Access',
    description: 'Owner/administrator with every capability',
    color: 'hsl(0 72% 51%)',
    grants: merge(
      view(MODULE_KEYS),
      keys(WORK_ALL), keys(DOCUMENTS_ALL), keys(CRM_ALL), keys(FINANCE_ALL),
      keys(TEAM_HR_ALL), keys(WIKI_ALL), keys(ADMIN_SECURITY_ALL),
    ),
  },
  // IT-Admin — full technical control, NO HR data categories, no finance
  // (the market-gap role: technical admin without salary access).
  it_admin: {
    name: 'IT Administrator',
    description: 'Technical administration without HR/salary data',
    color: 'hsl(25 95% 53%)',
    grants: merge(
      view(MODULE_KEYS.filter((m) => m !== 'finance')),
      keys([
        'work:task:read', 'work:project:read', 'documents:file:read',
        'crm:contact:read', 'crm:deal:read', 'wiki:article:read',
        'team:employee:read',
        'settings:personal:manage', 'settings:tenant:manage',
        'admin:module:view', 'admin:user:read', 'admin:user:invite', 'admin:user:deactivate',
        'admin:role:read', 'admin:role:create', 'admin:role:edit', 'admin:role:delete', 'admin:role:assign',
        'admin:integrations:manage', 'admin:modules:manage', 'admin:it:manage', 'admin:ai:manage',
        'mail:settings:manage',
        'security:module:view', 'security:audit:read', 'security:policy:manage',
      ]),
    ),
  },
  // HR-Admin — people management incl. protected data categories; assigns
  // roles but never creates/edits them (assign ≠ manage).
  hr_admin: {
    name: 'HR Administrator',
    description: 'People management incl. protected HR data; assigns roles',
    color: 'hsl(270 76% 55%)',
    grants: merge(
      view([...STANDARD_MODULES, 'admin']),
      keys(TEAM_HR_ALL),
      keys(['work:task:read', 'work:project:read', 'documents:file:read', 'documents:file:download',
        'documents:file:upload', 'wiki:article:read', 'wiki:article:create',
        'crm:contact:read']),
      keys(['work:task:create', 'work:task:edit', 'work:task:be_assigned', 'work:task:comment', 'work:time:log'], 'own'),
      keys(['settings:personal:manage',
        'admin:module:view', 'admin:user:read', 'admin:user:invite', 'admin:user:deactivate',
        'admin:role:read', 'admin:role:assign']),
    ),
  },
  // Teamleiter — leads people and work, approves, no system administration.
  manager: {
    name: 'Team Lead',
    description: 'Leads projects and people, approves, no system administration',
    color: 'hsl(217 91% 60%)',
    grants: merge(
      view([...STANDARD_MODULES, ...INDUSTRY_MODULES]),
      keys(['work:task:read', 'work:task:create', 'work:task:be_assigned', 'work:task:comment',
        'work:project:read', 'work:project:create', 'work:project:edit', 'work:project:manage_members',
        'work:time:log', 'work:board:export',
        'documents:file:read', 'documents:file:download', 'documents:file:upload', 'documents:share:manage',
        'crm:contact:read', 'crm:deal:read', 'crm:deal:create',
        'wiki:article:read', 'wiki:article:create', 'wiki:article:publish',
        'team:absence:read', 'team:absence:approve', 'team:corrections:manage',
        'team:training:manage', 'team:onboarding:manage',
        'settings:personal:manage']),
      keys(['work:task:edit', 'work:task:delete', 'documents:file:edit',
        'crm:contact:create', 'crm:contact:edit', 'crm:deal:edit', 'team:employee:read'], 'team'),
      keys(['wiki:article:edit'], 'own'),
    ),
  },
  // Mitarbeiter — day-to-day work, own-scope editing, no management.
  member: {
    name: 'Employee',
    description: 'Day-to-day work with own-scope editing',
    color: 'hsl(142 71% 45%)',
    grants: merge(
      view([...STANDARD_MODULES, ...INDUSTRY_MODULES]),
      keys(['work:task:read', 'work:task:create', 'work:task:be_assigned', 'work:task:comment',
        'work:project:read',
        'documents:file:read', 'documents:file:download', 'documents:file:upload',
        'crm:contact:read', 'crm:deal:read',
        'wiki:article:read', 'wiki:article:create',
        'team:employee:read', 'team:absence:read',
        'settings:personal:manage']),
      keys(['work:task:edit', 'work:time:log', 'documents:file:edit', 'wiki:article:edit'], 'own'),
      keys(['crm:contact:create', 'crm:contact:edit'], 'team'),
    ),
  },
  // Nur-Lesen — audit/tax-advisor style visibility, zero mutations, no download.
  readonly: {
    name: 'Read Only',
    description: 'Sees everything relevant, changes nothing',
    color: 'hsl(215 16% 47%)',
    grants: merge(
      view([...STANDARD_MODULES, ...INDUSTRY_MODULES, 'finance']),
      keys(['work:task:read', 'work:project:read', 'documents:file:read',
        'crm:contact:read', 'crm:deal:read', 'wiki:article:read',
        'team:employee:read', 'team:absence:read',
        'finance:invoice:read', 'finance:quote:read', 'finance:amounts:view',
        'settings:personal:manage']),
    ),
  },
  // Aushilfe/Extern — Dariens Referenzfall: sees assigned work and documents,
  // can tick off own tasks and comment, nothing else. No download.
  extern: {
    name: 'Temp / External',
    description: 'Assigned tasks and read-only documents, nothing else',
    color: 'hsl(180 45% 42%)',
    grants: merge(
      view(['dashboard', 'work', 'documents', 'notifications', 'settings']),
      keys(['work:task:read', 'work:project:read', 'documents:file:read'], 'team'),
      keys(['work:task:be_assigned', 'work:task:comment'], 'own'),
      keys(['settings:personal:manage']),
    ),
  },
}

export const PRESET_ROLE_IDS = Object.keys(ROLE_DEFS) as RoleId[]

// ── Demo account → roles assignment (mirrors Luke's user_roles n:m) ─────────

/**
 * Laura carries TWO roles (manager + hr_admin) — the multi-role union demo
 * that feeds the "Effektive Rechte" provenance view. AdminUser.role (singular)
 * still shows her primary role until A-1 moves to multi-role in R-2.
 */
export const USER_ROLE_ASSIGNMENTS: Record<string, RoleId[]> = {
  [CURRENT_USER.id]: ['admin'],
  [IDS.users.thomas]: ['it_admin'],
  [IDS.users.nina]: ['hr_admin'],
  [IDS.users.sarah]: ['manager'],
  [IDS.users.laura]: ['manager', 'hr_admin'],
  [IDS.users.markus]: ['member'],
  [IDS.users.felix]: ['member'],
  [IDS.users.julia]: ['member'],
  [IDS.users.jan]: ['member'],
  [IDS.users.lena]: ['member'],
  [IDS.users.david]: ['member'],
  [IDS.users.elena]: ['readonly'],
  [IDS.users.max]: ['extern'],
}

/** Roles for an account; unknown accounts fall back to least-privilege member. */
export function rolesForUser(userId: string): RoleId[] {
  return USER_ROLE_ASSIGNMENTS[userId] ?? ['member']
}

// ── Union resolution (server-side logic, mirrored by Luke later) ────────────

/** Resolve the effective capability map for a set of roles (widest scope wins, provenance kept). */
export function resolveCapabilities(roleIds: RoleId[]): Record<string, CapabilityGrant> {
  const result: Record<string, CapabilityGrant> = {}
  for (const roleId of roleIds) {
    const def = ROLE_DEFS[roleId]
    if (!def) continue
    for (const [key, scope] of Object.entries(def.grants)) {
      const existing = result[key]
      if (!existing) {
        result[key] = { scope, sources: [roleId] }
      } else {
        if (SCOPE_ORDER[scope] > SCOPE_ORDER[existing.scope]) existing.scope = scope
        if (!existing.sources.includes(roleId)) existing.sources.push(roleId)
      }
    }
  }
  return result
}

export function seedRoles(): Role[] {
  const memberCounts: Record<string, number> = {}
  for (const roleIds of Object.values(USER_ROLE_ASSIGNMENTS)) {
    for (const r of roleIds) memberCounts[r] = (memberCounts[r] ?? 0) + 1
  }
  return PRESET_ROLE_IDS.map((id) => ({
    id,
    name: ROLE_DEFS[id].name,
    description: ROLE_DEFS[id].description,
    tenantId: null,
    basedOn: null,
    isSystem: true,
    color: ROLE_DEFS[id].color,
    memberCount: memberCounts[id] ?? 0,
    capabilityCount: Object.keys(ROLE_DEFS[id].grants).length,
  }))
}

// ── Demo session (which account "is" logged in for MSW) ─────────────────────

/**
 * The MSW layer cannot read the demo JWT, so the profile switcher records the
 * active demo account here; `GET /auth/me/permissions` resolves against it.
 * Default: the canonical demo identity.
 */
let demoSessionUserId: string = CURRENT_USER.id

export function getDemoSessionUserId(): string {
  return demoSessionUserId
}

export function setDemoSessionUserId(userId: string): void {
  demoSessionUserId = userId
}

// ── Demo profiles (dev profile switcher) ────────────────────────────────────

export interface DemoProfile {
  /** Primary role id (panel identity + accent). */
  roleId: RoleId
  user: User
  initials: string
}

const demoUser = (id: string, firstName: string, lastName: string, email: string): User => ({
  id,
  firstName,
  lastName,
  email,
  roles: rolesForUser(id),
  avatarUrl: null,
})

/**
 * One switchable demo identity per preset role, plus Laura as the explicit
 * multi-role combo. Identities reuse the shared roster (IDS.users) so the
 * admin user list, team roster and switcher reference the same people.
 */
export const DEMO_PROFILES: DemoProfile[] = [
  { roleId: 'admin', user: demoUser(CURRENT_USER.id, CURRENT_USER.firstName, CURRENT_USER.lastName, CURRENT_USER.email), initials: 'SV' },
  { roleId: 'it_admin', user: demoUser(IDS.users.thomas, 'Thomas', 'Keller', 'thomas.keller@techvision.de'), initials: 'TK' },
  { roleId: 'hr_admin', user: demoUser(IDS.users.nina, 'Nina', 'Fischer', 'nina.fischer@techvision.de'), initials: 'NF' },
  { roleId: 'manager', user: demoUser(IDS.users.sarah, 'Sarah', 'Müller', 'sarah.mueller@techvision.de'), initials: 'SM' },
  { roleId: 'member', user: demoUser(IDS.users.markus, 'Markus', 'Weber', 'markus.weber@techvision.de'), initials: 'MW' },
  { roleId: 'readonly', user: demoUser(IDS.users.elena, 'Elena', 'Richter', 'elena.richter@extern.de'), initials: 'ER' },
  { roleId: 'extern', user: demoUser(IDS.users.max, 'Max', 'Steiner', 'max.steiner@extern.de'), initials: 'MS' },
  // Multi-role combo demo (manager + hr_admin union with provenance)
  { roleId: 'manager', user: demoUser(IDS.users.laura, 'Laura', 'Neumann', 'laura.neumann@techvision.de'), initials: 'LN' },
]
