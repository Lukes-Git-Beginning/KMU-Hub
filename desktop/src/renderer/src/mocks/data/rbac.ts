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
import { catalogCapabilityKeys } from '@/config/capability-catalog'
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

// ── Core-module fine capabilities (from the catalogue — one source, no drift) ─

const WORK_ALL = catalogCapabilityKeys('work')
const DOCUMENTS_ALL = catalogCapabilityKeys('documents')
const CRM_ALL = catalogCapabilityKeys('crm')
const FINANCE_ALL = catalogCapabilityKeys('finance')
const TEAM_HR_ALL = catalogCapabilityKeys('team')
const WIKI_ALL = catalogCapabilityKeys('wiki')

// R-3 batch 3: industry modules (inventar/einkauf/produktion/vertraege/helpdesk)
const INVENTAR_ALL = catalogCapabilityKeys('inventar')
const EINKAUF_ALL = catalogCapabilityKeys('einkauf')
const PRODUKTION_ALL = catalogCapabilityKeys('produktion')
const VERTRAEGE_ALL = catalogCapabilityKeys('vertraege')
const HELPDESK_ALL = catalogCapabilityKeys('helpdesk')

/** Read-level keys of the batch-3 modules (tab visibility for read-style roles). */
const INDUSTRY_B3_READS = [
  'inventar:item:read', 'inventar:location:read', 'inventar:movement:read', 'inventar:inventur:read',
  'einkauf:po:read', 'einkauf:supplier:read', 'einkauf:catalog:read', 'einkauf:contract:read',
  'produktion:order:read', 'produktion:bom:read', 'produktion:quality:read', 'produktion:machine:read',
  'vertraege:contract:read',
]

// R-3 batch 4: industry modules (schichten/fuhrpark/vermietung/rapporte/dialer)
const SCHICHTEN_ALL = catalogCapabilityKeys('schichten')
const FUHRPARK_ALL = catalogCapabilityKeys('fuhrpark')
const VERMIETUNG_ALL = catalogCapabilityKeys('vermietung')
const RAPPORTE_ALL = catalogCapabilityKeys('rapporte')
const DIALER_ALL = catalogCapabilityKeys('dialer')

/**
 * Read-level keys of the batch-4 modules. `fuhrpark:gps:read` deliberately
 * stays out — GPS routes are movement profiles (personal data), granted only
 * to admin + manager, not to insight-style roles.
 */
const INDUSTRY_B4_READS = [
  'schichten:shift:read', 'schichten:template:read', 'schichten:swap:read',
  'fuhrpark:vehicle:read', 'fuhrpark:service:read', 'fuhrpark:fuel:read', 'fuhrpark:trip:read',
  'vermietung:object:read', 'vermietung:rental:read',
  'rapporte:report:read', 'rapporte:measurement:read', 'rapporte:template:read',
  'dialer:campaigns:read', 'dialer:calls:read',
]

// R-3 batch 5 (close-out): berichte/formulare/automatisierung + the
// standard-module mini catalogues (management-only keys, daily use stays free).
const BERICHTE_ALL = catalogCapabilityKeys('berichte')
const FORMULARE_ALL = catalogCapabilityKeys('formulare')
const AUTOMATISIERUNG_ALL = catalogCapabilityKeys('automatisierung')
const KOMMUNIKATION_MGMT = catalogCapabilityKeys('kommunikation')
const KALENDER_MGMT = catalogCapabilityKeys('kalender')
const ZEITERFASSUNG_MGMT = catalogCapabilityKeys('zeiterfassung')
const INFRASTRUCTURE_MGMT = catalogCapabilityKeys('infrastructure')

/**
 * Read-level keys of the batch-5 modules. `berichte:datev:read` deliberately
 * stays out — DATEV/BWA/SuSa is finance data; it goes only to admin and to
 * readonly (the tax-advisor case), NOT to it_admin/manager (both deliberately
 * have no finance access). Mirrors the fuhrpark:gps:read privacy exception.
 */
const INDUSTRY_B5_READS = [
  'berichte:reports:read',
  'formulare:schemas:read', 'formulare:submissions:read',
  'automatisierung:automations:read', 'automatisierung:executions:read',
]

