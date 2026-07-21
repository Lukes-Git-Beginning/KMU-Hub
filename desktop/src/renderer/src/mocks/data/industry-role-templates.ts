/**
 * Industry role templates (RBAC R-5 — Part C.1).
 *
 * 3 sets × 4 roles = 12 pre-configured role templates surfaced in the
 * CloneRoleDialog "Aus Vorlage" tab. Every template maps to an existing
 * basedOn preset (admin/manager/member/extern) and carries a fully
 * specified grant set built from verified catalogue keys only.
 *
 * Key rules:
 * - Every key MUST exist in CAPABILITY_CATALOG or be a moduleViewKey().
 * - No invented keys — deviations noted in comments and in the report.
 * - Grants are ADDITIVE to the basedOn preset in the dialog (the pre-fill
 *   replaces the full form, the editor then lets the user adjust before save).
 */
import { moduleViewKey, type ModuleKey } from '@/config/capabilities'
import type { BusinessProfileId } from '@/config/business-profiles'

// ── Grant types (mirrors mocks/data/rbac.ts) ────────────────────────────────

import type { CapabilityScope } from '@/api/rbac-types'

type GrantSpec = Record<string, CapabilityScope>

const keys = (list: string[], scope: CapabilityScope = 'all'): GrantSpec =>
  Object.fromEntries(list.map((k) => [k, scope]))

const view = (modules: ModuleKey[]): GrantSpec =>
  keys(modules.map((m) => moduleViewKey(m)))

const merge = (...specs: GrantSpec[]): GrantSpec => Object.assign({}, ...specs)

// ── Helper: all module view-keys that are NOT in the given list ──────────────

function hideModules(visible: ModuleKey[]): GrantSpec {
  // Returns view-keys only for visible modules; all others are absent = hidden.
  return view(visible)
}

// ── Template interface ───────────────────────────────────────────────────────

export interface IndustryRoleTemplate {
  id: string
  /** i18n key: rbac.template.role.<id>.label */
  labelKey: string
  /** i18n key: rbac.template.role.<id>.description */
  descriptionKey: string
  /** System preset to use as basedOn in CloneRoleDialog (seeds the editor). */
  basedOn: 'admin' | 'manager' | 'member' | 'extern'
  /** Curated grant spec — replaces basedOn grants for pre-fill. */
  grants: GrantSpec
  /** Suggested accent colour from the existing palette. */
  color: string
  /**
   * Short bullet texts (i18n keys) shown on the template card to explain
   * the key access separations. Max 3.
   */
  highlightKeys: string[]
}

export interface IndustryRoleSet {
  id: string
  /** i18n key: rbac.template.set.<id> */
  labelKey: string
  /** Business-profile IDs this set is most relevant for (used for ordering). */
  businessProfileIds: BusinessProfileId[]
  roles: IndustryRoleTemplate[]
}

// ── SET 1: Handwerk ──────────────────────────────────────────────────────────
//
// Relevant for businessProfileIds: handwerk, bau

