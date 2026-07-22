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
    valueSetIds: ['ticket_priority'],
    fieldEntities: ['helpdesk_ticket'],
    Component: lazy(() => import('@/modules/helpdesk/HelpdeskPage')) as EditorModuleDef['Component'],
  },
]

export function getEditorModule(key: string): EditorModuleDef | undefined {
  return EDITOR_MODULES.find((m) => m.key === key)
}
