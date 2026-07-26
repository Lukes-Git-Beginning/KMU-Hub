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
    // CRM sub-tabs are router-based (blocked in the editor) → area toggles deferred
    // until the router-sandbox path is built (see MODUL-AUDIT).
    areas: [],
    Component: lazy(() => import('@/modules/kontakte/KontaktePage')) as EditorModuleDef['Component'],
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
    // is locked: it has no real data source until the CSAT survey feature ships.
    statWidgets: [
      { key: 'openTickets', labelKey: 'helpdesk.stats.openTickets' },
      { key: 'avgResponseTime', labelKey: 'helpdesk.stats.avgResponseTime' },
      { key: 'resolvedThisWeek', labelKey: 'helpdesk.stats.resolvedThisWeek' },
      { key: 'csat', labelKey: 'helpdesk.stats.customerSatisfaction', locked: true },
      { key: 'ticketsPerDay', labelKey: 'helpdesk.stats.ticketsPerDay' },
      { key: 'csatChart', labelKey: 'customization.editor.statistik.csatChartLabel', locked: true },
      { key: 'byStatus', labelKey: 'helpdesk.stats.byStatus' },
      { key: 'byPriority', labelKey: 'helpdesk.stats.byPriority' },
    ],
    Component: lazy(() => import('@/modules/helpdesk/HelpdeskPage')) as EditorModuleDef['Component'],
  },
]

export function getEditorModule(key: string): EditorModuleDef | undefined {
  return EDITOR_MODULES.find((m) => m.key === key)
}
