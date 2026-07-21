/**
 * Root application component.
 *
 * Sets up React Query with localStorage persistence, hash-based routing
 * (required for Electron's file:// protocol), and authentication guards.
 */
import { lazy, Suspense, useEffect } from 'react'
import { createHashRouter, Navigate, RouterProvider } from 'react-router-dom'
import { QueryClient } from '@tanstack/react-query'
import { PersistQueryClientProvider } from '@tanstack/react-query-persist-client'
import { createAsyncStoragePersister } from '@tanstack/query-async-storage-persister'
import { get, set, del } from 'idb-keyval'
import { TooltipProvider } from '@/components/ui/tooltip'
import { Toaster } from 'sonner'
import { I18nProvider } from '@/i18n'
import { useAuthStore } from '@/stores/auth'
import { STALE_TIME, GC_TIME } from '@/lib/constants'
import { DeskEnvironment } from '@/components/layout/DeskEnvironment'
import { ModuleLoadingFallback } from '@/components/layout/ModuleShell'
import type { ModuleSkeletonVariant } from '@/components/shared'
import LoginPage from '@/modules/auth/LoginPage'
import RegisterPage from '@/modules/auth/RegisterPage'
import ForgotPasswordPage from '@/modules/auth/ForgotPasswordPage'
import ResetPasswordPage from '@/modules/auth/ResetPasswordPage'
import { IdleLock } from '@/modules/auth/IdleLock'
import { LaunchOverlay } from '@/modules/auth/LaunchOverlay'
import { useLaunchStore } from '@/stores/launch'
import { DEMO_PROFILES } from '@/mocks/data/rbac'
import { ProfileSwitcher } from '@/components/dev/ProfileSwitcher'
import { PermissionPreviewBanner } from '@/components/layout/PermissionPreviewBanner'
import { ViewAsBanner } from '@/components/layout/ViewAsBanner'
import NotificationToast from '@/modules/notifications/NotificationToast'
import { OfflineBanner } from '@/components/ui/OfflineBanner'
import { NoAccessView } from '@/components/shared/rbac/NoAccessView'
import { useCapability } from '@/hooks/useCapability'
import { moduleViewKey, type ModuleKey } from '@/config/capabilities'

// Lazy-loaded module pages — existing (backend-connected)
const DashboardPage = lazy(() => import('@/modules/dashboard/DashboardPage'))

const WorkLayout = lazy(() => import('@/modules/work/WorkLayout'))
const KalenderPage = lazy(() => import('@/modules/kalender/KalenderPage'))
const VideoPage = lazy(() => import('@/modules/video/VideoPage'))
const MeetingsPage = lazy(() => import('@/modules/meetings/MeetingsPage'))
const NotificationCenter = lazy(() => import('@/modules/notifications/NotificationCenter'))
const DashboardSettings = lazy(() => import('@/modules/settings/DashboardSettings'))
const SettingsPage = lazy(() => import('@/modules/settings/SettingsPage'))


// Admin Hub — neue Routen /admin/it | /admin/security | /admin/billing | /admin/integrations
const AdminHubPage = lazy(() => import('@/modules/admin/AdminHubPage'))

// Role builder editor (R-2) — full page under /admin/roles/:roleId
const RoleEditorPage = lazy(() => import('@/modules/admin/roles/RoleEditorPage'))

// Per-user override editor (R-6) — full page under /admin/users/:userId/overrides
const UserOverrideEditorPage = lazy(() => import('@/modules/admin/users/UserOverrideEditorPage'))

// (AnpassungenHubPage entfernt — Anpassungen laufen jetzt als 9. AdminHubPage-Tab)

// CalDAV admin page
const CalDAVAdminPage = lazy(() => import('@/modules/admin/CalDAVAdminPage'))

// Plugin admin page
const PluginListPage = lazy(() => import('@/modules/admin/plugins/PluginListPage'))

