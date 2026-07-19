import { NavLink, Routes, Route, Navigate, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { PhoneCall, LayoutDashboard, Settings, Headset, Users, type LucideIcon } from 'lucide-react'
import { cn } from '@/lib/cn'
import { useDialerStore } from '@/stores/dialer'
import { useCapabilitySet } from '@/hooks/useCapability'
import { RestrictedModeBadge } from '@/components/shared/rbac/RestrictedModeBadge'
import { EmptyState } from '@/components/shared'

import CampaignListPage from './CampaignListPage'
import CampaignDetailPage from './CampaignDetailPage'
import DialerWorkspacePage from './DialerWorkspacePage'
import AgentDashboardPage from './AgentDashboardPage'
import SupervisorDashboardPage from './SupervisorDashboardPage'
import DialerSettingsPage from './DialerSettingsPage'

type DialerTab = 'campaigns' | 'workspace' | 'dashboard' | 'supervisor' | 'settings'

interface DialerNavItem {
  key: DialerTab
  to: string
  icon: LucideIcon
  labelKey: string
  capabilityKey: string
  showLiveDot?: boolean
}

const ALL_NAV_ITEMS: DialerNavItem[] = [
  {
    key: 'campaigns',
    to: '/dialer/campaigns',
    icon: PhoneCall,
    labelKey: 'dialer.nav.campaigns',
    capabilityKey: 'dialer:campaigns:read',
  },
  {
    key: 'workspace',
    to: '/dialer/workspace',
    icon: Headset,
    labelKey: 'dialer.nav.workspace',
    capabilityKey: 'dialer:calls:write',
    showLiveDot: true,
  },
  {
    key: 'dashboard',
    to: '/dialer/dashboard',
    icon: LayoutDashboard,
    labelKey: 'dialer.nav.dashboard',
    capabilityKey: 'dialer:calls:read',
  },
  {
    key: 'supervisor',
    to: '/dialer/supervisor',
    icon: Users,
    labelKey: 'dialer.nav.supervisor',
    capabilityKey: 'dialer:agent:manage',
  },
  {
    key: 'settings',
    to: '/dialer/settings',
    icon: Settings,
    labelKey: 'dialer.nav.settings',
    capabilityKey: 'dialer:outcomes:manage',
  },
]

// Map from sub-path to tab key for route-guard checks
const PATH_TO_TAB: Record<string, DialerTab> = {
  campaigns: 'campaigns',
  workspace: 'workspace',
  dashboard: 'dashboard',
  supervisor: 'supervisor',
  settings: 'settings',
}

export default function DialerLayout() {
  const { t } = useTranslation()
  const callPhase = useDialerStore((s) => s.callPhase)
  const location = useLocation()

  const { has, ready } = useCapabilitySet()

  // While not ready — show all tabs to avoid flicker (pessimistic default until
  // the permission set is loaded, same pattern as AdminHubPage / InventarPage).
  const visibleItems = ready
    ? ALL_NAV_ITEMS.filter((item) => has(item.capabilityKey))
    : ALL_NAV_ITEMS

  // Derive the requested tab from the current pathname
  const pathSegments = location.pathname.split('/')
  const subPath = pathSegments[2] as DialerTab | undefined
  const requestedTab: DialerTab | null = (subPath && PATH_TO_TAB[subPath]) ?? null

  // moduleEmpty: module visible (level 1) but no readable sub-route at all →
  // honest empty state instead of an empty nav (rbac gating convention)
  if (ready && visibleItems.length === 0) {
    return (
      <div className="flex h-full flex-col">
        <nav className="flex items-center justify-between gap-1 border-b border-border bg-card px-6 py-2">
          <span className="text-sm font-medium text-muted-foreground">
            {t('rbac.module.dialer')}
          </span>
          <RestrictedModeBadge module="dialer" />
        </nav>
        <div className="flex-1 flex items-center justify-center p-6">
          <EmptyState
            icon={PhoneCall}
            title={t('rbac.gate.moduleEmpty')}
            description={t('rbac.gate.noPermission')}
          />
        </div>
      </div>
    )
  }

  // Route guard: if the user directly navigates to a sub-route they can't see,
  // redirect to the first allowed one (only evaluate after ready to prevent flicker).
  if (ready && requestedTab !== null) {
    const isAllowed = visibleItems.some((item) => item.key === requestedTab)
    if (!isAllowed) {
      const firstAllowed = visibleItems[0]
      if (!firstAllowed) {
        return <Navigate to="/" replace />
      }
      return <Navigate to={firstAllowed.to} replace />
    }
  }

  // Index redirect: /dialer → first allowed tab (or campaigns while not ready)
  const indexRedirect =
    ready && visibleItems.length > 0
      ? visibleItems[0].to
      : '/dialer/campaigns'

  return (
    <div className="flex h-full flex-col animate-fade-in">
      <nav className="flex items-center justify-between gap-1 border-b border-border bg-card px-6 py-2">
        <div className="flex items-center gap-1">
          {visibleItems.map((item) => (
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
        </div>

        <RestrictedModeBadge module="dialer" />
      </nav>

      <div className="flex-1 overflow-auto">
        <Routes>
          <Route index element={<Navigate to={indexRedirect} replace />} />
          <Route path="campaigns" element={<CampaignListPage />} />
          <Route path="campaigns/:id" element={<CampaignDetailPage />} />
          <Route path="workspace" element={<DialerWorkspacePage />} />
          <Route path="dashboard" element={<AgentDashboardPage />} />
          <Route path="supervisor" element={<SupervisorDashboardPage />} />
          <Route path="settings" element={<DialerSettingsPage />} />
        </Routes>
      </div>
    </div>
  )
}
