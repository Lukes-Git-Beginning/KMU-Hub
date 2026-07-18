import {
  LayoutDashboard,
  FolderKanban,
  ListChecks,
  FileText,
  Receipt,
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
  BookOpen,
  PhoneCall,
  Workflow,
  Bell,
  ShieldCheck,
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
  /**
   * Optional click override. When set, renderers call this instead of
   * navigating to `to` (used to open the settings overlay rather than route).
   */
  onActivate?: () => void
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
  { id: 'dashboard', to: '/', icon: LayoutDashboard, label: 'layout.navItems.dashboard', enabled: true, section: 'main', color: { h: 240, s: 72 } },
  { id: 'projects', to: '/work/projects', icon: FolderKanban, label: 'layout.navItems.projects', enabled: true, section: 'main', color: { h: 255, s: 68 } },
  { id: 'tasks', to: '/work/my-tasks', icon: ListChecks, label: 'layout.navItems.tasks', enabled: true, section: 'main', badge: { type: 'text', value: '5' }, color: { h: 217, s: 78 } },
  // Unified communication (internal team chat + customer inbox). id stays 'chat'
  // to preserve module assignment + pin state; route/label point to the merged module.
  { id: 'chat', to: '/kommunikation', icon: MessageSquareText, label: 'layout.navItems.kommunikation', enabled: true, section: 'main', badge: { type: 'text', value: '3' }, color: { h: 195, s: 82 } },
  { id: 'contacts', to: '/kontakte', icon: Contact, label: 'layout.navItems.contacts', enabled: true, section: 'main', color: { h: 162, s: 68 } },

  // ── Team & HR ──
  { id: 'team', to: '/team', icon: Users, label: 'layout.navItems.team', enabled: true, section: 'main', color: { h: 271, s: 68 } },

  // ── Communication ──
  { id: 'meetings', to: '/meetings', icon: Video, label: 'layout.navItems.meetings', enabled: true, section: 'main', badge: { type: 'live' }, color: { h: 338, s: 72 } },
  { id: 'calendar', to: '/kalender', icon: Calendar, label: 'layout.navItems.calendar', enabled: true, section: 'main', color: { h: 25, s: 82 } },
  { id: 'zeiterfassung', to: '/zeiterfassung', icon: Timer, label: 'layout.navItems.zeiterfassung', enabled: true, section: 'main', color: { h: 280, s: 60 } },
  { id: 'documents', to: '/dokumente', icon: FileText, label: 'layout.navItems.documents', enabled: true, section: 'main', color: { h: 152, s: 62 } },
  { id: 'wiki', to: '/wiki', icon: BookOpen, label: 'layout.navItems.wiki', enabled: true, section: 'main', color: { h: 168, s: 58 } },
  { id: 'mail', to: '/mails', icon: Mail, label: 'layout.navItems.mail', enabled: true, section: 'main', badge: { type: 'text', value: '12' }, color: { h: 205, s: 78 } },

  // ── Finance ──
  { id: 'finance', to: '/finanzen', icon: Receipt, label: 'layout.navItems.finance', enabled: true, section: 'main', badge: { type: 'text', value: 'layout.navItems.badgeNew' }, color: { h: 38, s: 88 } },
  { id: 'infrastructure', to: '/infrastruktur', icon: Network, label: 'layout.navItems.infrastructure', enabled: true, section: 'main', color: { h: 215, s: 32 } },

  // ── Industry modules ──
  { id: 'inventar', to: '/inventar', icon: Warehouse, label: 'layout.navItems.inventar', enabled: true, section: 'main', color: { h: 15, s: 75 } },
  { id: 'schichten', to: '/schichten', icon: CalendarClock, label: 'layout.navItems.schichten', enabled: true, section: 'main', color: { h: 258, s: 62 } },
  { id: 'einkauf', to: '/einkauf', icon: ShoppingCart, label: 'layout.navItems.einkauf', enabled: true, section: 'main', color: { h: 45, s: 82 } },
  { id: 'helpdesk', to: '/helpdesk', icon: LifeBuoy, label: 'layout.navItems.helpdesk', enabled: true, section: 'main', color: { h: 308, s: 68 } },
  { id: 'fuhrpark', to: '/fuhrpark', icon: Truck, label: 'layout.navItems.fuhrpark', enabled: true, section: 'main', color: { h: 348, s: 72 } },
  { id: 'produktion', to: '/produktion', icon: Factory, label: 'layout.navItems.produktion', enabled: true, section: 'main', color: { h: 130, s: 58 } },
  { id: 'berichte', to: '/berichte', icon: BarChart3, label: 'layout.navItems.berichte', enabled: true, section: 'main', color: { h: 42, s: 78 } },
  { id: 'vertraege', to: '/vertraege', icon: FileSignature, label: 'layout.navItems.vertraege', enabled: true, section: 'main', color: { h: 155, s: 62 } },
  { id: 'formulare', to: '/formulare', icon: FileInput, label: 'layout.navItems.formulare', enabled: true, section: 'main', color: { h: 222, s: 68 } },
  { id: 'vermietung', to: '/vermietung', icon: Building2, label: 'layout.navItems.vermietung', enabled: true, section: 'main', color: { h: 8, s: 72 } },
  { id: 'rapporte', to: '/rapporte', icon: ClipboardCheck, label: 'layout.navItems.rapporte', enabled: true, section: 'main', color: { h: 98, s: 55 } },

  // ── Sales & Outbound ──
  { id: 'dialer', to: '/dialer', icon: PhoneCall, label: 'layout.navItems.dialer', enabled: true, section: 'main', color: { h: 142, s: 72 } },

  // ── Automation ──
  { id: 'automatisierung', to: '/automatisierung', icon: Workflow, label: 'layout.navItems.automatisierung', enabled: true, section: 'main', color: { h: 188, s: 78 } },

  // ── System (bottom) ──
  // "Verwaltung" is the IT home of Cosmi (role builder, users, security) — a
  // real module by Darien's R-2 call, not buried in settings. RBAC-gated via
  // NAV_ITEM_MODULE (admin:module:view → admin/it_admin/hr_admin only).
  { id: 'admin', to: '/admin/roles', icon: ShieldCheck, label: 'layout.navItems.admin', enabled: true, section: 'bottom', color: { h: 355, s: 65 } },
  { id: 'notifications', to: '/notifications', icon: Bell, label: 'layout.navItems.notifications', enabled: true, section: 'bottom', color: { h: 280, s: 65 } },
  { id: 'settings', to: '/settings', icon: Cog, label: 'layout.navItems.settings', enabled: true, section: 'bottom', color: { h: 220, s: 18 } },
]
