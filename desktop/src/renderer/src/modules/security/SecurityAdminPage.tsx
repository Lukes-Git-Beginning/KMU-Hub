/**
 * Security Admin hub page with sub-navigation.
 *
 * Same layout pattern as SettingsPage: left sidebar nav + right content area.
 * Admin-only — redirects non-admins to dashboard.
 * Each tab lazy-renders one of the 7 security sub-pages.
 */
import { useState, lazy, Suspense } from 'react'
import { Navigate } from 'react-router-dom'
import {
  ShieldCheck,
  ScrollText,
  Monitor,
  Lock,
  KeyRound,
  Globe,
  FileText,
  AlertTriangle,
  Search,
  Clock,
} from 'lucide-react'
import { useAuthStore } from '@/stores/auth'

// Lazy-load sub-pages so they don't bloat the initial bundle
const AuditLogPage = lazy(() => import('./AuditLogPage'))
const SessionsPage = lazy(() => import('./SessionsPage'))
const VaultPage = lazy(() => import('./VaultPage'))
const PasswordPolicyPage = lazy(() => import('./PasswordPolicyPage'))
const IPAccessPage = lazy(() => import('./IPAccessPage'))
const GDPRExportPage = lazy(() => import('./GDPRExportPage'))
const GDPRErasurePage = lazy(() => import('./GDPRErasurePage'))
const DSARSearchPage = lazy(() => import('./DSARSearchPage'))
const RetentionPolicyPage = lazy(() => import('./RetentionPolicyPage'))

type TabKey = 'audit' | 'sessions' | 'vault' | 'password-policy' | 'ip-access' | 'gdpr-exports' | 'gdpr-erasure' | 'dsar' | 'retention'

interface TabConfig {
  key: TabKey
  label: string
  icon: typeof ShieldCheck
  group: string
}

const ALL_TABS: TabConfig[] = [
  { key: 'audit', label: 'Audit Log', icon: ScrollText, group: 'Überwachung' },
  { key: 'sessions', label: 'Sitzungen', icon: Monitor, group: 'Überwachung' },
  { key: 'vault', label: 'Vault', icon: Lock, group: 'Zugriff' },
  { key: 'password-policy', label: 'Passwort-Richtlinie', icon: KeyRound, group: 'Zugriff' },
  { key: 'ip-access', label: 'IP-Zugriff', icon: Globe, group: 'Zugriff' },
  { key: 'gdpr-exports', label: 'Datenexport', icon: FileText, group: 'Datenschutz' },
  { key: 'gdpr-erasure', label: 'Datenlöschung', icon: AlertTriangle, group: 'Datenschutz' },
  { key: 'dsar', label: 'Auskunft (Art. 15)', icon: Search, group: 'Datenschutz' },
  { key: 'retention', label: 'Aufbewahrungsfristen', icon: Clock, group: 'Datenschutz' },
]

const TAB_GROUPS = ['Überwachung', 'Zugriff', 'Datenschutz']

function TabContent({ tab }: { tab: TabKey }) {
  return (
    <Suspense
      fallback={
        <div className="flex items-center justify-center h-64">
          <p className="text-sm text-muted-foreground">Laden...</p>
        </div>
      }
    >
      {tab === 'audit' && <AuditLogPage />}
      {tab === 'sessions' && <SessionsPage />}
      {tab === 'vault' && <VaultPage />}
      {tab === 'password-policy' && <PasswordPolicyPage />}
      {tab === 'ip-access' && <IPAccessPage />}
      {tab === 'gdpr-exports' && <GDPRExportPage />}
      {tab === 'gdpr-erasure' && <GDPRErasurePage />}
      {tab === 'dsar' && <DSARSearchPage />}
      {tab === 'retention' && <RetentionPolicyPage />}
    </Suspense>
  )
}

export default function SecurityAdminPage() {
  const user = useAuthStore((s) => s.user)
  const isAdmin = user?.roles.includes('admin')
  const [activeTab, setActiveTab] = useState<TabKey>('audit')

  if (!isAdmin) {
    return <Navigate to="/" replace />
  }

  return (
    <div className="flex h-full overflow-hidden">
      {/* Sub-navigation sidebar */}
      <aside className="w-56 shrink-0 border-r border-border bg-card p-4 overflow-y-auto">
        <div className="flex items-center gap-2 px-2 mb-4">
          <ShieldCheck className="h-5 w-5 text-primary" />
          <h3 className="text-sm font-medium text-foreground">Sicherheit</h3>
        </div>
        <nav className="space-y-4">
          {TAB_GROUPS.map((group) => {
            const groupTabs = ALL_TABS.filter((t) => t.group === group)
            return (
              <div key={group}>
                <p className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground px-3 mb-1">
                  {group}
                </p>
                <div className="space-y-0.5">
                  {groupTabs.map((tab) => {
                    const Icon = tab.icon
                    return (
                      <button
                        key={tab.key}
                        onClick={() => setActiveTab(tab.key)}
                        className={`flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-sm transition-colors ${
                          activeTab === tab.key
                            ? 'bg-primary-light text-primary font-medium'
                            : 'text-foreground hover:bg-secondary'
                        }`}
                      >
                        <Icon className="h-4 w-4 shrink-0" />
                        {tab.label}
                      </button>
                    )
                  })}
                </div>
              </div>
            )
          })}
        </nav>
      </aside>

      {/* Content area */}
      <div className="flex-1 overflow-hidden">
        <TabContent tab={activeTab} />
      </div>
    </div>
  )
}