// New module pages from design integration (mock data, Zustand stores)
const KontakteLayout = lazy(() => import('@/modules/kontakte/KontakteLayout'))
const KontaktePage = lazy(() => import('@/modules/kontakte/KontaktePage'))
const CompaniesListPage = lazy(() => import('@/modules/crm/companies/CompaniesListPage'))
const CompanyDetailPage = lazy(() => import('@/modules/crm/companies/CompanyDetailPage'))
const DealsListPage = lazy(() => import('@/modules/crm/deals/DealsListPage'))
const LeadsInboxPage = lazy(() => import('@/modules/kontakte/leads/LeadsInboxPage'))
const AuswertungenPage = lazy(() => import('@/modules/kontakte/AuswertungenPage'))
const AdvisoryProtocolEditor = lazy(() => import('@/modules/kontakte/advisory/AdvisoryProtocolEditor'))
const DealDetailPage = lazy(() => import('@/modules/crm/deals/DealDetailPage'))
const ActivitiesListPage = lazy(() => import('@/modules/crm/activities/ActivitiesListPage'))
const DokumentePage = lazy(() => import('@/modules/dokumente/DokumentePage'))
const MailsPage = lazy(() => import('@/modules/mails/MailsPage'))
const TeamPage = lazy(() => import('@/modules/team/TeamPage'))
const MemberDetailPage = lazy(() => import('@/modules/team/MemberDetailPage'))
const FinanzenPage = lazy(() => import('@/modules/finanzen/FinanzenPage'))
const InfrastrukturPage = lazy(() => import('@/modules/admin/InfrastrukturPage'))
const ProfilPage = lazy(() => import('@/modules/profil/ProfilPage'))
const ComposeWindowPage = lazy(() => import('@/modules/mails/ComposeWindowPage'))
const EmployeeWizardWindowPage = lazy(() => import('@/modules/team/EmployeeWizardWindowPage'))
const KommunikationPage = lazy(() => import('@/modules/kommunikation/KommunikationPage'))
const AutomatisierungPage = lazy(() => import('@/modules/automatisierung/AutomatisierungPage'))

// Industry-specific module pages
const InventarPage = lazy(() => import('@/modules/inventar/InventarPage'))
const SchichtenPage = lazy(() => import('@/modules/schichten/SchichtenPage'))
const EinkaufPage = lazy(() => import('@/modules/einkauf/EinkaufPage'))
const HelpdeskPage = lazy(() => import('@/modules/helpdesk/HelpdeskPage'))
const FuhrparkPage = lazy(() => import('@/modules/fuhrpark/FuhrparkPage'))
const ProduktionPage = lazy(() => import('@/modules/produktion/ProduktionPage'))
const BerichtePage = lazy(() => import('@/modules/berichte/BerichtePage'))
const VertraegePage = lazy(() => import('@/modules/vertraege/VertraegePage'))
const FormularePage = lazy(() => import('@/modules/formulare/FormularePage'))
const VermietungPage = lazy(() => import('@/modules/vermietung/VermietungPage'))
const RapportePage = lazy(() => import('@/modules/rapporte/RapportePage'))
const ZeiterfassungPage = lazy(() => import('@/modules/zeiterfassung/ZeiterfassungPage'))
const WikiPage = lazy(() => import('@/modules/wiki/WikiPage'))
const DialerLayout = lazy(() => import('@/modules/dialer/DialerLayout'))

// React Query client with offline-friendly defaults
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: STALE_TIME,
      gcTime: GC_TIME,
      retry: 2,
      refetchOnWindowFocus: true,
    },
  },
})

// Expose queryClient for Playwright QA scripts in dev/test environments.
// Tree-shaken in production builds (import.meta.env.DEV = false).
if (import.meta.env.DEV) {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  ;(window as any).__cosmi_queryClient__ = queryClient
}

// Persist query cache to IndexedDB for offline support.
// IndexedDB has no practical size limit and avoids blocking the main thread.
const persister = createAsyncStoragePersister({
  storage: {
    getItem: (key) => get(key),
    setItem: (key, value) => set(key, value),
    removeItem: (key) => del(key),
  },
  key: 'cosmi-query-cache',
})

/**
 * Redirect to login if user is not authenticated.
 *
 * DEV_BYPASS_AUTH: Automatically enabled in dev mode (npm run dev),
 * disabled in production builds. Opt out within a dev build by setting
 * RENDERER_VITE_DEV_BYPASS_AUTH=false (e.g. mode `localbackend`) to force the
 * real login screen against a live backend — used for the mock-exit workflow.
 */
const DEV_BYPASS_AUTH =
  import.meta.env.DEV && import.meta.env.RENDERER_VITE_DEV_BYPASS_AUTH !== 'false'

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const isLoading = useAuthStore((s) => s.isLoading)

  if (DEV_BYPASS_AUTH) {
    return <>{children}</>
  }

  if (isLoading) {
    return <ModuleLoadingFallback />
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  return <>{children}</>
}

