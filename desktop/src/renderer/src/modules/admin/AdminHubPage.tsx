/**
 * AdminHubPage — Tenant-weite Verwaltung.
 *
 * 4 Tabs: IT | Sicherheit | Abrechnung | Integrationen
 * Tab-Auswahl wird mit der URL synchronisiert:
 *   /admin/it          → tab=it
 *   /admin/security    → tab=security
 *   /admin/billing     → tab=billing
 *   /admin/integrations → tab=integrations
 *
 * Gated: nur admin / it_support.
 */
import { useEffect, useRef, lazy, Suspense } from 'react'
import { useNavigate, useLocation, Navigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ModuleLoadingFallback } from '@/components/layout/ModuleShell'
import { useAuthStore } from '@/stores/auth'
import { userHasRole } from '@/config/roles'

const UsersAdminHubTab = lazy(() => import('./users/UsersAdminHubTab'))
const ITAdminHubTab = lazy(() => import('./tabs/ITAdminHubTab'))
const SecurityAdminHubTab = lazy(() => import('./tabs/SecurityAdminHubTab'))
const BillingAdminHubTab = lazy(() => import('./tabs/BillingAdminHubTab'))
const IntegrationsAdminHubTab = lazy(() => import('./tabs/IntegrationsAdminHubTab'))

type AdminTab = 'users' | 'it' | 'security' | 'billing' | 'integrations'

const ROUTE_TO_TAB: Record<string, AdminTab> = {
  '/admin/users': 'users',
  '/admin/it': 'it',
  '/admin/security': 'security',
  '/admin/billing': 'billing',
  '/admin/integrations': 'integrations',
}

const TAB_TO_ROUTE: Record<AdminTab, string> = {
  users: '/admin/users',
  it: '/admin/it',
  security: '/admin/security',
  billing: '/admin/billing',
  integrations: '/admin/integrations',
}

export default function AdminHubPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const location = useLocation()
  const user = useAuthStore((s) => s.user)
  const contentRef = useRef<HTMLDivElement>(null)

  const canAccess = userHasRole(user, ['admin', 'it_support'])

  // Derive active tab from current pathname
  const activeTab: AdminTab = ROUTE_TO_TAB[location.pathname] ?? 'it'

  // Move focus to content region on tab switch (no-op when access is denied)
  useEffect(() => {
    if (!canAccess) return
    contentRef.current?.focus()
  }, [activeTab, canAccess])

  if (!canAccess) {
    return <Navigate to="/" replace />
  }

  const handleTabChange = (tab: AdminTab) => {
    navigate(TAB_TO_ROUTE[tab])
  }

  const tabs: { key: AdminTab; labelKey: string }[] = [
    { key: 'users', labelKey: 'admin.hub.tabs.users' },
    { key: 'it', labelKey: 'admin.hub.tabs.it' },
    { key: 'security', labelKey: 'admin.hub.tabs.security' },
    { key: 'billing', labelKey: 'admin.hub.tabs.billing' },
    { key: 'integrations', labelKey: 'admin.hub.tabs.integrations' },
  ]

  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* Page header */}
      <div className="shrink-0 px-6 pt-6 pb-0">
        <h1 className="text-lg font-semibold text-foreground">{t('admin.hub.title')}</h1>
        <p className="text-sm text-muted-foreground mt-0.5">{t('admin.hub.subtitle')}</p>

        {/* Tab bar */}
        <div
          role="tablist"
          aria-label={t('admin.hub.title')}
          className="flex gap-0 mt-5 border-b border-border"
        >
          {tabs.map((tab) => (
            <button
              key={tab.key}
              role="tab"
              aria-selected={activeTab === tab.key}
              aria-controls={`admin-tab-panel-${tab.key}`}
              id={`admin-tab-${tab.key}`}
              onClick={() => handleTabChange(tab.key)}
              className={[
                'px-4 py-2.5 text-sm font-medium border-b-2 -mb-px transition-colors',
                activeTab === tab.key
                  ? 'border-primary text-primary'
                  : 'border-transparent text-muted-foreground hover:text-foreground',
              ].join(' ')}
            >
              {t(tab.labelKey)}
            </button>
          ))}
        </div>
      </div>

      {/* Content area — skip-link target */}
      <div
        ref={contentRef}
        tabIndex={-1}
        role="tabpanel"
        id={`admin-tab-panel-${activeTab}`}
        aria-labelledby={`admin-tab-${activeTab}`}
        className="flex-1 overflow-y-auto outline-none"
      >
        <Suspense fallback={<ModuleLoadingFallback />}>
          {activeTab === 'users' && <UsersAdminHubTab />}
          {activeTab === 'it' && <ITAdminHubTab />}
          {activeTab === 'security' && <SecurityAdminHubTab />}
          {activeTab === 'billing' && <BillingAdminHubTab />}
          {activeTab === 'integrations' && <IntegrationsAdminHubTab />}
        </Suspense>
      </div>
    </div>
  )
}
