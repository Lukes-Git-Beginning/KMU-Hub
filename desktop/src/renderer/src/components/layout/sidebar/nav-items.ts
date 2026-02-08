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
  roles?: string[]
  enabled: boolean
  section: 'main' | 'bottom'
}

export const navItems: NavItemConfig[] = [
  { id: 'dashboard', to: '/', icon: LayoutDashboard, label: 'Uebersicht', enabled: true, section: 'main' },
  { id: 'projects', to: '/work/projects', icon: FolderKanban, label: 'Projekte', enabled: true, section: 'main' },
  { id: 'tasks', to: '/work/my-tasks', icon: CheckSquare, label: 'Aufgaben', enabled: true, section: 'main', badge: { type: 'text', value: '5' } },
  { id: 'documents', to: '/dokumente', icon: FileText, label: 'Dokumente', enabled: false, section: 'main' },
  { id: 'finance', to: '/buchhaltung', icon: Calculator, label: 'Buchhaltung', enabled: false, section: 'main', badge: { type: 'text', value: 'Neu' } },
  { id: 'chat', to: '/chat', icon: MessageSquare, label: 'Nachrichten', enabled: true, section: 'main', badge: { type: 'text', value: '3' } },
  { id: 'team', to: '/crm', icon: Users, label: 'Team & CRM', enabled: true, section: 'main' },
  { id: 'mail', to: '/mails', icon: Mail, label: 'E-Mail', enabled: false, section: 'main', badge: { type: 'text', value: '12' } },
  { id: 'contacts', to: '/kontakte', icon: Contact, label: 'Kontakte', enabled: false, section: 'main' },
  { id: 'meetings', to: '/meetings', icon: Calendar, label: 'Meetings', enabled: false, section: 'main', badge: { type: 'live' } },
  { id: 'settings', to: '/settings/dashboard', icon: Cog, label: 'Einstellungen', enabled: true, section: 'bottom', roles: ['admin'] },
]