/**
 * Redirect to app if user is already authenticated.
 */
function GuestRoute({ children }: { children: React.ReactNode }) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const isLoading = useAuthStore((s) => s.isLoading)

  if (isLoading) {
    return <ModuleLoadingFallback />
  }

  if (isAuthenticated) {
    return <Navigate to="/" replace />
  }

  return <>{children}</>
}

/**
 * Deep-link guard (R-3): a module route without level-1 visibility renders the
 * shared "Kein Zugriff" page instead of a blank screen or silent redirect.
 * While the first permission load is in flight we keep the skeleton — the
 * NoAccess page must never flash for a role that does have access.
 */
function ModuleGate({
  module,
  variant,
  children,
}: {
  module: ModuleKey
  variant: ModuleSkeletonVariant
  children: React.ReactNode
}) {
  const { allowed, isLoading } = useCapability(moduleViewKey(module))
  if (!allowed) {
    return isLoading ? <ModuleLoadingFallback variant={variant} /> : <NoAccessView module={module} />
  }
  return <>{children}</>
}

/** Helper to wrap a lazy page in Suspense. The `variant` picks a loading-skeleton
 *  archetype that matches the module's layout (list = default). `module` adds the
 *  level-1 deep-link guard (NoAccessView when the role lacks `<module>:module:view`). */
function lazyRoute(
  Component: React.LazyExoticComponent<() => React.JSX.Element | null>,
  variant: ModuleSkeletonVariant = 'list',
  module?: ModuleKey,
) {
  const page = (
    <Suspense fallback={<ModuleLoadingFallback variant={variant} />}>
      <Component />
    </Suspense>
  )
  if (!module) return page
  return (
    <ModuleGate module={module} variant={variant}>
      {page}
    </ModuleGate>
  )
}

/**
 * Dev-/QA-only: löst die geteilte Fly-in-/Öffnungs-Animation (Fall B) isoliert
 * über einem repräsentativen Dashboard-Hintergrund aus, damit der Übergang ohne
 * echten Login screenshot-bar ist. In Produktions-Builds nicht registriert.
 */
function FlyinPreview() {
  const startFlyin = useLaunchStore((s) => s.startFlyin)
  useEffect(() => {
    startFlyin()
  }, [startFlyin])
  return (
    <div className="flex h-screen w-screen items-center justify-center bg-background text-muted-foreground">
      <span className="text-sm">Dashboard-Hintergrund (Fly-in-Preview)</span>
    </div>
  )
}

// Hash router -- Electron loads from file:// in production, which
// breaks HTML5 history API. Hash routing (#/crm, #/chat) works everywhere.
/** Catch-all error boundary so a failed lazy-load or bad route doesn't blank the screen */
function RouteErrorFallback() {
  return (
    <div className="flex h-full items-center justify-center">
      <div className="text-center space-y-4 max-w-md">
        <p className="text-lg font-medium text-foreground">Seite nicht gefunden</p>
        <p className="text-sm text-muted-foreground">Die angeforderte Seite existiert nicht oder konnte nicht geladen werden.</p>
        <a href="#/" className="inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors">
          Zum Dashboard
        </a>
      </div>
    </div>
  )
}

