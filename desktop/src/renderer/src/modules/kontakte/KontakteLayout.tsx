/**
 * Kontakte module layout — "Kunden-Zentrale" with sub-navigation tabs.
 *
 * Provides horizontal tab navigation between customer-facing sections:
 * Kontakte (contacts master-detail), Firmen, Pipeline, Aktivitäten.
 * Renders the active child route via <Outlet/>.
 */
import { NavLink, Outlet } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Users, Building2, TrendingUp, Activity, Inbox } from 'lucide-react'
import { cn } from '@/lib/cn'

const kontakteNavItems = [
  { to: '/kontakte', icon: Users, labelKey: 'crm.nav.contacts', end: true },
  { to: '/kontakte/leads', icon: Inbox, labelKey: 'crm.nav.leads', end: false },
  { to: '/kontakte/firmen', icon: Building2, labelKey: 'crm.nav.companies', end: false },
  { to: '/kontakte/pipeline', icon: TrendingUp, labelKey: 'crm.nav.deals', end: false },
  { to: '/kontakte/aktivitaeten', icon: Activity, labelKey: 'crm.nav.activities', end: false },
] as const

export default function KontakteLayout() {
  const { t } = useTranslation()
  return (
    <div className="flex h-full flex-col animate-fade-in">
      {/* Sub-navigation bar — 1:1 Optik wie CRMLayout */}
      <nav className="flex items-center gap-1 border-b border-border bg-card px-6 py-2">
        {kontakteNavItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.end}
            className={({ isActive }) =>
              cn(
                'flex items-center gap-2 rounded-lg px-3 py-1.5 text-sm font-medium transition-all',
                isActive
                  ? 'bg-primary/10 text-primary tab-accent-active'
                  : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
              )
            }
          >
            <item.icon className="h-4 w-4" />
            <span>{t(item.labelKey)}</span>
          </NavLink>
        ))}
      </nav>

      {/* Content area */}
      <div className="flex-1 overflow-auto">
        <Outlet />
      </div>
    </div>
  )
}
