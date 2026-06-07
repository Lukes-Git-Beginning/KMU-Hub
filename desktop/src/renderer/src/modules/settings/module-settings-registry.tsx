import type { ComponentType } from 'react'
import {
  Contact,
  Receipt,
  Calendar,
  Mail,
  Users,
  Building2,
  CreditCard,
  Plug,
  Monitor,
  type LucideIcon,
} from 'lucide-react'
import type { RoleId } from '@/config/roles'
import { CrmSettingsPanel } from './panels/CrmSettingsPanel'
import { FinanceSettingsPanel } from './panels/FinanceSettingsPanel'
import { CalendarSettingsTab } from './tabs/CalendarSettingsTab'
import { MailSettingsTab } from './tabs/MailSettingsTab'
import { TeamSettingsTab } from './tabs/TeamSettingsTab'
import { CompanySettingsTab } from './tabs/CompanySettingsTab'
import { BillingSettingsTab } from './tabs/BillingSettingsTab'
import { IntegrationSettingsTab } from './tabs/IntegrationSettingsTab'
import { ITAdminTab } from './tabs/ITAdminTab'

/**
 * Registry for the Module-Settings overlay (opened from the bottom-left
 * "Einstellungen" button). Two groups:
 *  - 'module': per-module settings (context-preselected by active route)
 *  - 'cosmi':  cross-cutting / organisation-wide settings (formerly Admin)
 *
 * Personal settings (profile, appearance, language, security, notifications)
 * deliberately live elsewhere — the profile menu (top-right) → /settings.
 */
export type SettingsGroup = 'module' | 'cosmi'

export interface SettingsEntry {
  id: string
  group: SettingsGroup
  labelKey: string
  icon: LucideIcon
  /** Route prefixes that map to this entry (for context preselect). */
  navMatch?: string[]
  /** RBAC: visible only to these roles (undefined = everyone). */
  roles?: RoleId[]
  component: ComponentType
}

export const SETTINGS_ENTRIES: SettingsEntry[] = [
  // ── MODULE ──
  { id: 'crm', group: 'module', labelKey: 'moduleSettings.entries.crm', icon: Contact, navMatch: ['/kontakte', '/crm'], component: CrmSettingsPanel },
  { id: 'finance', group: 'module', labelKey: 'moduleSettings.entries.finance', icon: Receipt, navMatch: ['/finanzen', '/buchhaltung'], component: FinanceSettingsPanel },
  { id: 'calendar', group: 'module', labelKey: 'moduleSettings.entries.calendar', icon: Calendar, navMatch: ['/kalender'], component: CalendarSettingsTab },
  { id: 'mail', group: 'module', labelKey: 'moduleSettings.entries.mail', icon: Mail, navMatch: ['/mails'], component: MailSettingsTab },
  { id: 'team', group: 'module', labelKey: 'moduleSettings.entries.team', icon: Users, navMatch: ['/team'], roles: ['admin', 'hr'], component: TeamSettingsTab },

  // ── COSMI (Allgemein) ──
  { id: 'company', group: 'cosmi', labelKey: 'moduleSettings.entries.company', icon: Building2, roles: ['admin'], component: CompanySettingsTab },
  { id: 'billing', group: 'cosmi', labelKey: 'moduleSettings.entries.billing', icon: CreditCard, roles: ['admin'], component: BillingSettingsTab },
  { id: 'integrations', group: 'cosmi', labelKey: 'moduleSettings.entries.integrations', icon: Plug, roles: ['admin'], component: IntegrationSettingsTab },
  { id: 'it', group: 'cosmi', labelKey: 'moduleSettings.entries.it', icon: Monitor, roles: ['admin', 'it_support'], component: ITAdminTab },
]

/** Resolve the settings entry id that matches the current route (context preselect). */
export function resolveEntryForPath(pathname: string): string | undefined {
  const match = SETTINGS_ENTRIES.find((e) => e.navMatch?.some((p) => pathname.startsWith(p)))
  return match?.id
}