const router = createHashRouter([
  {
    path: '/',
    element: (
      <ProtectedRoute>
        <DeskEnvironment />
      </ProtectedRoute>
    ),
    errorElement: <RouteErrorFallback />,
    children: [
      // Core modules (backend-connected)
      { index: true, element: lazyRoute(DashboardPage, 'dashboard', 'dashboard') },
      { path: 'crm/*', element: <Navigate to="/kontakte" replace /> },
      { path: 'chat/*', element: <Navigate to="/kommunikation?bereich=team" replace /> },
      { path: 'work/*', element: lazyRoute(WorkLayout, 'board', 'work') },
      { path: 'kalender', element: lazyRoute(KalenderPage, 'calendar', 'kalender') },
      { path: 'video/*', element: lazyRoute(VideoPage, 'video', 'video') },
      { path: 'meetings/*', element: lazyRoute(MeetingsPage, 'meetings', 'video') },
      { path: 'notifications', element: lazyRoute(NotificationCenter, 'notifications', 'notifications') },
      { path: 'settings/dashboard', element: lazyRoute(DashboardSettings, 'detail') },
      { path: 'settings', element: lazyRoute(SettingsPage, 'detail') },

      // Admin security hub — legacy path now redirects to the consolidated hub
      // (SecurityAdminPage retained as component, no longer routed). No dead deep-links.
      { path: 'admin/security-legacy', element: <Navigate to="/admin/security" replace /> },

      // Admin Hub — neue /admin/* Routen
      { path: 'admin/users', element: lazyRoute(AdminHubPage, 'adminhub', 'admin') },
      { path: 'admin/users/:userId/overrides', element: lazyRoute(UserOverrideEditorPage, 'adminhub', 'admin') },
      { path: 'admin/roles', element: lazyRoute(AdminHubPage, 'adminhub', 'admin') },
      { path: 'admin/roles/:roleId', element: lazyRoute(RoleEditorPage, 'adminhub', 'admin') },
      { path: 'admin/license', element: lazyRoute(AdminHubPage, 'adminhub', 'admin') },
      { path: 'admin/branding', element: lazyRoute(AdminHubPage, 'adminhub', 'admin') },
      { path: 'admin/it', element: lazyRoute(AdminHubPage, 'adminhub', 'admin') },
      { path: 'admin/security', element: lazyRoute(AdminHubPage, 'adminhub', 'admin') },
      { path: 'admin/billing', element: lazyRoute(AdminHubPage, 'adminhub', 'admin') },
      { path: 'admin/integrations', element: lazyRoute(AdminHubPage, 'adminhub', 'admin') },
      // Backwards-Compat: /einstellungen?tab=it-admin → /admin/it (via Settings redirect)
      // Backwards-Compat: /einstellungen?tab=billing → /admin/billing (via Settings redirect)

      // Anpassungen-Hub (v1.1) — 9. AdminHubPage-Tab (Anpassungen)
      // /admin/anpassungen und /admin/anpassungen/felder laufen jetzt über AdminHubPage
      { path: 'admin/anpassungen', element: lazyRoute(AdminHubPage, 'adminhub', 'admin') },
      { path: 'admin/anpassungen/felder', element: lazyRoute(AdminHubPage, 'adminhub', 'admin') },

      // CalDAV admin
      { path: 'admin/caldav', element: lazyRoute(CalDAVAdminPage, 'list', 'admin') },

      // Plugin admin
      { path: 'admin/plugins', element: lazyRoute(PluginListPage, 'list', 'admin') },

      // Kontakte — Kunden-Zentrale (Kontakte + Firmen + Pipeline + Aktivitäten)
      {
        path: 'kontakte',
        element: lazyRoute(KontakteLayout, 'kontakte', 'crm'),
        children: [
          { index: true, element: lazyRoute(KontaktePage, 'kontakte') },
          { path: 'firmen', element: lazyRoute(CompaniesListPage, 'kontakte') },
          { path: 'firmen/:id', element: lazyRoute(CompanyDetailPage, 'detail') },
          { path: 'leads', element: lazyRoute(LeadsInboxPage, 'leads') },
          { path: 'pipeline', element: lazyRoute(DealsListPage, 'pipeline') },
          { path: 'pipeline/:id', element: lazyRoute(DealDetailPage, 'detail') },
          { path: 'aktivitaeten', element: lazyRoute(ActivitiesListPage, 'aktivitaeten') },
          { path: 'auswertungen', element: lazyRoute(AuswertungenPage, 'dashboard') },
          { path: 'protokoll/:contactId/:protocolId', element: lazyRoute(AdvisoryProtocolEditor, 'detail') },
        ],
      },
      { path: 'dokumente', element: lazyRoute(DokumentePage, 'dokumente', 'documents') },
      { path: 'mails', element: lazyRoute(MailsPage, 'mails', 'mail') },
      { path: 'kommunikation', element: lazyRoute(KommunikationPage, 'kommunikation', 'kommunikation') },
      { path: 'automatisierung', element: lazyRoute(AutomatisierungPage, 'automatisierung', 'automatisierung') },
      { path: 'team', element: lazyRoute(TeamPage, 'team', 'team') },
      { path: 'team/member/:id', element: lazyRoute(MemberDetailPage, 'detail', 'team') },
      { path: 'finanzen', element: lazyRoute(FinanzenPage, 'finanzen', 'finance') },
      { path: 'infrastruktur', element: lazyRoute(InfrastrukturPage, 'infrastruktur', 'infrastructure') },
      { path: 'profil', element: lazyRoute(ProfilPage, 'detail') },

      // Industry-specific modules
      { path: 'inventar', element: lazyRoute(InventarPage, 'inventar', 'inventar') },
      { path: 'schichten', element: lazyRoute(SchichtenPage, 'calendar', 'schichten') },
      { path: 'einkauf', element: lazyRoute(EinkaufPage, 'einkauf', 'einkauf') },
      { path: 'helpdesk', element: lazyRoute(HelpdeskPage, 'helpdesk', 'helpdesk') },
      { path: 'fuhrpark', element: lazyRoute(FuhrparkPage, 'fuhrpark', 'fuhrpark') },
      { path: 'produktion', element: lazyRoute(ProduktionPage, 'produktion', 'produktion') },
      { path: 'berichte', element: lazyRoute(BerichtePage, 'dashboard', 'berichte') },
      { path: 'vertraege', element: lazyRoute(VertraegePage, 'vertraege', 'vertraege') },
      { path: 'formulare', element: lazyRoute(FormularePage, 'formulare', 'formulare') },
      { path: 'vermietung', element: lazyRoute(VermietungPage, 'vermietung', 'vermietung') },
      { path: 'rapporte', element: lazyRoute(RapportePage, 'rapporte', 'rapporte') },
      { path: 'zeiterfassung', element: lazyRoute(ZeiterfassungPage, 'zeiterfassung', 'zeiterfassung') },
      { path: 'wiki', element: lazyRoute(WikiPage, 'wiki', 'wiki') },
      { path: 'dialer/*', element: lazyRoute(DialerLayout, 'dialer', 'dialer') },
    ],
  },
  {
    path: '/login',
    element: (
      <GuestRoute>
        <LoginPage />
      </GuestRoute>
    ),
  },
  {
    path: '/register',
    element: (
      <GuestRoute>
        <RegisterPage />
      </GuestRoute>
    ),
  },
  {
    // No GuestRoute: the reset link is an email landing reachable regardless of
    // session, and the dev auto-login must not redirect these away during QA.
    path: '/forgot-password',
    element: <ForgotPasswordPage />,
  },
  {
    path: '/reset-password',
    element: <ResetPasswordPage />,
  },
  // Dev-/QA-only: Login-Screen ohne GuestRoute ansehbar (DEV_BYPASS_AUTH leitet
  // /login sonst auf / um). In Produktions-Builds nicht registriert.
  ...(import.meta.env.DEV
    ? [
        { path: '/launch-preview', element: <LoginPage /> },
        // Fly-in-/Öffnungs-Animation (Fall B) isoliert ansehen/screenshotten.
        { path: '/flyin-preview', element: <FlyinPreview /> },
      ]
    : []),
  {
    path: '/compose-window',
    element: lazyRoute(ComposeWindowPage),
  },
  {
    path: '/employee-wizard-window',
    element: lazyRoute(EmployeeWizardWindowPage),
  },
])