const SET_HANDWERK: IndustryRoleSet = {
  id: 'handwerk',
  labelKey: 'rbac.template.set.handwerk',
  businessProfileIds: ['handwerk', 'bau'],
  roles: [
    // Role 1: Büro & Auftragswesen (basedOn manager)
    // VOLL: crm, finance operativ, kalender, documents, formulare, vertraege, einkauf
    // zeiterfassung:team:view, berichte:reports:read
    // VERBORGEN: HR-Gehalt, settings:tenant, infrastructure, admin, security, dialer
    {
      id: 'handwerk_buero',
      labelKey: 'rbac.template.role.handwerk_buero.label',
      descriptionKey: 'rbac.template.role.handwerk_buero.description',
      basedOn: 'manager',
      color: 'hsl(38 92% 50%)',
      highlightKeys: [
        'rbac.template.role.handwerk_buero.highlight.1',
        'rbac.template.role.handwerk_buero.highlight.2',
        'rbac.template.role.handwerk_buero.highlight.3',
      ],
      grants: merge(
        // Visible modules
        view(['dashboard', 'crm', 'finance', 'kalender', 'documents', 'formulare',
          'vertraege', 'einkauf', 'zeiterfassung', 'berichte', 'notifications',
          'settings', 'mail', 'wiki', 'kommunikation']),
        // CRM — full operative
        keys([
          'crm:contact:read', 'crm:contact:create', 'crm:contact:edit',
          'crm:deal:read', 'crm:deal:create', 'crm:deal:edit',
          'crm:pipeline:manage', 'crm:import:run',
        ]),
        // Finance — full operative (like manager level)
        keys([
          'finance:invoice:read', 'finance:invoice:create', 'finance:invoice:edit',
          'finance:invoice:send', 'finance:dunning:run',
          'finance:quote:read', 'finance:quote:create', 'finance:quote:send',
          'finance:amounts:view', 'finance:export:run',
          'finance:incoming:review', 'finance:incoming:book',
          // NOTE: finance:settings:manage withheld — only admin-level
        ]),
        // Kalender
        keys(['kalender:booking_page:manage', 'kalender:category:manage']),
        // Documents — full operative
        keys([
          'documents:file:read', 'documents:file:download', 'documents:file:upload',
          'documents:file:edit', 'documents:share:manage', 'documents:share_link:create',
          'documents:version:restore', 'documents:template:manage',
        ]),
        // Formulare — full operative
        keys([
          'formulare:schemas:read', 'formulare:schemas:create', 'formulare:schemas:edit',
          'formulare:schemas:publish', 'formulare:submissions:read', 'formulare:submissions:write',
          'formulare:export:run',
        ]),
        // Vertraege — full operative
        keys([
          'vertraege:contract:read', 'vertraege:contract:create',
          'vertraege:contract:edit', 'vertraege:contract:terminate',
        ]),
        // Einkauf — full operative
        keys([
          'einkauf:po:read', 'einkauf:po:create', 'einkauf:po:edit',
          'einkauf:po:send', 'einkauf:po:approve',
          'einkauf:supplier:read', 'einkauf:supplier:create', 'einkauf:supplier:edit',
          'einkauf:catalog:read', 'einkauf:contract:read',
          'einkauf:rating:create', 'einkauf:export:run',
        ]),
        // Zeiterfassung — team view only (no approval)
        keys(['zeiterfassung:team:view']),
        // Berichte — read only
        keys(['berichte:reports:read']),
        // Settings personal only
        keys(['settings:personal:manage']),
        // Wiki read
        keys(['wiki:article:read']),
      ),
    },

    // Role 2: Bauleiter / Projektleiter (basedOn manager)
    // VOLL: work, rapporte (inkl. approve), zeiterfassung team+approvals,
    //       schichten assignment:manage, kalender, documents, formulare
    // READ: crm contact/deal read, inventar item/location/movement read
    // VERBORGEN: finance KOMPLETT (kein view-Key!), einkauf, berichte, HR-Daten
    {
      id: 'handwerk_bauleiter',
      labelKey: 'rbac.template.role.handwerk_bauleiter.label',
      descriptionKey: 'rbac.template.role.handwerk_bauleiter.description',
      basedOn: 'manager',
      color: 'hsl(25 95% 53%)',
      highlightKeys: [
        'rbac.template.role.handwerk_bauleiter.highlight.1',
        'rbac.template.role.handwerk_bauleiter.highlight.2',
        'rbac.template.role.handwerk_bauleiter.highlight.3',
      ],
      grants: merge(
        view(['dashboard', 'work', 'rapporte', 'zeiterfassung', 'schichten',
          'kalender', 'documents', 'formulare', 'crm', 'inventar',
          'notifications', 'settings', 'wiki', 'kommunikation']),
        // Work — full
        keys([
          'work:task:read', 'work:task:create', 'work:task:edit', 'work:task:delete',
          'work:task:be_assigned', 'work:task:comment', 'work:time:log',
          'work:project:read', 'work:project:create', 'work:project:edit',
          'work:project:delete', 'work:project:manage_members', 'work:board:export',
        ]),
        // Rapporte — full + approve
        keys([
          'rapporte:report:read', 'rapporte:report:create', 'rapporte:report:edit',
          'rapporte:report:delete', 'rapporte:report:approve',
          'rapporte:measurement:read', 'rapporte:measurement:manage',
          'rapporte:template:read', 'rapporte:export:run',
        ]),
        // Zeiterfassung — full management (team + approvals)
        keys([
          'zeiterfassung:team:view', 'zeiterfassung:week:approve',
          'zeiterfassung:corrections:approve', 'zeiterfassung:export:run',
        ]),
        // Schichten — assignment manage only (no full shift publishing for bauleiter)
        keys([
          'schichten:shift:read', 'schichten:assignment:manage',
          'schichten:template:read', 'schichten:swap:read', 'schichten:swap:approve',
        ]),
        // Kalender
        keys(['kalender:booking_page:manage', 'kalender:category:manage']),
        // Documents
        keys([
          'documents:file:read', 'documents:file:download', 'documents:file:upload',
          'documents:file:edit', 'documents:share:manage', 'documents:template:manage',
        ]),
        // Formulare
        keys([
          'formulare:schemas:read', 'formulare:submissions:read',
          'formulare:submissions:write',
        ]),
        // CRM — read only
        keys(['crm:contact:read', 'crm:deal:read']),
        // Inventar — read (item/location/movement)
        keys(['inventar:item:read', 'inventar:location:read', 'inventar:movement:read']),
        // Wiki read
        keys(['wiki:article:read']),
        // Settings personal
        keys(['settings:personal:manage']),
      ),
    },

    // Role 3: Monteur / Geselle (basedOn member)
    // rapporte create+read own, zeiterfassung own, schichten own + swap:create
    // formulare, inventar movement:create + item:read, kalender own, documents read
    // VERBORGEN: crm, finance, einkauf, berichte, HR
    {
      id: 'handwerk_monteur',
      labelKey: 'rbac.template.role.handwerk_monteur.label',
      descriptionKey: 'rbac.template.role.handwerk_monteur.description',
      basedOn: 'member',
      color: 'hsl(160 84% 39%)',
      highlightKeys: [
        'rbac.template.role.handwerk_monteur.highlight.1',
        'rbac.template.role.handwerk_monteur.highlight.2',
        'rbac.template.role.handwerk_monteur.highlight.3',
      ],
      grants: merge(
        view(['dashboard', 'rapporte', 'zeiterfassung', 'schichten', 'formulare',
          'inventar', 'kalender', 'documents', 'notifications', 'settings']),
        // Rapporte — create + own read/edit
        keys(['rapporte:report:create', 'rapporte:measurement:read',
          'rapporte:measurement:manage', 'rapporte:template:read', 'rapporte:export:run']),
        keys(['rapporte:report:read', 'rapporte:report:edit'], 'own'),
        // Zeiterfassung — own only (no team keys available at this level without catalogDef scopeable)
        // NOTE: zeiterfassung fine keys are not scopeable per catalogDef; only team:view exists.
        // Decision: no zeiterfassung capability keys for Monteur (personal clocking = implicit in module visibility).
        // Schichten — own + swap create
        keys(['schichten:shift:read', 'schichten:swap:read']),
        keys(['schichten:swap:create'], 'own'),
        // Formulare — read + submit
        keys([
          'formulare:schemas:read', 'formulare:submissions:read',
          'formulare:submissions:write',
        ]),
        // Inventar — item read + movement create
        keys(['inventar:item:read', 'inventar:location:read', 'inventar:movement:read',
          'inventar:movement:create']),
        // Kalender — own use (no mgmt keys needed)
        // Documents — read
        keys(['documents:file:read', 'documents:file:download']),
        // Settings personal
        keys(['settings:personal:manage']),
      ),
    },

    // Role 4: Azubi (basedOn member)
    // zeiterfassung own, formulare read+submit, rapporte READ only, kalender own,
    // wiki read, documents read
    // VERBORGEN: crm, finance, einkauf, berichte, HR, inventar, schichten-swap
    // NOTE: schichten:shift:read included (schichten module visible = shift plan viewable)
    {
      id: 'handwerk_azubi',
      labelKey: 'rbac.template.role.handwerk_azubi.label',
      descriptionKey: 'rbac.template.role.handwerk_azubi.description',
      basedOn: 'member',
      color: 'hsl(200 98% 39%)',
      highlightKeys: [
        'rbac.template.role.handwerk_azubi.highlight.1',
        'rbac.template.role.handwerk_azubi.highlight.2',
        'rbac.template.role.handwerk_azubi.highlight.3',
      ],
      grants: merge(
        view(['dashboard', 'zeiterfassung', 'formulare', 'rapporte', 'kalender',
          'wiki', 'documents', 'notifications', 'settings', 'schichten']),
        // Formulare — read + submit
        keys(['formulare:schemas:read', 'formulare:submissions:read',
          'formulare:submissions:write']),
        // Rapporte — read only (no create/edit)
        keys(['rapporte:report:read', 'rapporte:template:read']),
        // Wiki — read
        keys(['wiki:article:read']),
        // Documents — read
        keys(['documents:file:read', 'documents:file:download']),
        // Schichten — shift plan read only (no swap)
        keys(['schichten:shift:read']),
        // Settings personal
        keys(['settings:personal:manage']),
      ),
    },
  ],
}

