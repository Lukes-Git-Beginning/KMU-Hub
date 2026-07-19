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
  video: [],
  notifications: [],

  // Batch-5 (R-3 close-out) — subjects mirror Luke's permission seeds where
  // they exist: berichte `reports` PLURAL (000080), formulare
  // `schemas`/`submissions` (000129; webhooks seeded in BE but the FE webhook
  // UI is an unmounted stub → no key until it ships), automatisierung BE
  // resource is `automations` WITHOUT module prefix (000129) — FE keys carry
  // the module prefix, mapping noted in backend-gaps. datev/schedule/share and
  // all mini-catalogue subjects below are FE-first (no BE seeds yet).

  berichte: [
    base('berichte:reports:read'),
    base('berichte:reports:create'),
    base('berichte:reports:edit', true),
    base('berichte:reports:delete', true),
    fine('berichte:reports:publish'),
    fine('berichte:schedule:manage'),
    fine('berichte:share:manage'),
    base('berichte:datev:read'),
    fine('berichte:export:run'),
  ],

  formulare: [
    base('formulare:schemas:read'),
    base('formulare:schemas:create'),
    base('formulare:schemas:edit'),
    base('formulare:schemas:delete'),
    fine('formulare:schemas:publish'),
    base('formulare:submissions:read'),
    fine('formulare:submissions:write'),
    fine('formulare:share:manage'),
    fine('formulare:export:run'),
  ],

  automatisierung: [
    base('automatisierung:automations:read'),
    base('automatisierung:automations:create'),
    base('automatisierung:automations:edit', true),
    base('automatisierung:automations:delete', true),
    fine('automatisierung:automations:toggle', true),
    base('automatisierung:executions:read'),
  ],

  // Standard-module mini catalogues (batch 5): only genuine administration is
  // gated — personal daily use (chatting, own calendar, clocking in/out,
  // notifications) stays capability-free. video + notifications deliberately
  // stay level-1 only (no management surfaces / all-personal actions).

  kommunikation: [
    fine('kommunikation:channel:manage'),
    fine('kommunikation:team_inbox:manage'),
    fine('kommunikation:routing:manage'),
    fine('kommunikation:canned:manage'),
    fine('kommunikation:webhook:manage'),
  ],

  kalender: [
    fine('kalender:booking_page:manage'),
    fine('kalender:category:manage'),
  ],

  zeiterfassung: [
    fine('zeiterfassung:team:view'),
    fine('zeiterfassung:week:approve'),
    fine('zeiterfassung:corrections:approve'),
    fine('zeiterfassung:export:run'),
  ],

  infrastructure: [
    fine('infrastructure:service:manage'),
    fine('infrastructure:backup:manage'),
    fine('infrastructure:security:manage'),
    fine('infrastructure:updates:manage'),
    fine('infrastructure:logs:export'),
  ],

  // Batch-3 industry modules — subjects mirror Luke's permission seeds
  // (000084/000185/000241 inventar, 000086/000209 einkauf, 000088/000191
  // produktion, 000090 vertraege); helpdesk is FE-first (BE only has the
  // coarse helpdesk read/write pair — gap noted in backend-gaps §RBAC).

  inventar: [
    base('inventar:item:read'),
    base('inventar:item:create'),
    base('inventar:item:edit'),
    base('inventar:item:delete'),
    base('inventar:location:read'),
    base('inventar:movement:read'),
    base('inventar:movement:create'),
    fine('inventar:movement:adjust'),
    base('inventar:inventur:read'),
    base('inventar:inventur:create'),
    fine('inventar:inventur:count'),
    fine('inventar:inventur:book'),
    fine('inventar:attachment:manage'),
    fine('inventar:export:run'),
  ],

  einkauf: [
    base('einkauf:po:read'),
    base('einkauf:po:create'),
    base('einkauf:po:edit'),
    base('einkauf:po:delete'),
    fine('einkauf:po:send'),
    fine('einkauf:po:approve'),
    fine('einkauf:po:cancel'),
    fine('einkauf:po:receive'),
    base('einkauf:supplier:read'),
    base('einkauf:supplier:create'),
    base('einkauf:supplier:edit'),
    fine('einkauf:supplier:deactivate'),
    fine('einkauf:rating:create'),
    base('einkauf:catalog:read'),
    base('einkauf:contract:read'),
    fine('einkauf:contract:call'),
    fine('einkauf:export:run'),
  ],

  produktion: [
    base('produktion:order:read'),
    base('produktion:order:create'),
    base('produktion:order:edit'),
    base('produktion:order:delete'),
    fine('produktion:order:start'),
    fine('produktion:order:complete'),
    fine('produktion:order:cancel'),
    fine('produktion:workstep:edit'),
    base('produktion:bom:read'),
    base('produktion:bom:create'),
    base('produktion:bom:edit'),
    base('produktion:quality:read'),
    base('produktion:quality:create'),
    base('produktion:machine:read'),
    fine('produktion:machine:manage'),
    fine('produktion:export:run'),
  ],

  vertraege: [
    base('vertraege:contract:read'),
    base('vertraege:contract:create'),
    base('vertraege:contract:edit'),
    base('vertraege:contract:delete'),
    fine('vertraege:contract:terminate'),
  ],

  helpdesk: [
    base('helpdesk:ticket:read', true),
    base('helpdesk:ticket:create'),
    base('helpdesk:ticket:reply'),
    base('helpdesk:ticket:edit', true),
    fine('helpdesk:kb:manage'),
    fine('helpdesk:canned:manage'),
    fine('helpdesk:stats:view'),
  ],

  // Batch-4 industry modules — subjects mirror Luke's permission seeds
  // (000095/000161 schichten incl. swap:create/approve, 000097/000196
  // fuhrpark, 000099 vermietung, 000093/000100/000164 rapporte incl.
  // report:approve, 000068 dialer with PLURAL subjects campaigns/calls).
  // FE folds rapporte line/attachment into report:* and fuhrpark document
  // into vehicle:read (read-only display) — mapping noted in backend-gaps.

  schichten: [
    base('schichten:shift:read'),
    fine('schichten:shift:publish'),
    base('schichten:assignment:manage'),
    base('schichten:template:read'),
    base('schichten:template:manage'),
    base('schichten:swap:read', true),
    fine('schichten:swap:create', true),
    fine('schichten:swap:approve'),
    fine('schichten:export:run'),
  ],

  fuhrpark: [
    base('fuhrpark:vehicle:read'),
    base('fuhrpark:vehicle:manage'),
    base('fuhrpark:service:read'),
    base('fuhrpark:service:create'),
    base('fuhrpark:fuel:read'),
    base('fuhrpark:fuel:create'),
    base('fuhrpark:trip:read'),
    base('fuhrpark:trip:create'),
    base('fuhrpark:damage:create'),
    fine('fuhrpark:gps:read'),
    fine('fuhrpark:export:run'),
  ],

  vermietung: [
    base('vermietung:object:read'),
    base('vermietung:object:create'),
    base('vermietung:object:edit'),
    base('vermietung:object:delete'),
    base('vermietung:rental:read'),
    base('vermietung:rental:create'),
    base('vermietung:rental:edit'),
    fine('vermietung:rental:cancel'),
    fine('vermietung:rental:handover'),
    base('vermietung:inspection:create'),
    fine('vermietung:export:run'),
  ],

  rapporte: [
    base('rapporte:report:read', true),
    base('rapporte:report:create'),
    base('rapporte:report:edit', true),
    base('rapporte:report:delete', true),
    fine('rapporte:report:approve'),
    base('rapporte:measurement:read'),
    base('rapporte:measurement:manage'),
    base('rapporte:template:read'),
    fine('rapporte:export:run'),
  ],

  dialer: [
    base('dialer:campaigns:read'),
    base('dialer:campaigns:manage'),
    base('dialer:calls:read'),
    base('dialer:calls:write'),
    fine('dialer:outcomes:manage'),
    fine('dialer:agent:manage'),
  ],

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
    base('crm:deal:delete', true),
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
    fine('wiki:category:manage'),
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
