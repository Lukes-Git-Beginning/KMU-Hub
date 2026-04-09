import { NavLink, Routes, Route, Navigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { PhoneCall, LayoutDashboard, Settings, Headset } from 'lucide-react'
import { cn } from '@/lib/cn'
import { useDialerStore } from '@/stores/dialer'

import CampaignListPage from './CampaignListPage'
import CampaignDetailPage from './CampaignDetailPage'
import DialerWorkspacePage from './DialerWorkspacePage'
import AgentDashboardPage from './AgentDashboardPage'
import DialerSettingsPage from './DialerSettingsPage'

const dialerNavItems = [
  { to: '/dialer/campaigns', icon: PhoneCall, labelKey: 'dialer.nav.campaigns' },
  { to: '/dialer/workspace', icon: Headset, labelKey: 'dialer.nav.workspace', showLiveDot: true },
  { to: '/dialer/dashboard', icon: LayoutDashboard, labelKey: 'dialer.nav.dashboard' },
  { to: '/dialer/settings', icon: Settings, labelKey: 'dialer.nav.settings' },
] as const

export default function DialerLayout() {
  const { t } = useTranslation()
  const callPhase = useDialerStore((s) => s.callPhase)

  return (
    <div className="flex h-full flex-col animate-fade-in">
      <nav className="flex items-center gap-1 border-b border-border bg-card px-6 py-2">
        {dialerNavItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            className={({ isActive }) =>
              cn(
                'relative flex items-center gap-2 rounded-lg px-3 py-1.5 text-sm font-medium transition-all',
                isActive
                  ? 'bg-primary/10 text-primary tab-accent-active'
                  : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
              )
            }
          >
            <item.icon className="h-4 w-4" />
            <span>{t(item.labelKey)}</span>
            {item.showLiveDot && callPhase !== 'idle' && (
              <span className="absolute -right-0.5 -top-0.5 h-2.5 w-2.5 rounded-full bg-success animate-glow-pulse" />
            )}
          </NavLink>
        ))}
      </nav>

      <div className="flex-1 overflow-auto">
        <Routes>
          <Route index element={<Navigate to="/dialer/campaigns" replace />} />
          <Route path="campaigns" element={<CampaignListPage />} />
          <Route path="campaigns/:id" element={<CampaignDetailPage />} />
          <Route path="workspace" element={<DialerWorkspacePage />} />
          <Route path="dashboard" element={<AgentDashboardPage />} />
          <Route path="settings" element={<DialerSettingsPage />} />
        </Routes>
      </div>
    </div>
  )
}
