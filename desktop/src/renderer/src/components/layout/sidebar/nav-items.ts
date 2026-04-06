import i18next from 'i18next'
import {
  LayoutDashboard,
  FolderKanban,
  ListChecks,
  FileText,
  Receipt,
  MessageSquare,
  MessageSquareText,
  Users,
  Mail,
  Contact,
  Calendar,
  Cog,
  Video,
  Network,
  Warehouse,
  CalendarClock,
  ShoppingCart,
  LifeBuoy,
  Truck,
  Factory,
  BarChart3,
  FileSignature,
  FileInput,
  Building2,
  ClipboardCheck,
  Timer,
  ShieldCheck,
  BookOpen,
} from 'lucide-react'

export interface NavBadge {
  type: 'text' | 'live'
  value?: string
}

export interface NavItemColor {
  /** Hue (0-360) */
  h: number
  /** Saturation (0-100) */
  s: number
}

export interface NavItemConfig {
  id: string
  to: string
  icon: typeof LayoutDashboard
  label: string
  badge?: NavBadge
  enabled: boolean
  section: 'main' | 'bottom'
  color: NavItemColor
}

/** Look up a nav item's color by module ID. Returns a default gray if not found. */
export function getModuleColor(moduleId: string): NavItemColor {
  const item = navItems.find((i) => i.id === moduleId)
  return item?.color ?? { h: 220, s: 18 }
}

/** Convert a NavItemColor to an HSL color string at a given lightness. */
export function moduleHsl(moduleId: string, lightness = 42): string {
  const c = getModuleColor(moduleId)
  return `hsl(${c.h} ${c.s}% ${lightness}%)`
}

/** Convert a NavItemColor to an HSL background string (10% opacity feel via high lightness). */
export function moduleHslBg(moduleId: string): string {
  const c = getModuleColor(moduleId)
  return `hsl(${c.h} ${c.s}% 95%)`
}

/** Dark mode variant of module background. */
export function moduleHslBgDark(moduleId: string): string {
  const c = getModuleColor(moduleId)
  return `hsl(${c.h} ${Math.round(c.s * 0.6)}% 18%)`
}

