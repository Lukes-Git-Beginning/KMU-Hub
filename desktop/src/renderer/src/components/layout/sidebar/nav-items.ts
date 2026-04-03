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
  { id: 'dashboard', to: '/', icon: LayoutDashboard, label: 'Übersicht', enabled: true, section: 'main', color: { h: 240, s: 72 } },
  { id: 'projects', to: '/work/projects', icon: FolderKanban, label: 'Projekte', enabled: true, section: 'main', color: { h: 255, s: 68 } },
  { id: 'tasks', to: '/work/my-tasks', icon: ListChecks, label: 'Aufgaben', enabled: true, section: 'main', badge: { type: 'text', value: '5' }, color: { h: 217, s: 78 } },
  { id: 'chat', to: '/chat', icon: MessageSquare, label: 'Team Chat', enabled: true, section: 'main', badge: { type: 'text', value: '3' }, color: { h: 195, s: 82 } },
  { id: 'contacts', to: '/kontakte', icon: Contact, label: 'Kontakte', enabled: true, section: 'main', color: { h: 162, s: 68 } },

  // ── Team & HR ──
  { id: 'team', to: '/team', icon: Users, label: 'Team', enabled: true, section: 'main', color: { h: 271, s: 68 } },

  // ── Communication ──
  { id: 'meetings', to: '/meetings', icon: Video, label: 'Meetings', enabled: true, section: 'main', badge: { type: 'live' }, color: { h: 338, s: 72 } },
  { id: 'calendar', to: '/kalender', icon: Calendar, label: 'Kalender', enabled: true, section: 'main', color: { h: 25, s: 82 } },
  { id: 'zeiterfassung', to: '/zeiterfassung', icon: Timer, label: 'Zeiterfassung', enabled: true, section: 'main', color: { h: 280, s: 60 } },
  { id: 'documents', to: '/dokumente', icon: FileText, label: 'Dokumente', enabled: true, section: 'main', color: { h: 152, s: 62 } },
  { id: 'wiki', to: '/wiki', icon: BookOpen, label: 'Wiki', enabled: true, section: 'main', color: { h: 168, s: 58 } },
  { id: 'mail', to: '/mails', icon: Mail, label: 'E-Mail', enabled: true, section: 'main', badge: { type: 'text', value: '12' }, color: { h: 205, s: 78 } },
  { id: 'kommunikation', to: '/kommunikation', icon: MessageSquareText, label: 'Posteingang', enabled: true, section: 'main', badge: { type: 'text', value: '' }, color: { h: 185, s: 72 } },

  // ── Finance ──
  { id: 'finance', to: '/finanzen', icon: Receipt, label: 'Finanzen', enabled: true, section: 'main', badge: { type: 'text', value: 'Neu' }, color: { h: 38, s: 88 } },
  { id: 'infrastructure', to: '/infrastruktur', icon: Network, label: 'Infrastruktur', enabled: true, section: 'main', color: { h: 215, s: 32 } },

  // ── Industry modules ──
  { id: 'inventar', to: '/inventar', icon: Warehouse, label: 'Inventar', enabled: true, section: 'main', color: { h: 15, s: 75 } },
  { id: 'schichten', to: '/schichten', icon: CalendarClock, label: 'Schichtplanung', enabled: true, section: 'main', color: { h: 258, s: 62 } },
  { id: 'einkauf', to: '/einkauf', icon: ShoppingCart, label: 'Einkauf', enabled: true, section: 'main', color: { h: 45, s: 82 } },
  { id: 'helpdesk', to: '/helpdesk', icon: LifeBuoy, label: 'Helpdesk', enabled: true, section: 'main', color: { h: 308, s: 68 } },
  { id: 'fuhrpark', to: '/fuhrpark', icon: Truck, label: 'Fuhrpark', enabled: true, section: 'main', color: { h: 348, s: 72 } },
  { id: 'produktion', to: '/produktion', icon: Factory, label: 'Produktion', enabled: true, section: 'main', color: { h: 130, s: 58 } },
  { id: 'berichte', to: '/berichte', icon: BarChart3, label: 'Berichte', enabled: true, section: 'main', color: { h: 42, s: 78 } },
  { id: 'vertraege', to: '/vertraege', icon: FileSignature, label: 'Verträge', enabled: true, section: 'main', color: { h: 155, s: 62 } },
  { id: 'formulare', to: '/formulare', icon: FileInput, label: 'Formulare', enabled: true, section: 'main', color: { h: 222, s: 68 } },
  { id: 'vermietung', to: '/vermietung', icon: Building2, label: 'Vermietung', enabled: true, section: 'main', color: { h: 8, s: 72 } },
  { id: 'rapporte', to: '/rapporte', icon: ClipboardCheck, label: 'Rapporte', enabled: true, section: 'main', color: { h: 98, s: 55 } },

  // ── System (bottom) ──
  { id: 'security-admin', to: '/admin/security', icon: ShieldCheck, label: 'Sicherheit', enabled: true, section: 'bottom', color: { h: 0, s: 68 } },
  { id: 'settings', to: '/settings', icon: Cog, label: 'Einstellungen', enabled: true, section: 'bottom', color: { h: 220, s: 18 } },
]
