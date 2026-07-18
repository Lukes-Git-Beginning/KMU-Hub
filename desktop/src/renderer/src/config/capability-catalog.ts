/**
 * Capability catalogue (RBAC R-2) — the UI's single source of truth for which
 * level-2/3 capabilities exist per module, mirrored from
 * `.planning/rbac-block/CAPABILITY-KATALOG.md`.
 *
 * Level 1 (module visibility) is implicit via `moduleViewKey()` and NOT listed
 * here. Core modules are fully curated; the remaining modules carry an empty
 * list until their R-3 batch curates them (the editor shows visibility only
 * plus a "fine switches follow" hint for those).
 *
 * The preset grant seed (mocks/data/rbac.ts ROLE_DEFS) derives its key lists
 * from this catalogue — one place to add a capability, no drift.
 *
 * i18n: labels compose from `rbac.subject.<subject>` + `rbac.action.<action>`
 * (subject = middle key segment), module names from `rbac.module.<module>` —
 * the same building blocks BerechtigungenTab already renders.
 */
import type { ModuleKey } from './capabilities'

/** Editor tree grouping. */
export type ModuleCategory = 'standard' | 'industry' | 'verwaltung'

export interface CapabilityDef {
  /** Full capability key (`resource:action`). */
  key: string
  /** Level-3 fine switch (level-2 base action otherwise). */
  fine: boolean
  /** Grant carries a data-scope choice (own/team/all) in the editor. */
  scopeable: boolean
}

const base = (key: string, scopeable = false): CapabilityDef => ({ key, fine: false, scopeable })
const fine = (key: string, scopeable = false): CapabilityDef => ({ key, fine: true, scopeable })

export const MODULE_CATEGORY: Record<ModuleKey, ModuleCategory> = {
  dashboard: 'standard',
  work: 'standard',
  kommunikation: 'standard',
  crm: 'standard',
  team: 'standard',
  video: 'standard',
  kalender: 'standard',
  zeiterfassung: 'standard',
  documents: 'standard',
  wiki: 'standard',
  mail: 'standard',
  finance: 'standard',
  infrastructure: 'standard',
  notifications: 'standard',
  inventar: 'industry',
  schichten: 'industry',
  einkauf: 'industry',
  helpdesk: 'industry',
  fuhrpark: 'industry',
  produktion: 'industry',
  berichte: 'industry',
  vertraege: 'industry',
  formulare: 'industry',
  vermietung: 'industry',
  rapporte: 'industry',
  dialer: 'industry',
  automatisierung: 'industry',
  settings: 'verwaltung',
  admin: 'verwaltung',
  security: 'verwaltung',
}

/**
 * Level-2/3 capabilities per module. Missing/empty entry ⇒ only level-1
 * visibility is curated yet (R-3 batches fill these in).
 */
export const CAPABILITY_CATALOG: Record<ModuleKey, CapabilityDef[]> = {
  dashboard: [],
  kommunikation: [],
  video: [],
  kalender: [],
  zeiterfassung: [],
  infrastructure: [],
  notifications: [],
  inventar: [],
  schichten: [],
  einkauf: [],
  helpdesk: [],
  fuhrpark: [],
  produktion: [],
  berichte: [],
  vertraege: [],
  formulare: [],
  vermietung: [],
  rapporte: [],
  dialer: [],
  automatisierung: [],

  work: [
    base('work:task:read', true),
    base('work:task:create'),
    base('work:task:edit', true),
    base('work:task:delete', true),
    fine('work:task:be_assigned', true),
    fine('work:task:comment', true),
    base('work:project:read', true),
    base('work:project:create'),
    base('work:project:edit', true),
    base('work:project:delete', true),
    fine('work:project:manage_members'),
    fine('work:time:log', true),
    base('work:board:export'),
  ],

  documents: [
    base('documents:file:read', true),
    fine('documents:file:download'),
    base('documents:file:upload'),
    base('documents:file:edit', true),
    base('documents:file:delete', true),
    fine('documents:share:manage'),
    fine('documents:share_link:create'),
    fine('documents:version:restore'),
    fine('documents:template:manage'),
  ],

  crm: [
    base('crm:contact:read', true),
    base('crm:contact:create'),
    base('crm:contact:edit', true),
    base('crm:contact:delete', true),
    fine('crm:contact:export'),
    base('crm:deal:read', true),
    base('crm:deal:create'),
    base('crm:deal:edit', true),
    fine('crm:pipeline:manage'),
    fine('crm:import:run'),
    fine('crm:advisory:read'),
    fine('crm:advisory:write'),
    fine('crm:segment:override'),
  ],

  finance: [
    base('finance:invoice:read'),
    base('finance:invoice:create'),
    base('finance:invoice:edit'),
    base('finance:invoice:delete'),
    fine('finance:invoice:send'),
    fine('finance:dunning:run'),
    base('finance:quote:read'),
    base('finance:quote:create'),
    fine('finance:quote:send'),
    fine('finance:amounts:view'),
    fine('finance:export:run'),
    fine('finance:incoming:review'),
    fine('finance:incoming:book'),
    fine('finance:settings:manage'),
  ],

  team: [
    base('team:employee:read', true),
    base('team:employee:create'),
    base('team:employee:edit', true),
    fine('team:employee:deactivate'),
    fine('team:data_personal:view'),
    fine('team:data_personal:edit'),
    fine('team:data_job:view'),
    fine('team:data_job:edit'),
    fine('team:salary:view'),
    fine('team:salary:edit'),
    fine('team:documents:view'),
    fine('team:documents:edit'),
    base('team:absence:read', true),
    fine('team:absence:approve'),
    fine('team:role:assign'),
    fine('team:training:manage'),
    fine('team:payroll:view'),
    fine('team:payroll:run'),
    fine('team:corrections:manage'),
    fine('team:onboarding:manage'),
  ],

  wiki: [
    base('wiki:article:read', true),
    base('wiki:article:create'),
    base('wiki:article:edit', true),
    base('wiki:article:delete', true),
    fine('wiki:article:publish'),
    fine('wiki:share_token:create'),
    fine('wiki:template:manage'),
  ],

  mail: [fine('mail:settings:manage')],

  settings: [fine('settings:personal:manage'), fine('settings:tenant:manage')],

  admin: [
    fine('admin:user:read'),
    fine('admin:user:invite'),
    fine('admin:user:deactivate'),
    fine('admin:role:read'),
    fine('admin:role:create'),
    fine('admin:role:edit'),
    fine('admin:role:delete'),
    fine('admin:role:assign'),
    fine('admin:license:manage'),
    fine('admin:branding:manage'),
    fine('admin:integrations:manage'),
    fine('admin:company:manage'),
    fine('admin:modules:manage'),
    fine('admin:it:manage'),
    fine('admin:ai:manage'),
    fine('admin:impersonate:run'),
  ],

  security: [
    fine('security:audit:read'),
    fine('security:policy:manage'),
    fine('security:gdpr:execute'),
  ],
}

/** All curated level-2/3 keys of a module (empty ⇒ visibility only so far). */
export function catalogCapabilityKeys(module: ModuleKey): string[] {
  return CAPABILITY_CATALOG[module].map((c) => c.key)
}

/** Catalogue def for a key (scope/level lookups in editor + compare views). */
export function catalogDef(key: string): CapabilityDef | undefined {
  const module = key.split(':')[0] as ModuleKey
  return CAPABILITY_CATALOG[module]?.find((c) => c.key === key)
}