// ── SET 2: Dienstleister ─────────────────────────────────────────────────────
//
// Relevant for businessProfileIds: dienstleistung, it_tech

const SET_DIENSTLEISTER: IndustryRoleSet = {
  id: 'dienstleister',
  labelKey: 'rbac.template.set.dienstleister',
  businessProfileIds: ['dienstleistung', 'it_tech'],
  roles: [
    // Role 5: Projektleiter / Senior (basedOn manager)
    // VOLL: crm, work, kalender, zeiterfassung team, documents, wiki, vertraege, formulare
    // finance: view-Level only (module sichtbar + read-Keys, KEINE export/send)
    // berichte:reports:read
    // VERBORGEN: HR-Gehalt, settings:tenant, admin, security
    {
      id: 'dl_projektleiter',
      labelKey: 'rbac.template.role.dl_projektleiter.label',
      descriptionKey: 'rbac.template.role.dl_projektleiter.description',
      basedOn: 'manager',
      color: 'hsl(245 58% 61%)',
      highlightKeys: [
        'rbac.template.role.dl_projektleiter.highlight.1',
        'rbac.template.role.dl_projektleiter.highlight.2',
        'rbac.template.role.dl_projektleiter.highlight.3',
      ],
      grants: merge(
        view(['dashboard', 'crm', 'work', 'kalender', 'zeiterfassung', 'documents',
          'wiki', 'vertraege', 'formulare', 'finance', 'berichte',
          'notifications', 'settings', 'mail', 'kommunikation']),
        // CRM — full
        keys([
          'crm:contact:read', 'crm:contact:create', 'crm:contact:edit',
          'crm:deal:read', 'crm:deal:create', 'crm:deal:edit',
          'crm:pipeline:manage',
        ]),
        // Work — full
        keys([
          'work:task:read', 'work:task:create', 'work:task:edit', 'work:task:delete',
          'work:task:be_assigned', 'work:task:comment', 'work:time:log',
          'work:project:read', 'work:project:create', 'work:project:edit',
          'work:project:manage_members', 'work:board:export',
        ]),
        // Kalender — full
        keys(['kalender:booking_page:manage', 'kalender:category:manage']),
        // Zeiterfassung — team view
        keys(['zeiterfassung:team:view', 'zeiterfassung:export:run']),
        // Documents — full
        keys([
          'documents:file:read', 'documents:file:download', 'documents:file:upload',
          'documents:file:edit', 'documents:share:manage', 'documents:template:manage',
        ]),
        // Wiki — full
        keys([
          'wiki:article:read', 'wiki:article:create', 'wiki:article:edit',
          'wiki:article:publish', 'wiki:category:manage',
        ]),
        // Vertraege — full
        keys([
          'vertraege:contract:read', 'vertraege:contract:create',
          'vertraege:contract:edit', 'vertraege:contract:terminate',
        ]),
        // Formulare — full
        keys([
          'formulare:schemas:read', 'formulare:schemas:create', 'formulare:schemas:edit',
          'formulare:submissions:read', 'formulare:submissions:write',
        ]),
        // Finance — VIEW LEVEL ONLY (no send, no export, no dunning, no settings)
        keys([
          'finance:invoice:read', 'finance:quote:read', 'finance:amounts:view',
          'finance:incoming:review',
        ]),
        // Berichte — read
        keys(['berichte:reports:read']),
        // Settings personal
        keys(['settings:personal:manage']),
      ),
    },

    // Role 6: Consultant / Bearbeiter (basedOn member)
    // work own/team-Basics, zeiterfassung own, wiki read+create+edit, documents read+upload
    // kalender own, crm READ
    // VERBORGEN: finance, berichte, HR, dialer
    {
      id: 'dl_consultant',
      labelKey: 'rbac.template.role.dl_consultant.label',
      descriptionKey: 'rbac.template.role.dl_consultant.description',
      basedOn: 'member',
      color: 'hsl(330 81% 60%)',
      highlightKeys: [
        'rbac.template.role.dl_consultant.highlight.1',
        'rbac.template.role.dl_consultant.highlight.2',
        'rbac.template.role.dl_consultant.highlight.3',
      ],
      grants: merge(
        view(['dashboard', 'work', 'zeiterfassung', 'wiki', 'documents', 'kalender',
          'crm', 'notifications', 'settings', 'kommunikation']),
        // Work — own + team basics
        keys(['work:task:read', 'work:task:create', 'work:task:be_assigned',
          'work:task:comment', 'work:project:read', 'work:time:log', 'work:board:export']),
        keys(['work:task:edit'], 'own'),
        // Zeiterfassung — no fine keys (own clocking = implicit); team:view withheld
        // Wiki — read + create + edit
        keys(['wiki:article:read', 'wiki:article:create']),
        keys(['wiki:article:edit'], 'own'),
        // Documents — read + upload
        keys(['documents:file:read', 'documents:file:download', 'documents:file:upload']),
        // CRM — contact read only
        keys(['crm:contact:read']),
        // Settings personal
        keys(['settings:personal:manage']),
      ),
    },

    // Role 7: Backoffice / Office (basedOn member)
    // finance operativ (invoices/expenses create+edit+read, KEINE tenant-Settings)
    // zeiterfassung team:view + export:run, kalender, documents, vertraege, einkauf
    // crm contact:read, berichte:reports:read
    // VERBORGEN: crm-Deals/Pipeline (deal-Keys weglassen; contact:read bleibt), HR-Gehalt, admin
    {
      id: 'dl_backoffice',
      labelKey: 'rbac.template.role.dl_backoffice.label',
      descriptionKey: 'rbac.template.role.dl_backoffice.description',
      basedOn: 'member',
      color: 'hsl(15 86% 50%)',
      highlightKeys: [
        'rbac.template.role.dl_backoffice.highlight.1',
        'rbac.template.role.dl_backoffice.highlight.2',
        'rbac.template.role.dl_backoffice.highlight.3',
      ],
      grants: merge(
        view(['dashboard', 'finance', 'zeiterfassung', 'kalender', 'documents',
          'vertraege', 'einkauf', 'crm', 'berichte', 'notifications', 'settings',
          'mail', 'kommunikation']),
        // Finance — operativ (no send/dunning for member-level; no settings:manage)
        keys([
          'finance:invoice:read', 'finance:invoice:create', 'finance:invoice:edit',
          'finance:quote:read', 'finance:quote:create',
          'finance:amounts:view', 'finance:incoming:review', 'finance:incoming:book',
          // NOTE: finance:invoice:send and finance:dunning:run withheld for backoffice member
          // NOTE: finance:export:run withheld — export is manager-level
        ]),
        // Zeiterfassung — team view + export
        keys(['zeiterfassung:team:view', 'zeiterfassung:export:run']),
        // Kalender
        keys(['kalender:category:manage']),
        // Documents
        keys([
          'documents:file:read', 'documents:file:download', 'documents:file:upload',
          'documents:file:edit', 'documents:template:manage',
        ]),
        // Vertraege
        keys([
          'vertraege:contract:read', 'vertraege:contract:create', 'vertraege:contract:edit',
        ]),
        // Einkauf — operative
        keys([
          'einkauf:po:read', 'einkauf:po:create', 'einkauf:po:edit',
          'einkauf:supplier:read', 'einkauf:supplier:create',
          'einkauf:catalog:read', 'einkauf:contract:read',
        ]),
        // CRM — contact read only (NO deals/pipeline)
        keys(['crm:contact:read']),
        // Berichte — read
        keys(['berichte:reports:read']),
        // Settings personal
        keys(['settings:personal:manage']),
      ),
    },

    // Role 8: Freelancer / Extern (basedOn extern)
    // zeiterfassung own, formulare, work own-Basics, documents read, kalender own
    // VERBORGEN: crm gesamt, finance, berichte, wiki, HR
    {
      id: 'dl_freelancer',
      labelKey: 'rbac.template.role.dl_freelancer.label',
      descriptionKey: 'rbac.template.role.dl_freelancer.description',
      basedOn: 'extern',
      color: 'hsl(180 45% 42%)',
      highlightKeys: [
        'rbac.template.role.dl_freelancer.highlight.1',
        'rbac.template.role.dl_freelancer.highlight.2',
        'rbac.template.role.dl_freelancer.highlight.3',
      ],
      grants: merge(
        view(['dashboard', 'zeiterfassung', 'formulare', 'work', 'documents',
          'kalender', 'notifications', 'settings']),
        // Work — own basics (like extern preset)
        keys(['work:project:read']),
        keys(['work:task:read', 'work:task:be_assigned', 'work:task:comment'], 'own'),
        keys(['work:time:log'], 'own'),
        // Formulare — read + submit
        keys(['formulare:schemas:read', 'formulare:submissions:read',
          'formulare:submissions:write']),
        // Documents — read
        keys(['documents:file:read', 'documents:file:download']),
        // Settings personal
        keys(['settings:personal:manage']),
      ),
    },
  ],
}