const ADMIN_SECURITY_ALL = [
  ...catalogCapabilityKeys('settings'),
  ...catalogCapabilityKeys('admin'),
  ...catalogCapabilityKeys('security'),
  ...catalogCapabilityKeys('mail'),
  moduleViewKey('admin'),
  moduleViewKey('security'),
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
      keys(INVENTAR_ALL), keys(EINKAUF_ALL), keys(PRODUKTION_ALL),
      keys(VERTRAEGE_ALL), keys(HELPDESK_ALL),
      keys(SCHICHTEN_ALL), keys(FUHRPARK_ALL), keys(VERMIETUNG_ALL),
      keys(RAPPORTE_ALL), keys(DIALER_ALL),
      keys(BERICHTE_ALL), keys(FORMULARE_ALL), keys(AUTOMATISIERUNG_ALL),
      keys(KOMMUNIKATION_MGMT), keys(KALENDER_MGMT),
      keys(ZEITERFASSUNG_MGMT), keys(INFRASTRUCTURE_MGMT),
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
      // Industry modules: read-level insight; helpdesk is the IT domain → full.
      keys(INDUSTRY_B3_READS),
      keys(HELPDESK_ALL),
      keys(INDUSTRY_B4_READS),
      // Batch 5: automatisierung + infrastructure are IT domains → full
      // (mirrors it_admin↔helpdesk); webhooks are technical integrations.
      keys(INDUSTRY_B5_READS),
      keys(AUTOMATISIERUNG_ALL), keys(INFRASTRUCTURE_MGMT),
      keys(['kommunikation:webhook:manage']),
    ),
  },
  // HR-Admin — people management incl. protected data categories; assigns
  // roles but never creates/edits them (assign ≠ manage).
  hr_admin: {
    name: 'HR Administrator',
    description: 'People management incl. protected HR data; assigns roles',
    color: 'hsl(270 76% 55%)',
    grants: merge(
      // schichten is the HR domain (workforce scheduling à la Personio) —
      // mirrors the it_admin↔helpdesk pattern from batch 3.
      view([...STANDARD_MODULES, 'admin', 'schichten']),
      keys(TEAM_HR_ALL),
      keys(SCHICHTEN_ALL),
      // Batch 5: zeiterfassung management (week/correction approvals, team
      // view, payroll exports) is the HR domain — full, like schichten.
      keys(ZEITERFASSUNG_MGMT),
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
        'wiki:article:read', 'wiki:article:create', 'wiki:article:publish', 'wiki:category:manage',
        'team:absence:read', 'team:absence:approve', 'team:corrections:manage',
        'team:training:manage', 'team:onboarding:manage',
        'settings:personal:manage']),
      keys(['work:task:edit', 'work:task:delete', 'documents:file:edit',
        'crm:contact:create', 'crm:contact:edit', 'crm:deal:edit', 'crm:deal:delete',
        'team:employee:read'], 'team'),
      keys(['wiki:article:edit'], 'own'),
      // Industry batch 3: fully operative (approves POs, runs production, manages helpdesk).
      keys(INVENTAR_ALL), keys(EINKAUF_ALL), keys(PRODUKTION_ALL),
      keys(VERTRAEGE_ALL), keys(HELPDESK_ALL),
      // Industry batch 4: fully operative (plans shifts, approves swaps +
      // reports, manages fleet/rentals/campaigns incl. GPS insight).
      keys(SCHICHTEN_ALL), keys(FUHRPARK_ALL), keys(VERMIETUNG_ALL),
      keys(RAPPORTE_ALL), keys(DIALER_ALL),
      // Batch 5: fully operative EXCEPT berichte:datev:read (finance data —
      // manager deliberately has no finance access) and the technical
      // kommunikation webhook config (admin/it_admin only). Leads team
      // communication (channels, inboxes, routing, canned responses),
      // approves time weeks/corrections, manages booking pages + categories.
      keys(BERICHTE_ALL.filter((k) => k !== 'berichte:datev:read')),
      keys(FORMULARE_ALL), keys(AUTOMATISIERUNG_ALL),
      keys(ZEITERFASSUNG_MGMT), keys(KALENDER_MGMT),
      keys(KOMMUNIKATION_MGMT.filter((k) => k !== 'kommunikation:webhook:manage')),
    ),
  },
  // Mitarbeiter — day-to-day work, own-scope editing, no management.
  member: {
    name: 'Employee',
    description: 'Day-to-day work with own-scope editing',
    color: 'hsl(142 71% 45%)',
    grants: merge(
      // Batch-5 curation: automatisierung is administration (flows can touch
      // modules the member cannot see) — module hidden for members entirely
      // instead of an empty default-deny shell. Personal automations stay a
      // future option (owner_id is a real user id already).
      view([...STANDARD_MODULES, ...INDUSTRY_MODULES.filter((m) => m !== 'automatisierung')]),
      keys(['work:task:read', 'work:task:create', 'work:task:be_assigned', 'work:task:comment',
        'work:project:read',
        'documents:file:read', 'documents:file:download', 'documents:file:upload',
        'crm:contact:read', 'crm:deal:read',
        'wiki:article:read', 'wiki:article:create',
        'team:employee:read', 'team:absence:read',
        'settings:personal:manage']),
      keys(['work:task:edit', 'work:time:log', 'documents:file:edit', 'wiki:article:edit'], 'own'),
      keys(['crm:contact:create', 'crm:contact:edit'], 'team'),
      // Industry batch 3: operative basics — records stock movements, counts
      // stocktakes, drafts + receives POs, runs the shop floor (incl. the
      // Laufkarte via export:run), raises + answers own helpdesk tickets.
      keys([
        'inventar:item:read', 'inventar:location:read', 'inventar:movement:read',
        'inventar:movement:create', 'inventar:inventur:read', 'inventar:inventur:count',
        'einkauf:po:read', 'einkauf:po:create', 'einkauf:po:receive',
        'einkauf:supplier:read', 'einkauf:catalog:read', 'einkauf:contract:read',
        'produktion:order:read', 'produktion:order:start', 'produktion:order:complete',
        'produktion:workstep:edit', 'produktion:bom:read', 'produktion:quality:read',
        'produktion:quality:create', 'produktion:machine:read', 'produktion:export:run',
        'vertraege:contract:read',
        'helpdesk:ticket:create', 'helpdesk:ticket:reply',
      ]),
      keys(['helpdesk:ticket:read'], 'own'),
      // Industry batch 4: operative basics — sees the shift plan, logs fuel +
      // trips, reports damage, runs the rental desk (handover + protocols),
      // writes own daily reports from templates (incl. the PDF via
      // export:run), dials as an agent. No plan management, no GPS, no
      // campaign administration.
      keys([
        'schichten:shift:read',
        'fuhrpark:vehicle:read', 'fuhrpark:service:read', 'fuhrpark:fuel:read',
        'fuhrpark:fuel:create', 'fuhrpark:trip:read', 'fuhrpark:trip:create',
        'fuhrpark:damage:create',
        'vermietung:object:read', 'vermietung:rental:read', 'vermietung:rental:create',
        'vermietung:rental:edit', 'vermietung:rental:handover', 'vermietung:inspection:create',
        'rapporte:report:create', 'rapporte:measurement:read', 'rapporte:measurement:manage',
        'rapporte:template:read', 'rapporte:export:run',
        'dialer:campaigns:read', 'dialer:calls:read', 'dialer:calls:write',
      ]),
      keys(['schichten:swap:read', 'schichten:swap:create', 'rapporte:report:read', 'rapporte:report:edit'], 'own'),
      // Batch 5: reads business reports + creates own ones from templates,
      // sees form submissions and triages them (field-service case, mirrors
      // Luke's 000234 member grant), creates chat channels (Slack-style
      // default). No form designer, no lifecycle, no schedules/shares, no
      // DATEV, no time-tracking administration.
      keys([
        'berichte:reports:read', 'berichte:reports:create',
        'formulare:schemas:read', 'formulare:submissions:read', 'formulare:submissions:write',
        'kommunikation:channel:manage',
      ]),
      keys(['berichte:reports:edit'], 'own'),
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
      // Industry batch 3+4: reads only (audit/tax-advisor sees, never changes).
      keys([...INDUSTRY_B3_READS, 'helpdesk:ticket:read', 'helpdesk:stats:view']),
      keys(INDUSTRY_B4_READS),
      // Batch 5: reads + DATEV view — the tax-advisor case (Elena) is exactly
      // who BWA/SuSa exist for. No export (readonly never downloads), no
      // zeiterfassung team view (foreign working hours stay HR-internal).
      keys([...INDUSTRY_B5_READS, 'berichte:datev:read']),
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

export function isPresetRole(roleId: string): roleId is RoleId {
  return roleId in ROLE_DEFS
}

// ── Custom roles (R-2 builder — mirrors Luke's tenant-scoped roles rows) ────

export interface CustomRoleDef {
  id: string
  name: string
  description: string
  color: string
  basedOn: string
  grants: GrantSpec
}

/** Tenant custom-role limit (KONZEPT §3: cap against role sprawl). */
export const CUSTOM_ROLE_LIMIT = 20

const CUSTOM_ROLES = new Map<string, CustomRoleDef>()
let customRoleSeq = 0

/**
 * One seeded custom role so list/editor/diff surfaces demo without setup:
 * a warehouse-temp clone of `extern` with two deviations (inventar visible,
 * documents downloadable). Nobody holds it → delete stays demoable.
 */
function seedCustomRoles(): void {
  customRoleSeq = 1
  CUSTOM_ROLES.set('role-c1', {
    id: 'role-c1',
    name: 'Lager & Logistik',
    description: 'Aushilfen im Lager: Aufgaben + Inventar-Einblick, Dokumente mit Download',
    color: 'hsl(38 92% 50%)',
    basedOn: 'extern',
    grants: merge(ROLE_DEFS.extern.grants, {
      [moduleViewKey('inventar')]: 'all',
      'documents:file:download': 'all',
      // Warehouse basics (R-3 batch 3: visibility alone would leave the
      // module empty under default-deny — temps record movements + count).
      'inventar:item:read': 'all',
      'inventar:location:read': 'all',
      'inventar:movement:read': 'all',
      'inventar:movement:create': 'all',
      'inventar:inventur:read': 'all',
      'inventar:inventur:count': 'all',
    }),
  })
}
seedCustomRoles()

export function customRoleCount(): number {
  return CUSTOM_ROLES.size
}

export function getCustomRole(roleId: string): CustomRoleDef | undefined {
  return CUSTOM_ROLES.get(roleId)
}

export function roleNameExists(name: string, exceptId?: string): boolean {
  const needle = name.trim().toLowerCase()
  if (PRESET_ROLE_IDS.some((id) => ROLE_DEFS[id].name.toLowerCase() === needle)) return true
  return [...CUSTOM_ROLES.values()].some(
    (r) => r.id !== exceptId && r.name.trim().toLowerCase() === needle,
  )
}

export function createCustomRole(input: {
  name: string
  description: string
  color: string
  basedOn: string
}): CustomRoleDef {
  customRoleSeq += 1
  const id = `role-c${customRoleSeq}`
  const baseGrants = getRoleGrants(input.basedOn) ?? {}
  const def: CustomRoleDef = {
    id,
    name: input.name.trim(),
    description: input.description.trim(),
    color: input.color,
    basedOn: input.basedOn,
    grants: { ...baseGrants },
  }
  CUSTOM_ROLES.set(id, def)
  return def
}

export function updateCustomRole(
  roleId: string,
  patch: Partial<Pick<CustomRoleDef, 'name' | 'description' | 'color'>>,
): CustomRoleDef | undefined {
  const def = CUSTOM_ROLES.get(roleId)
  if (!def) return undefined
  if (patch.name !== undefined) def.name = patch.name.trim()
  if (patch.description !== undefined) def.description = patch.description.trim()
  if (patch.color !== undefined) def.color = patch.color
  return def
}

export function deleteCustomRole(roleId: string): boolean {
  return CUSTOM_ROLES.delete(roleId)
}

export function setCustomRoleGrants(roleId: string, grants: GrantSpec): CustomRoleDef | undefined {
  const def = CUSTOM_ROLES.get(roleId)
  if (!def) return undefined
  def.grants = { ...grants }
  return def
}

/** Grant map of any role (preset or custom). */
export function getRoleGrants(roleId: string): GrantSpec | undefined {
  if (isPresetRole(roleId)) return ROLE_DEFS[roleId].grants
  return CUSTOM_ROLES.get(roleId)?.grants
}

// ── Demo account → roles assignment (mirrors Luke's user_roles n:m) ─────────

/**
 * Laura carries TWO roles (manager + hr_admin) — the multi-role union demo
 * that feeds the "Effektive Rechte" provenance view. Mutable in R-2: the
 * assignment handlers write here, A-1/team read through it.
 */
export const USER_ROLE_ASSIGNMENTS: Record<string, string[]> = {
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
export function rolesForUser(userId: string): string[] {
  return USER_ROLE_ASSIGNMENTS[userId] ?? ['member']
}

export function assignRoleToUser(userId: string, roleId: string): string[] {
  const current = USER_ROLE_ASSIGNMENTS[userId] ?? ['member']
  if (!current.includes(roleId)) USER_ROLE_ASSIGNMENTS[userId] = [...current, roleId]
  return USER_ROLE_ASSIGNMENTS[userId] ?? []
}

export function removeRoleFromUser(userId: string, roleId: string): string[] {
  const current = USER_ROLE_ASSIGNMENTS[userId] ?? []
  USER_ROLE_ASSIGNMENTS[userId] = current.filter((r) => r !== roleId)
  return USER_ROLE_ASSIGNMENTS[userId] ?? []
}

/** Accounts currently holding a role (guardrails + member views). */
export function membersOfRole(roleId: string): string[] {
  return Object.entries(USER_ROLE_ASSIGNMENTS)
    .filter(([, roles]) => roles.includes(roleId))
    .map(([userId]) => userId)
}

/**
 * Last-admin guardrail: number of accounts holding the full-access preset.
 * Removing/deactivating the last one must be rejected (Report C P0 set).
 */
export function fullAccessHolderCount(): number {
  return membersOfRole('admin').length
}

// ── Union resolution (server-side logic, mirrored by Luke later) ────────────

/** Resolve the effective capability map for a set of roles (widest scope wins, provenance kept). */
export function resolveCapabilities(roleIds: string[]): Record<string, CapabilityGrant> {
  const result: Record<string, CapabilityGrant> = {}
  for (const roleId of roleIds) {
    const grants = getRoleGrants(roleId)
    if (!grants) continue
    for (const [key, scope] of Object.entries(grants)) {
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

/** Role summary (id/name/isSystem/color) for effective-permission responses. */
export function roleSummary(roleId: string): Pick<Role, 'id' | 'name' | 'isSystem' | 'color'> {
  if (isPresetRole(roleId)) {
    return { id: roleId, name: ROLE_DEFS[roleId].name, isSystem: true, color: ROLE_DEFS[roleId].color }
  }
  const custom = CUSTOM_ROLES.get(roleId)
  return {
    id: roleId,
    name: custom?.name ?? roleId,
    isSystem: false,
    color: custom?.color ?? 'hsl(215 16% 47%)',
  }
}

/** All roles of the tenant: the 7 presets followed by custom roles. */
export function listRoles(): Role[] {
  const memberCounts: Record<string, number> = {}
  for (const roleIds of Object.values(USER_ROLE_ASSIGNMENTS)) {
    for (const r of roleIds) memberCounts[r] = (memberCounts[r] ?? 0) + 1
  }
  const presets: Role[] = PRESET_ROLE_IDS.map((id) => ({
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
  const customs: Role[] = [...CUSTOM_ROLES.values()].map((def) => ({
    id: def.id,
    name: def.name,
    description: def.description,
    tenantId: 'tenant-demo',
    basedOn: def.basedOn,
    isSystem: false,
    color: def.color,
    memberCount: memberCounts[def.id] ?? 0,
    capabilityCount: Object.keys(def.grants).length,
  }))
  return [...presets, ...customs]
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
