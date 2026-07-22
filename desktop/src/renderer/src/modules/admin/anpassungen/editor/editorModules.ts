/**
 * Editor module registry (Modul-Editor v1) — the modules that can be opened in
 * the customization editor. v1 pilot = Kontakte + Helpdesk (KONZEPT §0 ④); the
 * gallery in E-4 renders from this list and the shell in E-2 boots the picked
 * module into the sandbox canvas.
 *
 * titleKey reuses a LABEL_WHITELIST key so the editor title itself reflects the
 * tenant's terminology (e.g. "Patienten" instead of "CRM").
 */
import { lazy } from 'react'
import type { ComponentType, LazyExoticComponent } from 'react'

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
   * LABEL_WHITELIST keys this module exposes in the Begriffe editor. The first
   * key is the module title (wired into the module header for live preview).
   */
  labelKeys: string[]
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
      'rbac.module.crm',
      'layout.navItems.contacts',
      'crm.contacts.title',
      'crm.companies.title',
      'crm.deals.title',
    ],
    Component: lazy(() => import('@/modules/kontakte/KontaktePage')) as EditorModuleDef['Component'],
  },
  {
    key: 'helpdesk',
    titleKey: 'rbac.module.helpdesk',
    previewPath: '/helpdesk',
    icon: 'lifeBuoy',
    labelKeys: ['rbac.module.helpdesk', 'layout.navItems.helpdesk'],
    Component: lazy(() => import('@/modules/helpdesk/HelpdeskPage')) as EditorModuleDef['Component'],
  },
]

export function getEditorModule(key: string): EditorModuleDef | undefined {
  return EDITOR_MODULES.find((m) => m.key === key)
}
