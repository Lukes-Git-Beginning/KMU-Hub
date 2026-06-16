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
import LoginPage from '@/modules/auth/LoginPage'
import RegisterPage from '@/modules/auth/RegisterPage'
import ForgotPasswordPage from '@/modules/auth/ForgotPasswordPage'
import ResetPasswordPage from '@/modules/auth/ResetPasswordPage'
import { DEV_PROFILES } from '@/config/roles'
import NotificationToast from '@/modules/notifications/NotificationToast'
import { OfflineBanner } from '@/components/ui/OfflineBanner'

// Lazy-loaded module pages — existing (backend-connected)
const DashboardPage = lazy(() => import('@/modules/dashboard/DashboardPage'))

const WorkLayout = lazy(() => import('@/modules/work/WorkLayout'))
const KalenderPage = lazy(() => import('@/modules/kalender/KalenderPage'))
const VideoPage = lazy(() => import('@/modules/video/VideoPage'))
const MeetingsPage = lazy(() => import('@/modules/meetings/MeetingsPage'))
const NotificationCenter = lazy(() => import('@/modules/notifications/NotificationCenter'))
const DashboardSettings = lazy(() => import('@/modules/settings/DashboardSettings'))
const SettingsPage = lazy(() => import('@/modules/settings/SettingsPage'))

// Security admin hub (lazy-loaded, admin-only) — legacy, ersetzt durch AdminHubPage
const SecurityAdminPage = lazy(() => import('@/modules/security/SecurityAdminPage'))

// Admin Hub — neue Routen /admin/it | /admin/security | /admin/billing | /admin/integrations
const AdminHubPage = lazy(() => import('@/modules/admin/AdminHubPage'))

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
 * disabled in production builds. No manual toggle needed.
 */
const DEV_BYPASS_AUTH = import.meta.env.DEV

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

/** Helper to wrap a lazy page in Suspense */
function lazyRoute(Component: React.LazyExoticComponent<() => JSX.Element>) {
  return (
    <Suspense fallback={<ModuleLoadingFallback />}>
      <Component />
    </Suspense>
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
      { index: true, element: lazyRoute(DashboardPage) },
      { path: 'crm/*', element: <Navigate to="/kontakte" replace /> },
      { path: 'chat/*', element: <Navigate to="/kommunikation?bereich=team" replace /> },
      { path: 'work/*', element: lazyRoute(WorkLayout) },
      { path: 'kalender', element: lazyRoute(KalenderPage) },
      { path: 'video/*', element: lazyRoute(VideoPage) },
      { path: 'meetings/*', element: lazyRoute(MeetingsPage) },
      { path: 'notifications', element: lazyRoute(NotificationCenter) },
      { path: 'settings/dashboard', element: lazyRoute(DashboardSettings) },
      { path: 'settings', element: lazyRoute(SettingsPage) },

      // Admin security hub — legacy redirect (Backwards-Compat, wird in Phase 4 entfernt)
      { path: 'admin/security-legacy', element: lazyRoute(SecurityAdminPage) },

      // Admin Hub — neue /admin/* Routen
      { path: 'admin/it', element: lazyRoute(AdminHubPage) },
      { path: 'admin/security', element: lazyRoute(AdminHubPage) },
      { path: 'admin/billing', element: lazyRoute(AdminHubPage) },
      { path: 'admin/integrations', element: lazyRoute(AdminHubPage) },
      // Backwards-Compat: /einstellungen?tab=it-admin → /admin/it (via Settings redirect)
      // Backwards-Compat: /einstellungen?tab=billing → /admin/billing (via Settings redirect)

      // CalDAV admin
      { path: 'admin/caldav', element: lazyRoute(CalDAVAdminPage) },

      // Plugin admin
      { path: 'admin/plugins', element: lazyRoute(PluginListPage) },

      // Kontakte — Kunden-Zentrale (Kontakte + Firmen + Pipeline + Aktivitäten)
      {
        path: 'kontakte',
        element: lazyRoute(KontakteLayout),
        children: [
          { index: true, element: lazyRoute(KontaktePage) },
          { path: 'firmen', element: lazyRoute(CompaniesListPage) },
          { path: 'firmen/:id', element: lazyRoute(CompanyDetailPage) },
          { path: 'leads', element: lazyRoute(LeadsInboxPage) },
          { path: 'pipeline', element: lazyRoute(DealsListPage) },
          { path: 'pipeline/:id', element: lazyRoute(DealDetailPage) },
          { path: 'aktivitaeten', element: lazyRoute(ActivitiesListPage) },
          { path: 'auswertungen', element: lazyRoute(AuswertungenPage) },
          { path: 'protokoll/:contactId/:protocolId', element: lazyRoute(AdvisoryProtocolEditor) },
        ],
      },
      { path: 'dokumente', element: lazyRoute(DokumentePage) },
      { path: 'mails', element: lazyRoute(MailsPage) },
      { path: 'kommunikation', element: lazyRoute(KommunikationPage) },
      { path: 'automatisierung', element: lazyRoute(AutomatisierungPage) },
      { path: 'team', element: lazyRoute(TeamPage) },
      { path: 'team/member/:id', element: lazyRoute(MemberDetailPage) },
      { path: 'finanzen', element: lazyRoute(FinanzenPage) },
      { path: 'infrastruktur', element: lazyRoute(InfrastrukturPage) },
      { path: 'profil', element: lazyRoute(ProfilPage) },

      // Industry-specific modules
      { path: 'inventar', element: lazyRoute(InventarPage) },
      { path: 'schichten', element: lazyRoute(SchichtenPage) },
      { path: 'einkauf', element: lazyRoute(EinkaufPage) },
      { path: 'helpdesk', element: lazyRoute(HelpdeskPage) },
      { path: 'fuhrpark', element: lazyRoute(FuhrparkPage) },
      { path: 'produktion', element: lazyRoute(ProduktionPage) },
      { path: 'berichte', element: lazyRoute(BerichtePage) },
      { path: 'vertraege', element: lazyRoute(VertraegePage) },
      { path: 'formulare', element: lazyRoute(FormularePage) },
      { path: 'vermietung', element: lazyRoute(VermietungPage) },
      { path: 'rapporte', element: lazyRoute(RapportePage) },
      { path: 'zeiterfassung', element: lazyRoute(ZeiterfassungPage) },
      { path: 'wiki', element: lazyRoute(WikiPage) },
      { path: 'dialer/*', element: lazyRoute(DialerLayout) },
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
        const adminProfile = DEV_PROFILES.find((p) => p.id === 'admin')
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
          <Toaster richColors position="bottom-right" closeButton />
          <NotificationToast />
          <OfflineBanner />
          {/* Dev profile switcher temporarily hidden per request (DEV_BYPASS_AUTH still drives the default mock user). */}
        </TooltipProvider>
      </I18nProvider>
    </PersistQueryClientProvider>
  )
}
