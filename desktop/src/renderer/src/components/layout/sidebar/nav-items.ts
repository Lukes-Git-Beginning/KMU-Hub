import {
  LayoutDashboard,
  FolderKanban,
  CheckSquare,
  FileText,
  Calculator,
  MessageSquare,
  Users,
  Mail,
  Contact,
  Calendar,
  Cog,
  Video,
  Server,
  Package,
  CalendarClock,
  ShoppingCart,
  Headphones,
  Truck,
  Factory,
  BarChart3,
  FileSignature,
  ClipboardList,
  KeyRound,
  HardHat,
  Timer,
} from 'lucide-react'

export interface NavBadge {
  type: 'text' | 'live'
  value?: string
}

export interface NavItemConfig {
  id: string
  to: string
  icon: typeof LayoutDashboard
  label: string
  badge?: NavBadge
  enabled: boolean
  section: 'main' | 'bottom'
}

export const navItems: NavItemConfig[] = [
  // -- Visible to ALL roles --
  { id: 'dashboard', to: '/', icon: LayoutDashboard, label: 'Übersicht', enabled: true, section: 'main' },
  { id: 'projects', to: '/work/projects', icon: FolderKanban, label: 'Projekte', enabled: true, section: 'main' },
  { id: 'tasks', to: '/work/my-tasks', icon: CheckSquare, label: 'Aufgaben', enabled: true, section: 'main', badge: { type: 'text', value: '5' } },
  { id: 'chat', to: '/chat', icon: MessageSquare, label: 'Nachrichten', enabled: true, section: 'main', badge: { type: 'text', value: '3' } },
  { id: 'contacts', to: '/kontakte', icon: Contact, label: 'Kontakte', enabled: true, section: 'main' },

  // -- Role-restricted (filtered by canSeeNavItem in Sidebar) --
  { id: 'team', to: '/team', icon: Users, label: 'Team', enabled: true, section: 'main' },

  // -- Visible to ALL roles --
  { id: 'meetings', to: '/meetings', icon: Video, label: 'Meetings', enabled: true, section: 'main', badge: { type: 'live' } },
  { id: 'calendar', to: '/kalender', icon: Calendar, label: 'Kalender', enabled: true, section: 'main' },
  { id: 'zeiterfassung', to: '/zeiterfassung', icon: Timer, label: 'Zeiterfassung', enabled: true, section: 'main' },
  { id: 'documents', to: '/dokumente', icon: FileText, label: 'Dokumente', enabled: true, section: 'main' },
  { id: 'mail', to: '/mails', icon: Mail, label: 'E-Mail', enabled: true, section: 'main', badge: { type: 'text', value: '12' } },

  // -- Role-restricted --
  { id: 'finance', to: '/buchhaltung', icon: Calculator, label: 'Buchhaltung', enabled: true, section: 'main', badge: { type: 'text', value: 'Neu' } },
  { id: 'infrastructure', to: '/infrastruktur', icon: Server, label: 'Infrastruktur', enabled: true, section: 'main' },

  // -- Industry-specific modules (filtered by business profile) --
  { id: 'inventar', to: '/inventar', icon: Package, label: 'Inventar', enabled: true, section: 'main' },
  { id: 'schichten', to: '/schichten', icon: CalendarClock, label: 'Schichtplanung', enabled: true, section: 'main' },
  { id: 'einkauf', to: '/einkauf', icon: ShoppingCart, label: 'Einkauf', enabled: true, section: 'main' },
  { id: 'helpdesk', to: '/helpdesk', icon: Headphones, label: 'Helpdesk', enabled: true, section: 'main' },
  { id: 'fuhrpark', to: '/fuhrpark', icon: Truck, label: 'Fuhrpark', enabled: true, section: 'main' },
  { id: 'produktion', to: '/produktion', icon: Factory, label: 'Produktion', enabled: true, section: 'main' },
  { id: 'berichte', to: '/berichte', icon: BarChart3, label: 'Berichte', enabled: true, section: 'main' },
  { id: 'vertraege', to: '/vertraege', icon: FileSignature, label: 'Vertraege', enabled: true, section: 'main' },
  { id: 'formulare', to: '/formulare', icon: ClipboardList, label: 'Formulare', enabled: true, section: 'main' },
  { id: 'vermietung', to: '/vermietung', icon: KeyRound, label: 'Vermietung', enabled: true, section: 'main' },
  { id: 'rapporte', to: '/rapporte', icon: HardHat, label: 'Rapporte', enabled: true, section: 'main' },

  // -- Bottom: Admin/IT only in sidebar --
  { id: 'settings', to: '/settings', icon: Cog, label: 'Einstellungen', enabled: true, section: 'bottom' },
]
