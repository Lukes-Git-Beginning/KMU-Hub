/**
 * AdminHubPage — Tenant-weite Verwaltung.
 *
 * 8 Tabs (users/roles/license/branding/it/security/billing/integrations),
 * Tab-Auswahl mit der URL synchronisiert (/admin/<tab>).
 *
 * Gated: hub entry via `admin:module:view`, each tab additionally via
 * TAB_CAPABILITY (R-3 batch 2) — a role only sees the tabs it may manage;
 * direct URLs to hidden tabs redirect to the first allowed one.
 */
import { useEffect, useRef, lazy, Suspense } from 'react'
import { useNavigate, useLocation, Navigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ModuleLoadingFallback } from '@/components/layout/ModuleShell'
import { useCapabilitySet } from '@/hooks/useCapability'
import { moduleViewKey } from '@/config/capabilities'

const UsersAdminHubTab = lazy(() => import('./users/UsersAdminHubTab'))
const RolesBuilderTab = lazy(() => import('./roles/RolesBuilderTab'))
const LicenseAdminHubTab = lazy(() => import('./license/LicenseAdminHubTab'))
const BrandingAdminHubTab = lazy(() => import('./branding/BrandingAdminHubTab'))
const ITAdminHubTab = lazy(() => import('./tabs/ITAdminHubTab'))
const SecurityAdminHubTab = lazy(() => import('./tabs/SecurityAdminHubTab'))
const BillingAdminHubTab = lazy(() => import('./tabs/BillingAdminHubTab'))
const IntegrationsAdminHubTab = lazy(() => import('./tabs/IntegrationsAdminHubTab'))
const AnpassungenTab = lazy(() => import('./anpassungen/AnpassungenHubPage'))

type AdminTab = 'users' | 'roles' | 'license' | 'branding' | 'it' | 'security' | 'billing' | 'integrations' | 'anpassungen'

const ROUTE_TO_TAB: Record<string, AdminTab> = {
  '/admin/users': 'users',
  '/admin/roles': 'roles',
  '/admin/license': 'license',
  '/admin/branding': 'branding',
  '/admin/it': 'it',
  '/admin/security': 'security',
  '/admin/billing': 'billing',
  '/admin/integrations': 'integrations',
  '/admin/anpassungen': 'anpassungen',
  '/admin/anpassungen/felder': 'anpassungen',
  '/admin/anpassungen/begriffe': 'anpassungen',
}

const TAB_TO_ROUTE: Record<AdminTab, string> = {
  users: '/admin/users',
  roles: '/admin/roles',
  license: '/admin/license',
  branding: '/admin/branding',
  it: '/admin/it',
  security: '/admin/security',
  billing: '/admin/billing',
  integrations: '/admin/integrations',
  anpassungen: '/admin/anpassungen',
}

/** Tab capability map — undefined means "no per-tab gate beyond module access". */
const TAB_CAPABILITY: Partial<Record<AdminTab, string>> = {
  users: 'admin:user:read',
  roles: 'admin:role:read',
  license: 'admin:license:manage',
  branding: 'admin:branding:manage',
  it: 'admin:it:manage',
  security: moduleViewKey('security'),
  billing: 'admin:license:manage',
  integrations: 'admin:integrations:manage',
  anpassungen: 'admin:customization:manage',
}

export default function AdminHubPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const location = useLocation()
  const contentRef = useRef<HTMLDivElement>(null)

  const { has, ready } = useCapabilitySet()

  const canAccessModule = has('admin:module:view')

  // Derive active tab from current pathname
  const requestedTab: AdminTab = ROUTE_TO_TAB[location.pathname] ?? 'it'

  // All tab definitions
  const ALL_TABS: { key: AdminTab; labelKey: string }[] = [
    { key: 'users', labelKey: 'admin.hub.tabs.users' },
    { key: 'roles', labelKey: 'admin.hub.tabs.roles' },
    { key: 'license', labelKey: 'admin.hub.tabs.license' },
    { key: 'branding', labelKey: 'admin.hub.tabs.branding' },
    { key: 'it', labelKey: 'admin.hub.tabs.it' },
    { key: 'security', labelKey: 'admin.hub.tabs.security' },
    { key: 'billing', labelKey: 'admin.hub.tabs.billing' },
    { key: 'integrations', labelKey: 'admin.hub.tabs.integrations' },
    { key: 'anpassungen', labelKey: 'admin.hub.tabs.customization' },
  ]

  // Filter to only tabs the user may see (pessimistic until ready)
  const tabs = ready
    ? ALL_TABS.filter((tab) => {
        const cap = TAB_CAPABILITY[tab.key]
        return cap ? has(cap) : true
      })
    : []

  // Move focus to content region on tab switch (hook must be top-level, before early returns)
  useEffect(() => {
    if (ready) contentRef.current?.focus()
  }, [requestedTab, ready])

  // If not ready yet, do nothing — avoid redirect flicker
  if (!ready) {
    return <ModuleLoadingFallback />
  }

  if (!canAccessModule) {
    return <Navigate to="/" replace />
  }

  // If the requested tab is not in the allowed list, redirect to first allowed
  const firstAllowed = tabs[0]
  const isRequestedAllowed = tabs.some((tab) => tab.key === requestedTab)
  if (!isRequestedAllowed) {
    if (!firstAllowed) {
      return <Navigate to="/" replace />
    }
    return <Navigate to={TAB_TO_ROUTE[firstAllowed.key]} replace />
  }

  const activeTab = requestedTab

  const handleTabChange = (tab: AdminTab) => {
    navigate(TAB_TO_ROUTE[tab])
  }

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
          {activeTab === 'roles' && <RolesBuilderTab />}
          {activeTab === 'license' && <LicenseAdminHubTab />}
          {activeTab === 'branding' && <BrandingAdminHubTab />}
          {activeTab === 'it' && <ITAdminHubTab />}
          {activeTab === 'security' && <SecurityAdminHubTab />}
          {activeTab === 'billing' && <BillingAdminHubTab />}
          {activeTab === 'integrations' && <IntegrationsAdminHubTab />}
          {activeTab === 'anpassungen' && <AnpassungenTab />}
        </Suspense>
      </div>
    </div>
  )
}
