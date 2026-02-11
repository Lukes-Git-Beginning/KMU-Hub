/**
 * Main settings page with sidebar navigation and tab content.
 *
 * Available tabs:
 * - General: Dashboard widget defaults (admin only, existing DashboardSettings)
 * - Security: 2FA, sessions, password change
 * - Language: Locale picker with format previews
 * - Privacy: GDPR data export and deletion info
 *
 * Inspired by design/brainstorm SettingsPage layout with sidebar navigation.
 */
import { useState } from 'react'
import { Navigate } from 'react-router-dom'
import { Globe, Lock, LayoutDashboard, Shield } from 'lucide-react'
import { FormattedMessage } from 'react-intl'
import { useAuthStore } from '@/stores/auth'
import DashboardSettings from './DashboardSettings'
import { SecuritySettingsTab } from './SecuritySettingsTab'
import { LanguageSettingsTab } from './LanguageSettingsTab'
import { PrivacySettingsTab } from './PrivacySettingsTab'

type TabKey = 'general' | 'security' | 'language' | 'privacy'

interface TabConfig {
  key: TabKey
  labelId: string
  subtitleId: string
  icon: typeof Globe
  /** If set, only users with one of these roles see this tab. */
  roles?: string[]
}

const TABS: TabConfig[] = [
  {
    key: 'general',
    labelId: 'settings.general.title',
    subtitleId: 'settings.general.subtitle',
    icon: LayoutDashboard,
    roles: ['admin'],
  },
  {
    key: 'security',
    labelId: 'settings.security.title',
    subtitleId: 'settings.security.subtitle',
    icon: Shield,
  },
  {
    key: 'language',
    labelId: 'settings.language.title',
    subtitleId: 'settings.language.subtitle',
    icon: Globe,
  },
  {
    key: 'privacy',
    labelId: 'settings.privacy.title',
    subtitleId: 'settings.privacy.subtitle',
    icon: Lock,
  },
]

export default function SettingsPage() {
  const user = useAuthStore((s) => s.user)
  const isAdmin = user?.roles.includes('admin')

  // Filter tabs by role
  const visibleTabs = TABS.filter((tab) => {
    if (!tab.roles) return true
    return tab.roles.some((r) => user?.roles.includes(r))
  })

  // Default to first visible tab
  const defaultTab = visibleTabs[0]?.key ?? 'security'
  const [activeTab, setActiveTab] = useState<TabKey>(defaultTab)

  // If active tab got hidden (e.g. after role change), fall back
  const isActiveVisible = visibleTabs.some((t) => t.key === activeTab)
  const effectiveTab = isActiveVisible ? activeTab : defaultTab

  // Non-authenticated users shouldn't reach here, but guard anyway
  if (!user) {
    return <Navigate to="/login" replace />
  }

  return (
    <div className="flex h-full overflow-hidden">
      {/* Settings sidebar */}
      <aside className="w-56 shrink-0 border-r border-border bg-card p-4 overflow-y-auto">
        <h3 className="text-sm font-medium text-foreground mb-4 px-2">
          <FormattedMessage id="nav.settings" />
        </h3>
        <nav className="space-y-0.5">
          {visibleTabs.map((tab) => {
            const Icon = tab.icon
            return (
              <button
                key={tab.key}
                onClick={() => setActiveTab(tab.key)}
                className={`flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-sm transition-colors ${
                  effectiveTab === tab.key
                    ? 'bg-secondary text-secondary-foreground font-medium'
                    : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
                }`}
              >
                <Icon className="h-4 w-4 shrink-0" />
                <FormattedMessage id={tab.labelId} />
              </button>
            )
          })}
        </nav>
      </aside>

      {/* Content area */}
      <div className="flex-1 overflow-y-auto">
        {effectiveTab === 'general' && isAdmin && <DashboardSettings />}
        {effectiveTab === 'security' && <SecuritySettingsTab />}
        {effectiveTab === 'language' && <LanguageSettingsTab />}
        {effectiveTab === 'privacy' && <PrivacySettingsTab />}
      </div>
    </div>
  )
}
