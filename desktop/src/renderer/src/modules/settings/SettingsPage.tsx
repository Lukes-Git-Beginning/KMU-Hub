/**
 * Main settings page with sidebar navigation and tab content.
 *
 * Available tabs:
 * - General: Dashboard widget defaults (admin only)
 * - Security: 2FA, sessions, password change
 * - Language: Locale picker with format previews
 * - Privacy: GDPR data export and deletion info
 * - Calendar: Calendar display & defaults
 * - Finance: Invoice & currency settings
 * - Mail: Email account configuration
 * - Team: HR & team settings (admin/manager)
 * - Integrations: Teams, Slack & external services (admin)
 */
import { useState } from 'react'
import { Navigate } from 'react-router-dom'
import { Globe, Lock, LayoutDashboard, Shield, Calendar, Receipt, Mail, Users, RefreshCw, Plug } from 'lucide-react'
import { FormattedMessage } from 'react-intl'
import { useAuthStore } from '@/stores/auth'
import DashboardSettings from './DashboardSettings'
import { SecuritySettingsTab } from './SecuritySettingsTab'
import { LanguageSettingsTab } from './LanguageSettingsTab'
import { PrivacySettingsTab } from './PrivacySettingsTab'
import { CalendarSettingsTab } from './tabs/CalendarSettingsTab'
import { FinanceSettingsTab } from './tabs/FinanceSettingsTab'
import { MailSettingsTab } from './tabs/MailSettingsTab'
import { TeamSettingsTab } from './tabs/TeamSettingsTab'
import { CalDAVSettingsTab } from './tabs/CalDAVSettingsTab'
import { IntegrationsSettingsTab } from './tabs/IntegrationsSettingsTab'

type TabKey = 'general' | 'security' | 'language' | 'privacy' | 'calendar' | 'finance' | 'mail' | 'team' | 'caldav' | 'integrations'

interface TabConfig {
  key: TabKey
  labelId: string
  subtitleId: string
  icon: typeof Globe
  /** If set, only users with one of these roles see this tab. */
  roles?: string[]
  group?: string
}

const TABS: TabConfig[] = [
  {
    key: 'general',
    labelId: 'settings.general.title',
    subtitleId: 'settings.general.subtitle',
    icon: LayoutDashboard,
    roles: ['admin'],
    group: 'system',
  },
  {
    key: 'security',
    labelId: 'settings.security.title',
    subtitleId: 'settings.security.subtitle',
    icon: Shield,
    group: 'personal',
  },
  {
    key: 'language',
    labelId: 'settings.language.title',
    subtitleId: 'settings.language.subtitle',
    icon: Globe,
    group: 'personal',
  },
  {
    key: 'privacy',
    labelId: 'settings.privacy.title',
    subtitleId: 'settings.privacy.subtitle',
    icon: Lock,
    group: 'personal',
  },
  {
    key: 'calendar',
    labelId: 'settings.calendar.title',
    subtitleId: 'settings.calendar.subtitle',
    icon: Calendar,
    group: 'modules',
  },
  {
    key: 'caldav',
    labelId: 'settings.caldav.title',
    subtitleId: 'settings.caldav.subtitle',
    icon: RefreshCw,
    group: 'modules',
  },
  {
    key: 'finance',
    labelId: 'settings.finance.title',
    subtitleId: 'settings.finance.subtitle',
    icon: Receipt,
    roles: ['admin', 'manager'],
    group: 'modules',
  },
  {
    key: 'mail',
    labelId: 'settings.mail.title',
    subtitleId: 'settings.mail.subtitle',
    icon: Mail,
    group: 'modules',
  },
  {
    key: 'team',
    labelId: 'settings.team.title',
    subtitleId: 'settings.team.subtitle',
    icon: Users,
    roles: ['admin', 'manager', 'hr'],
    group: 'modules',
  },
  {
    key: 'integrations',
    labelId: 'settings.integrations.title',
    subtitleId: 'settings.integrations.subtitle',
    icon: Plug,
    roles: ['admin'],
    group: 'modules',
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

  // Group tabs for sidebar sections
  const personalTabs = visibleTabs.filter((t) => t.group === 'personal' || t.group === 'system')
  const moduleTabs = visibleTabs.filter((t) => t.group === 'modules')

  const renderTabButton = (tab: TabConfig) => {
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
        <FormattedMessage id={tab.labelId} defaultMessage={tab.key} />
      </button>
    )
  }

  return (
    <div className="flex h-full overflow-hidden">
      {/* Settings sidebar */}
      <aside className="w-56 shrink-0 border-r border-border bg-card p-4 overflow-y-auto">
        <h3 className="text-sm font-medium text-foreground mb-4 px-2">
          <FormattedMessage id="nav.settings" />
        </h3>
        <nav className="space-y-0.5">
          {personalTabs.map(renderTabButton)}
        </nav>

        {moduleTabs.length > 0 && (
          <>
            <div className="my-3 border-t border-border-muted" />
            <h4 className="text-xs font-medium text-muted-foreground mb-2 px-2 uppercase tracking-wider">
              <FormattedMessage id="settings.modules" defaultMessage="Module" />
            </h4>
            <nav className="space-y-0.5">
              {moduleTabs.map(renderTabButton)}
            </nav>
          </>
        )}
      </aside>

      {/* Content area */}
      <div className="flex-1 overflow-y-auto">
        {effectiveTab === 'general' && isAdmin && <DashboardSettings />}
        {effectiveTab === 'security' && <SecuritySettingsTab />}
        {effectiveTab === 'language' && <LanguageSettingsTab />}
        {effectiveTab === 'privacy' && <PrivacySettingsTab />}
        {effectiveTab === 'calendar' && <CalendarSettingsTab />}
        {effectiveTab === 'caldav' && <CalDAVSettingsTab />}
        {effectiveTab === 'finance' && <FinanceSettingsTab />}
        {effectiveTab === 'mail' && <MailSettingsTab />}
        {effectiveTab === 'team' && <TeamSettingsTab />}
        {effectiveTab === 'integrations' && <IntegrationsSettingsTab />}
      </div>
    </div>
  )
}
