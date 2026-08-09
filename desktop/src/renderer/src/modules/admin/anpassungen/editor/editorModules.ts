/**
 * Editor module registry (Modul-Editor v1) — the modules that can be opened in
 * the customization editor. v1 pilot = Kontakte + Helpdesk (KONZEPT §0 ④); the
 * gallery in E-4 renders from this list and the shell in E-2 boots the picked
 * module into the sandbox canvas.
 *
 * titleKey is a FIXED module-name key (rbac.module.*) — module names are
 * immutable (Darien 2026-07-22), so the editor title always shows the stable
 * name ("Kontakte bearbeiten"), never a tenant-renamed one. Only the CONTENT
 * keys in labelKeys are customizable.
 */
import { lazy } from 'react'
import type { ComponentType, LazyExoticComponent } from 'react'
import type { CustomFieldEntity } from '@/mocks/data/custom-fields'

export interface EditorModuleDef {
  /** Stable module key used in routes/drafts. */
  key: string
  /** i18n key for the display name (a LABEL_WHITELIST key → customization-aware). */
  titleKey: string
  /** Route the sandbox MemoryRouter boots at. */
  previewPath: string
  /** Lucide icon name rendered on the gallery tile (resolved in the gallery). */
  icon: 'contact' | 'lifeBuoy'
  /**
   * LABEL_WHITELIST keys this module exposes in the Begriffe editor — CONTENT
   * headings only (object/record nouns), never the module name (that's fixed).
   */
  labelKeys: string[]
  /** Value-set ids this module exposes in the Wertelisten editor. */
  valueSetIds: string[]
  /** Custom-field entities this module exposes in the Felder editor (E-3c). */
  fieldEntities: CustomFieldEntity[]
  /**
   * Toggleable sub-areas (tabs/sections) this module exposes in the Bereiche
   * editor (R4). `key` matches the module's own area/tab key; `labelKey` is an
   * i18n key for the display name. Empty → module has no on/off areas.
   */
  areas: { key: string; labelKey: string }[]
  /**
   * Statistics widgets this module's stats view exposes in the Statistik editor.
   * Each toggles visibility via moduleAreas under a `stat:` prefix (so it reuses
   * the areas draft/resolve/deploy machinery). `locked` widgets need a feature
   * that isn't built yet (e.g. CSAT) → shown greyed in the catalog, hidden in the
   * module until the feature ships.
   */
  statWidgets?: { key: string; labelKey: string; locked?: boolean }[]
  /**
   * Built-in columns of this module's list view, toggleable in the Spalten editor
   * (Darien 2026-08-04) via moduleAreas under a `col:` prefix. These default to
   * visible; every custom field additionally offers an opt-in column, derived at
   * runtime rather than listed here. Empty → module has no configurable list.
   */
  listColumns?: {
    key: string
    labelKey: string
    /**
     * The value list this built-in column already renders. Without it the Spalten
     * panel would offer that list under "Wertelisten ohne Spalte" even though it
     * has had a column all along (priority/status).
     */
    valueSetId?: string
  }[]
  /**
   * Ticket-Intake P6 — this module has configurable creation channels (agent /
   * self-service / external) shown in the editor's Kanäle panel. Only modules
   * with an intake target set this (helpdesk first). Undefined → no Kanäle tab.
   */
  intake?: boolean
  /** The module page rendered read-only in the sandbox canvas. */
  Component: LazyExoticComponent<ComponentType<unknown>>
}