export default function App() {
  // Set a default mock user when DEV_BYPASS_AUTH is active so role-based
  // filtering works immediately. The ProfileSwitcher lets Darien switch roles.
  useEffect(() => {
    if (DEV_BYPASS_AUTH) {
      const { user } = useAuthStore.getState()
      if (!user) {
        const adminProfile = DEMO_PROFILES.find((p) => p.roleId === 'admin')
        if (adminProfile) {
          useAuthStore.setState({
            user: adminProfile.user,
            isAuthenticated: true,
          })
        }
      }
    }
  }, [])

  return (
    <PersistQueryClientProvider
      client={queryClient}
      persistOptions={{ persister, maxAge: GC_TIME }}
    >
      <I18nProvider>
        <TooltipProvider>
          <RouterProvider router={router} />
          {/* App-weites Öffnungs-/Flug-Overlay — maskiert den Suspense-Flash
              beim Übergang ins Dashboard (Fall A Startup + Fall B post-login). */}
          <LaunchOverlay />
          <Toaster richColors position="bottom-right" closeButton />
          <NotificationToast />
          <OfflineBanner />
          <PermissionPreviewBanner />
          <ViewAsBanner />
          <IdleLock />
          {/* Dev profile switcher — re-enabled for the RBAC R-block: switching
              presets (incl. Aushilfe/Extern + multi-role combo) is the primary
              review artefact for capability gating. */}
          {DEV_BYPASS_AUTH && <ProfileSwitcher />}
        </TooltipProvider>
      </I18nProvider>
    </PersistQueryClientProvider>
  )
}
