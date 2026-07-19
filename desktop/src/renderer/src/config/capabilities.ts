/**
 * Capability key mappings (RBAC R-1) — the single place that translates UI
 * surfaces (nav items, settings tabs, team tabs) to capability keys.
 *
 * Level-1 visibility: every module resolves to `<module>:module:view`.
 * Missing capability = hidden (default-deny; replaces the old default-allow
 * RESTRICTED_NAV_ITEMS in config/roles.ts). Key catalogue:
 * `.planning/rbac-block/CAPABILITY-KATALOG.md`.
 */

/** Canonical module keys (level-1 visibility targets). */
export const MODULE_KEYS = [
  'dashboard',
  'work',
  'kommunikation',
  'crm',
  'team',
  'video',
  'kalender',
  'zeiterfassung',
  'documents',
  'wiki',
  'mail',
  'finance',
  'infrastructure',
  'inventar',
  'schichten',
  'einkauf',
  'helpdesk',
  'fuhrpark',
  'produktion',
  'berichte',
  'vertraege',
  'formulare',
  'vermietung',
  'rapporte',
  'dialer',
  'automatisierung',
  'notifications',
  'settings',
  'admin',
  'security',
] as const

export type ModuleKey = (typeof MODULE_KEYS)[number]

/** Full level-1 visibility key for a module. */
export function moduleViewKey(module: ModuleKey): string {
  return `${module}:module:view`
}

/**
 * Sidebar nav-item id → module key. Two nav items (projects/tasks) share the
 * work module; `chat` is the merged kommunikation module; `meetings` is video.
 */
export const NAV_ITEM_MODULE: Record<string, ModuleKey> = {
  dashboard: 'dashboard',
  projects: 'work',
  tasks: 'work',
  chat: 'kommunikation',
  contacts: 'crm',
  team: 'team',
  meetings: 'video',
  calendar: 'kalender',
  zeiterfassung: 'zeiterfassung',
  documents: 'documents',
  wiki: 'wiki',
  mail: 'mail',
  finance: 'finance',
  infrastructure: 'infrastructure',
  inventar: 'inventar',
  schichten: 'schichten',
  einkauf: 'einkauf',
  helpdesk: 'helpdesk',
  fuhrpark: 'fuhrpark',
  produktion: 'produktion',
  berichte: 'berichte',
  vertraege: 'vertraege',
  formulare: 'formulare',
  vermietung: 'vermietung',
  rapporte: 'rapporte',
  dialer: 'dialer',
  automatisierung: 'automatisierung',
  notifications: 'notifications',
  settings: 'settings',
  // legacy admin surfaces reachable via nav in some layouts
  admin: 'admin',
  'security-admin': 'security',
}

/**
 * Resolve a dashboard/nav surface id (widget module, module card, quick
 * action) to its RBAC module key. Nav ids take precedence (tasks→work,
 * chat→kommunikation, meetings→video); ids that already are module keys
 * (e.g. `crm`) pass through. Unknown ids stay ungated (undefined).
 */
export function resolveModuleKey(id: string | undefined): ModuleKey | undefined {
  if (!id) return undefined
  if (id in NAV_ITEM_MODULE) return NAV_ITEM_MODULE[id]
  return (MODULE_KEYS as readonly string[]).includes(id) ? (id as ModuleKey) : undefined
}

/**
 * Settings-page tab key → required capability (tabs without an entry are
 * visible to everyone: profile, appearance, language, security(personal),
 * notifications, about).
 */
export const SETTINGS_TAB_CAPABILITY: Record<string, string> = {
  mail: 'mail:module:view',
  calendar: 'kalender:module:view',
  company: 'admin:company:manage',
  billing: 'admin:license:manage',
  integrations: 'admin:integrations:manage',
  business: 'admin:company:manage',
  team: 'team:employee:edit',
  privacy: 'security:module:view',
  ai: 'admin:ai:manage',
}

/**
 * Settings-overlay module entries follow module visibility: entry id → module
 * key (ids mostly match; calendar/dokumente/video map to their module keys).
 * Entries with an explicit SETTINGS_ENTRY_CAPABILITY take precedence.
 */
export const SETTINGS_ENTRY_MODULE: Record<string, ModuleKey> = {
  crm: 'crm',
  dialer: 'dialer',
  video: 'video',
  finance: 'finance',
  calendar: 'kalender',
  mail: 'mail',
  kommunikation: 'kommunikation',
  work: 'work',
  automatisierung: 'automatisierung',
  helpdesk: 'helpdesk',
  berichte: 'berichte',
  zeiterfassung: 'zeiterfassung',
  dokumente: 'documents',
  vertraege: 'vertraege',
  notifications: 'notifications',
  formulare: 'formulare',
  wiki: 'wiki',
  inventar: 'inventar',
  vermietung: 'vermietung',
  rapporte: 'rapporte',
  schichten: 'schichten',
  fuhrpark: 'fuhrpark',
  einkauf: 'einkauf',
  produktion: 'produktion',
  dashboard: 'dashboard',
}

/** Settings-overlay entry id → required capability (entries without one are public). */
export const SETTINGS_ENTRY_CAPABILITY: Record<string, string> = {
  team: 'team:employee:edit',
  company: 'admin:company:manage',
  billing: 'admin:license:manage',
  modulzuteilung: 'admin:modules:manage',
  integrations: 'admin:integrations:manage',
  it: 'admin:it:manage',
  security: 'security:module:view',
  branding: 'admin:branding:manage',
}

/**
 * Team-module tab key → required capability. Tabs without an entry are visible
 * to all (members, absences, orgchart, selfservice).
 */
export const TEAM_TAB_CAPABILITY: Record<string, string> = {
  requests: 'team:absence:approve',
  korrekturen: 'team:corrections:manage',
  personalakte: 'team:data_personal:view',
  lohnvorbereitung: 'team:payroll:view',
  schulungen: 'team:training:manage',
  onboarding: 'team:onboarding:manage',
}