export const EDITOR_MODULES: EditorModuleDef[] = [
  {
    key: 'kontakte',
    titleKey: 'rbac.module.crm',
    previewPath: '/kontakte',
    icon: 'contact',
    labelKeys: [
      'crm.contacts.title',
      'crm.companies.title',
      'crm.deals.title',
    ],
    valueSetIds: ['deal_stages'],
    fieldEntities: ['crm_contact', 'crm_company', 'crm_deal', 'crm_activity'],
    // Sub-areas were router-based until 2026-08-10, which is why this used to be
    // empty — the sandbox has no matching URL and may not open its own router.
    // KontakteLayout now drives them by state (routes remain as entry points), so
    // they toggle like any other module's tabs. `key` matches KontakteSection.
    areas: [
      { key: 'kontakte', labelKey: 'crm.nav.contacts' },
      { key: 'leads', labelKey: 'crm.nav.leads' },
      { key: 'firmen', labelKey: 'crm.nav.companies' },
      { key: 'pipeline', labelKey: 'crm.nav.deals' },
      { key: 'aktivitaeten', labelKey: 'crm.nav.activities' },
      { key: 'auswertungen', labelKey: 'crm.nav.reports' },
    ],
    // The layout, not the contacts list — otherwise the preview shows a module
    // without its own navigation and the area toggles have nothing to act on.
    Component: lazy(() => import('@/modules/kontakte/KontakteLayout')) as EditorModuleDef['Component'],
  },
  {
    key: 'helpdesk',
    titleKey: 'rbac.module.helpdesk',
    previewPath: '/helpdesk',
    icon: 'lifeBuoy',
    // No customizable CONTENT heading whitelisted for helpdesk yet — its editor
    // value in v1 is Wertelisten (priorities) + Felder. Module name stays fixed.
    labelKeys: [],
    valueSetIds: ['ticket_priority', 'ticket_status'],
    fieldEntities: ['helpdesk_ticket'],
    // State-based tabs → toggleable in the editor (R4).
    areas: [
      { key: 'tickets', labelKey: 'helpdesk.tabs.ticketsLabel' },
      { key: 'wissensdatenbank', labelKey: 'helpdesk.tabs.knowledgeBase' },
      { key: 'statistik', labelKey: 'helpdesk.tabs.statistics' },
    ],
    // Stats-view widgets, toggleable in the Statistik editor. CSAT (kachel + chart)
    // reads real ratings off the wire (intake P3) → no longer locked.
    statWidgets: [
      { key: 'openTickets', labelKey: 'helpdesk.stats.openTickets' },
      { key: 'avgResponseTime', labelKey: 'helpdesk.stats.avgResponseTime' },
      { key: 'resolvedThisWeek', labelKey: 'helpdesk.stats.resolvedThisWeek' },
      { key: 'csat', labelKey: 'helpdesk.stats.customerSatisfaction' },
      { key: 'ticketsPerDay', labelKey: 'helpdesk.stats.ticketsPerDay' },
      { key: 'csatChart', labelKey: 'customization.editor.statistik.csatChartLabel' },
      { key: 'byStatus', labelKey: 'helpdesk.stats.byStatus' },
      { key: 'byPriority', labelKey: 'helpdesk.stats.byPriority' },
    ],
    // Ticket list columns — what is readable without opening a ticket.
    listColumns: [
      { key: 'ticketNr', labelKey: 'helpdesk.table.ticketNr' },
      { key: 'subject', labelKey: 'helpdesk.table.subject' },
      { key: 'category', labelKey: 'helpdesk.table.category' },
      { key: 'priority', labelKey: 'helpdesk.table.priority', valueSetId: 'ticket_priority' },
      // Own key, not the shared `common.status`: renaming the column must stay in
      // the helpdesk instead of retitling every status label in the app.
      { key: 'status', labelKey: 'helpdesk.table.status', valueSetId: 'ticket_status' },
      { key: 'assignedTo', labelKey: 'helpdesk.table.assignedTo' },
      { key: 'sla', labelKey: 'helpdesk.table.sla' },
      { key: 'createdAt', labelKey: 'helpdesk.table.createdAt' },
    ],
    // Ticket-Intake P6 — helpdesk has the three creation channels (agent /
    // self-service / external), configured in the editor's Kanäle panel.
    intake: true,
    Component: lazy(() => import('@/modules/helpdesk/HelpdeskPage')) as EditorModuleDef['Component'],
  },
]

export function getEditorModule(key: string): EditorModuleDef | undefined {
  return EDITOR_MODULES.find((m) => m.key === key)
}
