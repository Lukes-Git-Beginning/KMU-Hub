import type { ComponentType } from 'react'
import {
  Contact,
  Receipt,
  Calendar,
  Mail,
  MessagesSquare,
  Users,
  Building2,
  CreditCard,
  Plug,
  Monitor,
  Package,
  KanbanSquare,
  FolderOpen,
  FileSignature,
  LayoutDashboard,
  Clock,
  LifeBuoy,
  Workflow,
  BarChart3,
  Bell,
  FileInput,
  BookOpen,
  PhoneCall,
  Video,
  ShieldCheck,
  Palette,
  Warehouse,
  ClipboardCheck,
  type LucideIcon,
} from 'lucide-react'
import type { RoleId } from '@/config/roles'
import { CrmSettingsPanel } from './panels/CrmSettingsPanel'
import { FinanceSettingsPanel } from './panels/FinanceSettingsPanel'
import { CalendarSettingsTab } from './tabs/CalendarSettingsTab'
import { MailsSettingsPanel } from '@/modules/mails/settings/MailsSettingsPanel'
import { KommunikationSettingsPanel } from '@/modules/kommunikation/KommunikationSettingsPanel'
import { TeamSettingsPanel } from './panels/TeamSettingsPanel'
import { WorkSettingsPanel } from './panels/WorkSettingsPanel'
import { CompanySettingsTab } from './tabs/CompanySettingsTab'
import { BillingSettingsTab } from './tabs/BillingSettingsTab'
import { IntegrationSettingsTab } from './tabs/IntegrationSettingsTab'
import { ITAdminTab } from './tabs/ITAdminTab'
import { ModuleAssignmentSettingsPanel } from './panels/ModuleAssignmentSettingsPanel'
import { DokumenteSettingsPanel } from '@/modules/dokumente/settings/DokumenteSettingsPanel'
import { VertraegeSettingsPanel } from '@/modules/vertraege/settings/VertraegeSettingsPanel'
import { ZeiterfassungSettingsPanel } from '@/modules/zeiterfassung/settings/ZeiterfassungSettingsPanel'
import { DashboardSettingsPanel } from '@/modules/dashboard/settings/DashboardSettingsPanel'
import { HelpdeskSettingsPanel } from '@/modules/helpdesk/settings/HelpdeskSettingsPanel'
import { AutomatisierungSettingsPanel } from '@/modules/automatisierung/settings/AutomatisierungSettingsPanel'
import { BerichteSettingsPanel } from '@/modules/berichte/settings/BerichteSettingsPanel'
import { NotificationsSettingsPanel } from './panels/NotificationsSettingsPanel'
import { FormulareSettingsPanel } from '@/modules/formulare/settings/FormulareSettingsPanel'
import { WikiSettingsPanel } from '@/modules/wiki/settings/WikiSettingsPanel'
import { DialerSettingsPanel } from '@/modules/dialer/settings/DialerSettingsPanel'
import { VideoSettingsPanel } from '@/modules/video/settings/VideoSettingsPanel'
import { InventarSettingsPanel } from '@/modules/inventar/settings/InventarSettingsPanel'
import { VermietungSettingsPanel } from '@/modules/vermietung/settings/VermietungSettingsPanel'
import { RapporteSettingsPanel } from '@/modules/rapporte/settings/RapporteSettingsPanel'
import { SecuritySettingsPanel } from './panels/SecuritySettingsPanel'
// admin-batch (parallel/admin): tenant branding surfaced in the settings overlay
import BrandingAdminHubTab from '@/modules/admin/branding/BrandingAdminHubTab'

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
  { id: 'dialer', group: 'module', labelKey: 'moduleSettings.entries.dialer', icon: PhoneCall, navMatch: ['/dialer'], component: DialerSettingsPanel },
  { id: 'video', group: 'module', labelKey: 'moduleSettings.entries.video', icon: Video, navMatch: ['/video'], component: VideoSettingsPanel },
  { id: 'finance', group: 'module', labelKey: 'moduleSettings.entries.finance', icon: Receipt, navMatch: ['/finanzen', '/buchhaltung'], component: FinanceSettingsPanel },
  { id: 'calendar', group: 'module', labelKey: 'moduleSettings.entries.calendar', icon: Calendar, navMatch: ['/kalender'], component: CalendarSettingsTab },
  { id: 'mail', group: 'module', labelKey: 'moduleSettings.entries.mail', icon: Mail, navMatch: ['/mails'], component: MailsSettingsPanel },
  { id: 'kommunikation', group: 'module', labelKey: 'moduleSettings.entries.kommunikation', icon: MessagesSquare, navMatch: ['/kommunikation', '/chat'], component: KommunikationSettingsPanel },
  { id: 'team', group: 'module', labelKey: 'moduleSettings.entries.team', icon: Users, navMatch: ['/team'], roles: ['admin', 'hr'], component: TeamSettingsPanel },
  { id: 'work', group: 'module', labelKey: 'moduleSettings.entries.work', icon: KanbanSquare, navMatch: ['/work'], component: WorkSettingsPanel },
  { id: 'automatisierung', group: 'module', labelKey: 'moduleSettings.entries.automatisierung', icon: Workflow, navMatch: ['/automatisierung'], component: AutomatisierungSettingsPanel },
  { id: 'helpdesk', group: 'module', labelKey: 'moduleSettings.entries.helpdesk', icon: LifeBuoy, navMatch: ['/helpdesk'], component: HelpdeskSettingsPanel },
  { id: 'berichte', group: 'module', labelKey: 'moduleSettings.entries.berichte', icon: BarChart3, navMatch: ['/berichte'], component: BerichteSettingsPanel },
  { id: 'zeiterfassung', group: 'module', labelKey: 'moduleSettings.entries.zeiterfassung', icon: Clock, navMatch: ['/zeiterfassung'], component: ZeiterfassungSettingsPanel },
  { id: 'dokumente', group: 'module', labelKey: 'moduleSettings.entries.dokumente', icon: FolderOpen, navMatch: ['/dokumente'], component: DokumenteSettingsPanel },
  { id: 'vertraege', group: 'module', labelKey: 'moduleSettings.entries.vertraege', icon: FileSignature, navMatch: ['/vertraege'], component: VertraegeSettingsPanel },
  { id: 'notifications', group: 'module', labelKey: 'moduleSettings.entries.notifications', icon: Bell, navMatch: ['/notifications'], component: NotificationsSettingsPanel },
  { id: 'formulare', group: 'module', labelKey: 'moduleSettings.entries.formulare', icon: FileInput, navMatch: ['/formulare'], component: FormulareSettingsPanel },
  { id: 'wiki', group: 'module', labelKey: 'moduleSettings.entries.wiki', icon: BookOpen, navMatch: ['/wiki'], component: WikiSettingsPanel },
  // branchen-block: inventar is the pilot — the remaining Branchen modules follow the same pattern
  { id: 'inventar', group: 'module', labelKey: 'moduleSettings.entries.inventar', icon: Warehouse, navMatch: ['/inventar'], component: InventarSettingsPanel },
  { id: 'vermietung', group: 'module', labelKey: 'moduleSettings.entries.vermietung', icon: Building2, navMatch: ['/vermietung'], component: VermietungSettingsPanel },
  { id: 'rapporte', group: 'module', labelKey: 'moduleSettings.entries.rapporte', icon: ClipboardCheck, navMatch: ['/rapporte'], component: RapporteSettingsPanel },
  // '/' is exact-match only in the resolver — keep this entry from swallowing other routes.
  { id: 'dashboard', group: 'module', labelKey: 'moduleSettings.entries.dashboard', icon: LayoutDashboard, navMatch: ['/'], component: DashboardSettingsPanel },

  // ── COSMI (Allgemein) ──
  { id: 'company', group: 'cosmi', labelKey: 'moduleSettings.entries.company', icon: Building2, roles: ['admin'], component: CompanySettingsTab },
  { id: 'billing', group: 'cosmi', labelKey: 'moduleSettings.entries.billing', icon: CreditCard, roles: ['admin'], component: BillingSettingsTab },
  { id: 'modulzuteilung', group: 'cosmi', labelKey: 'moduleSettings.entries.moduleAssignment', icon: Package, roles: ['admin', 'it_support'], component: ModuleAssignmentSettingsPanel },
  { id: 'integrations', group: 'cosmi', labelKey: 'moduleSettings.entries.integrations', icon: Plug, roles: ['admin'], component: IntegrationSettingsTab },
  { id: 'it', group: 'cosmi', labelKey: 'moduleSettings.entries.it', icon: Monitor, roles: ['admin', 'it_support'], component: ITAdminTab },
  { id: 'security', group: 'cosmi', labelKey: 'moduleSettings.entries.security', icon: ShieldCheck, roles: ['admin', 'it_support'], navMatch: ['/admin/security'], component: SecuritySettingsPanel },
  // admin-batch (parallel/admin) — additive entry; new tenant branding surface
  { id: 'branding', group: 'cosmi', labelKey: 'moduleSettings.entries.branding', icon: Palette, roles: ['admin'], navMatch: ['/admin/branding'], component: BrandingAdminHubTab },
]

/** Resolve the settings entry id that matches the current route (context preselect). */
export function resolveEntryForPath(pathname: string): string | undefined {
  const match = SETTINGS_ENTRIES.find((e) =>
    e.navMatch?.some((p) => (p === '/' ? pathname === '/' : pathname.startsWith(p))),
  )
  return match?.id
}
