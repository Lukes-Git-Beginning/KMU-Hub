/**
 * Module display metadata for the licensing tab — reuses the existing nav-item
 * i18n labels (one source of truth) so module names stay consistent app-wide.
 * Only `crm` lacks a nav entry, so it gets a dedicated key.
 */
import type { ModuleGroupId } from '@/api/admin-types'

/** moduleId → i18n label key. */
export const MODULE_LABEL_KEY: Record<string, string> = {
  crm: 'admin.license.module.crm',
  tasks: 'layout.navItems.tasks',
  finance: 'layout.navItems.finance',
  calendar: 'layout.navItems.calendar',
  documents: 'layout.navItems.documents',
  chat: 'layout.navItems.kommunikation',
  meetings: 'layout.navItems.meetings',
  mail: 'layout.navItems.mail',
  dialer: 'layout.navItems.dialer',
  team: 'layout.navItems.team',
  zeiterfassung: 'layout.navItems.zeiterfassung',
  schichten: 'layout.navItems.schichten',
  projects: 'layout.navItems.projects',
  inventar: 'layout.navItems.inventar',
  einkauf: 'layout.navItems.einkauf',
  helpdesk: 'layout.navItems.helpdesk',
  fuhrpark: 'layout.navItems.fuhrpark',
  vertraege: 'layout.navItems.vertraege',
  produktion: 'layout.navItems.produktion',
  vermietung: 'layout.navItems.vermietung',
  berichte: 'layout.navItems.berichte',
  formulare: 'layout.navItems.formulare',
  wiki: 'layout.navItems.wiki',
  rapporte: 'layout.navItems.rapporte',
}

export const MODULE_GROUP_ORDER: ModuleGroupId[] = ['core', 'comm', 'team', 'industry', 'tools']

export function moduleLabelKey(moduleId: string): string {
  return MODULE_LABEL_KEY[moduleId] ?? moduleId
}