export const navItems: NavItemConfig[] = [
  // ── Core ──
  { id: 'dashboard', to: '/', icon: LayoutDashboard, label: i18next.t('layout.navItems.dashboard'), enabled: true, section: 'main', color: { h: 240, s: 72 } },
  { id: 'projects', to: '/work/projects', icon: FolderKanban, label: i18next.t('layout.navItems.projects'), enabled: true, section: 'main', color: { h: 255, s: 68 } },
  { id: 'tasks', to: '/work/my-tasks', icon: ListChecks, label: i18next.t('layout.navItems.tasks'), enabled: true, section: 'main', badge: { type: 'text', value: '5' }, color: { h: 217, s: 78 } },
  { id: 'chat', to: '/chat', icon: MessageSquare, label: i18next.t('layout.navItems.chat'), enabled: true, section: 'main', badge: { type: 'text', value: '3' }, color: { h: 195, s: 82 } },
  { id: 'contacts', to: '/kontakte', icon: Contact, label: i18next.t('layout.navItems.contacts'), enabled: true, section: 'main', color: { h: 162, s: 68 } },

  // ── Team & HR ──
  { id: 'team', to: '/team', icon: Users, label: i18next.t('layout.navItems.team'), enabled: true, section: 'main', color: { h: 271, s: 68 } },

  // ── Communication ──
  { id: 'meetings', to: '/meetings', icon: Video, label: i18next.t('layout.navItems.meetings'), enabled: true, section: 'main', badge: { type: 'live' }, color: { h: 338, s: 72 } },
  { id: 'calendar', to: '/kalender', icon: Calendar, label: i18next.t('layout.navItems.calendar'), enabled: true, section: 'main', color: { h: 25, s: 82 } },
  { id: 'zeiterfassung', to: '/zeiterfassung', icon: Timer, label: i18next.t('layout.navItems.zeiterfassung'), enabled: true, section: 'main', color: { h: 280, s: 60 } },
  { id: 'documents', to: '/dokumente', icon: FileText, label: i18next.t('layout.navItems.documents'), enabled: true, section: 'main', color: { h: 152, s: 62 } },
  { id: 'wiki', to: '/wiki', icon: BookOpen, label: i18next.t('layout.navItems.wiki'), enabled: true, section: 'main', color: { h: 168, s: 58 } },
  { id: 'mail', to: '/mails', icon: Mail, label: i18next.t('layout.navItems.mail'), enabled: true, section: 'main', badge: { type: 'text', value: '12' }, color: { h: 205, s: 78 } },
  { id: 'kommunikation', to: '/kommunikation', icon: MessageSquareText, label: i18next.t('layout.navItems.kommunikation'), enabled: true, section: 'main', badge: { type: 'text', value: '' }, color: { h: 185, s: 72 } },

  // ── Finance ──
  { id: 'finance', to: '/finanzen', icon: Receipt, label: i18next.t('layout.navItems.finance'), enabled: true, section: 'main', badge: { type: 'text', value: i18next.t('layout.navItems.badgeNew') }, color: { h: 38, s: 88 } },
  { id: 'infrastructure', to: '/infrastruktur', icon: Network, label: i18next.t('layout.navItems.infrastructure'), enabled: true, section: 'main', color: { h: 215, s: 32 } },

  // ── Industry modules ──
  { id: 'inventar', to: '/inventar', icon: Warehouse, label: i18next.t('layout.navItems.inventar'), enabled: true, section: 'main', color: { h: 15, s: 75 } },
  { id: 'schichten', to: '/schichten', icon: CalendarClock, label: i18next.t('layout.navItems.schichten'), enabled: true, section: 'main', color: { h: 258, s: 62 } },
  { id: 'einkauf', to: '/einkauf', icon: ShoppingCart, label: i18next.t('layout.navItems.einkauf'), enabled: true, section: 'main', color: { h: 45, s: 82 } },
  { id: 'helpdesk', to: '/helpdesk', icon: LifeBuoy, label: i18next.t('layout.navItems.helpdesk'), enabled: true, section: 'main', color: { h: 308, s: 68 } },
  { id: 'fuhrpark', to: '/fuhrpark', icon: Truck, label: i18next.t('layout.navItems.fuhrpark'), enabled: true, section: 'main', color: { h: 348, s: 72 } },
  { id: 'produktion', to: '/produktion', icon: Factory, label: i18next.t('layout.navItems.produktion'), enabled: true, section: 'main', color: { h: 130, s: 58 } },
  { id: 'berichte', to: '/berichte', icon: BarChart3, label: i18next.t('layout.navItems.berichte'), enabled: true, section: 'main', color: { h: 42, s: 78 } },
  { id: 'vertraege', to: '/vertraege', icon: FileSignature, label: i18next.t('layout.navItems.vertraege'), enabled: true, section: 'main', color: { h: 155, s: 62 } },
  { id: 'formulare', to: '/formulare', icon: FileInput, label: i18next.t('layout.navItems.formulare'), enabled: true, section: 'main', color: { h: 222, s: 68 } },
  { id: 'vermietung', to: '/vermietung', icon: Building2, label: i18next.t('layout.navItems.vermietung'), enabled: true, section: 'main', color: { h: 8, s: 72 } },
  { id: 'rapporte', to: '/rapporte', icon: ClipboardCheck, label: i18next.t('layout.navItems.rapporte'), enabled: true, section: 'main', color: { h: 98, s: 55 } },

  // ── System (bottom) ──
  { id: 'security-admin', to: '/admin/security', icon: ShieldCheck, label: i18next.t('layout.navItems.securityAdmin'), enabled: true, section: 'bottom', color: { h: 0, s: 68 } },
  { id: 'settings', to: '/settings', icon: Cog, label: i18next.t('layout.navItems.settings'), enabled: true, section: 'bottom', color: { h: 220, s: 18 } },
]