// ── SET 3: Handel ────────────────────────────────────────────────────────────
//
// Relevant for businessProfileIds: einzelhandel, logistik

const SET_HANDEL: IndustryRoleSet = {
  id: 'handel',
  labelKey: 'rbac.template.set.handel',
  businessProfileIds: ['einzelhandel', 'logistik'],
  roles: [
    // Role 9: Filialleiter (basedOn manager)
    // VOLL: crm, kalender, zeiterfassung team, schichten, inventar, team read/directory
    // finance view-Level, berichte:reports:read
    // VERBORGEN: einkauf (EK-Preise!), HR-Gehalt, settings:tenant
    // NOTE: 1.0-Limit „kein Standort-Scope" in description
    {
      id: 'handel_filialleiter',
      labelKey: 'rbac.template.role.handel_filialleiter.label',
      descriptionKey: 'rbac.template.role.handel_filialleiter.description',
      basedOn: 'manager',
      color: 'hsl(84 59% 40%)',
      highlightKeys: [
        'rbac.template.role.handel_filialleiter.highlight.1',
        'rbac.template.role.handel_filialleiter.highlight.2',
        'rbac.template.role.handel_filialleiter.highlight.3',
      ],
      grants: merge(
        view(['dashboard', 'crm', 'kalender', 'zeiterfassung', 'schichten', 'inventar',
          'team', 'finance', 'berichte', 'notifications', 'settings', 'kommunikation',
          'documents']),
        // CRM — full
        keys([
          'crm:contact:read', 'crm:contact:create', 'crm:contact:edit',
          'crm:deal:read', 'crm:deal:create',
        ]),
        // Kalender
        keys(['kalender:booking_page:manage', 'kalender:category:manage']),
        // Zeiterfassung — team management
        keys([
          'zeiterfassung:team:view', 'zeiterfassung:week:approve',
          'zeiterfassung:corrections:approve', 'zeiterfassung:export:run',
        ]),
        // Schichten — full management
        keys([
          'schichten:shift:read', 'schichten:shift:publish',
          'schichten:assignment:manage', 'schichten:template:read',
          'schichten:template:manage', 'schichten:swap:read', 'schichten:swap:approve',
          'schichten:export:run',
        ]),
        // Inventar — full
        keys([
          'inventar:item:read', 'inventar:item:create', 'inventar:item:edit',
          'inventar:location:read', 'inventar:movement:read', 'inventar:movement:create',
          'inventar:movement:adjust', 'inventar:inventur:read', 'inventar:inventur:create',
          'inventar:inventur:count', 'inventar:inventur:book', 'inventar:export:run',
        ]),
        // Team — read + directory (no HR data)
        keys(['team:employee:read', 'team:absence:read', 'team:self:propose',
          'team:training:manage', 'team:onboarding:manage']),
        keys(['team:directory:full']),
        // Finance — VIEW LEVEL ONLY (no export/send/dunning/settings)
        keys(['finance:invoice:read', 'finance:quote:read', 'finance:amounts:view']),
        // Berichte — read
        keys(['berichte:reports:read']),
        // Documents
        keys(['documents:file:read', 'documents:file:download', 'documents:file:upload']),
        // Settings personal
        keys(['settings:personal:manage']),
      ),
    },

    // Role 10: Verkauf / Kasse (basedOn member)
    // zeiterfassung own, schichten own + swap
    // crm contact read+create, kalender own, inventar item:read
    // VERBORGEN: einkauf, finance, berichte, HR
    {
      id: 'handel_verkauf',
      labelKey: 'rbac.template.role.handel_verkauf.label',
      descriptionKey: 'rbac.template.role.handel_verkauf.description',
      basedOn: 'member',
      color: 'hsl(190 77% 40%)',
      highlightKeys: [
        'rbac.template.role.handel_verkauf.highlight.1',
        'rbac.template.role.handel_verkauf.highlight.2',
        'rbac.template.role.handel_verkauf.highlight.3',
      ],
      grants: merge(
        view(['dashboard', 'zeiterfassung', 'schichten', 'crm', 'kalender',
          'inventar', 'notifications', 'settings']),
        // Schichten — own + swap
        keys(['schichten:shift:read', 'schichten:swap:read']),
        keys(['schichten:swap:create'], 'own'),
        // CRM — contact read + create
        keys(['crm:contact:read', 'crm:contact:create']),
        // Inventar — item read only
        keys(['inventar:item:read']),
        // Settings personal
        keys(['settings:personal:manage']),
      ),
    },

    // Role 11: Lager / Logistik (basedOn member)
    // inventar VOLL operativ (movement:create/adjust, inventur count/book, item read; item:edit NICHT)
    // zeiterfassung own, schichten own
    // einkauf NUR po:receive + po:read
    // VERBORGEN: crm, finance, berichte, HR
    {
      id: 'handel_lager',
      labelKey: 'rbac.template.role.handel_lager.label',
      descriptionKey: 'rbac.template.role.handel_lager.description',
      basedOn: 'member',
      color: 'hsl(38 92% 50%)',
      highlightKeys: [
        'rbac.template.role.handel_lager.highlight.1',
        'rbac.template.role.handel_lager.highlight.2',
        'rbac.template.role.handel_lager.highlight.3',
      ],
      grants: merge(
        view(['dashboard', 'inventar', 'zeiterfassung', 'schichten', 'einkauf',
          'notifications', 'settings', 'documents']),
        // Inventar — operativ ohne item:edit und item:delete
        keys([
          'inventar:item:read', 'inventar:location:read',
          'inventar:movement:read', 'inventar:movement:create', 'inventar:movement:adjust',
          'inventar:inventur:read', 'inventar:inventur:create',
          'inventar:inventur:count', 'inventar:inventur:book',
          'inventar:export:run',
        ]),
        // Schichten — own (no swap: Lager often has fixed shifts)
        keys(['schichten:shift:read']),
        // Einkauf — receive + read only
        keys(['einkauf:po:read', 'einkauf:po:receive', 'einkauf:catalog:read']),
        // Documents read
        keys(['documents:file:read', 'documents:file:download']),
        // Settings personal
        keys(['settings:personal:manage']),
      ),
    },

    // Role 12: Einkauf (basedOn member)
    // einkauf VOLL, inventar VOLL inkl. item:edit, vertraege, documents
    // crm read (Lieferanten), berichte:reports:read
    // VERBORGEN: finance, HR, dialer
    {
      id: 'handel_einkauf',
      labelKey: 'rbac.template.role.handel_einkauf.label',
      descriptionKey: 'rbac.template.role.handel_einkauf.description',
      basedOn: 'member',
      color: 'hsl(25 95% 53%)',
      highlightKeys: [
        'rbac.template.role.handel_einkauf.highlight.1',
        'rbac.template.role.handel_einkauf.highlight.2',
        'rbac.template.role.handel_einkauf.highlight.3',
      ],
      grants: merge(
        view(['dashboard', 'einkauf', 'inventar', 'vertraege', 'documents',
          'crm', 'berichte', 'notifications', 'settings', 'kalender']),
        // Einkauf — full
        keys([
          'einkauf:po:read', 'einkauf:po:create', 'einkauf:po:edit',
          'einkauf:po:send', 'einkauf:po:approve', 'einkauf:po:receive',
          'einkauf:po:cancel',
          'einkauf:supplier:read', 'einkauf:supplier:create', 'einkauf:supplier:edit',
          'einkauf:supplier:deactivate', 'einkauf:rating:create',
          'einkauf:catalog:read', 'einkauf:contract:read', 'einkauf:contract:call',
          'einkauf:export:run',
        ]),
        // Inventar — full inkl. item:edit (Stammdata pflege)
        keys([
          'inventar:item:read', 'inventar:item:create', 'inventar:item:edit',
          'inventar:location:read', 'inventar:movement:read', 'inventar:movement:create',
          'inventar:inventur:read', 'inventar:attachment:manage', 'inventar:export:run',
        ]),
        // Vertraege — full
        keys([
          'vertraege:contract:read', 'vertraege:contract:create',
          'vertraege:contract:edit', 'vertraege:contract:terminate',
        ]),
        // Documents — full
        keys([
          'documents:file:read', 'documents:file:download', 'documents:file:upload',
          'documents:file:edit', 'documents:share:manage',
        ]),
        // CRM — contact + supplier read (Lieferanten)
        keys(['crm:contact:read']),
        // Berichte — read
        keys(['berichte:reports:read']),
        // Settings personal
        keys(['settings:personal:manage']),
      ),
    },
  ],
}

// ── Exported sets in canonical order ────────────────────────────────────────

export const INDUSTRY_ROLE_SETS: IndustryRoleSet[] = [
  SET_HANDWERK,
  SET_DIENSTLEISTER,
  SET_HANDEL,
]

/**
 * Returns the sets ordered so that the one most relevant for the tenant's
 * business profile comes first. Falls back to canonical order when no match.
 */
export function orderedSetsForProfile(profileId: BusinessProfileId | null): IndustryRoleSet[] {
  if (!profileId) return INDUSTRY_ROLE_SETS
  const primary = INDUSTRY_ROLE_SETS.find((s) =>
    s.businessProfileIds.includes(profileId),
  )
  if (!primary) return INDUSTRY_ROLE_SETS
  return [primary, ...INDUSTRY_ROLE_SETS.filter((s) => s.id !== primary.id)]
}

/** Flat lookup by template id across all sets. */
export function findTemplate(templateId: string): IndustryRoleTemplate | undefined {
  for (const set of INDUSTRY_ROLE_SETS) {
    const found = set.roles.find((r) => r.id === templateId)
    if (found) return found
  }
  return undefined
}

// Re-export for convenience (modules that only need the type)
export type { GrantSpec }
