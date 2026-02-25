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
import { createSyncStoragePersister } from '@tanstack/query-sync-storage-persister'
import { TooltipProvider } from '@/components/ui/tooltip'
import { Toaster } from 'sonner'
import { I18nProvider } from '@/i18n'
import { useAuthStore } from '@/stores/auth'
import { STALE_TIME, GC_TIME } from '@/lib/constants'
import { DeskEnvironment } from '@/components/layout/DeskEnvironment'
import { ModuleLoadingFallback } from '@/components/layout/ModuleShell'
import LoginPage from '@/modules/auth/LoginPage'
import { DEV_PROFILES } from '@/config/roles'
import { ProfileSwitcher } from '@/components/dev/ProfileSwitcher'
import NotificationToast from '@/modules/notifications/NotificationToast'

// Lazy-loaded module pages — existing (backend-connected)
const DashboardPage = lazy(() => import('@/modules/dashboard/DashboardPage'))
const CRMLayout = lazy(() => import('@/modules/crm/CRMLayout'))
const ChatLayout = lazy(() => import('@/modules/chat/ChatLayout'))
const WorkLayout = lazy(() => import('@/modules/work/WorkLayout'))
const KalenderPage = lazy(() => import('@/modules/kalender/KalenderPage'))
const VideoPage = lazy(() => import('@/modules/video/VideoPage'))
const MeetingsPage = lazy(() => import('@/modules/meetings/MeetingsPage'))
const NotificationCenter = lazy(() => import('@/modules/notifications/NotificationCenter'))
const DashboardSettings = lazy(() => import('@/modules/settings/DashboardSettings'))
const SettingsPage = lazy(() => import('@/modules/settings/SettingsPage'))

// Security admin hub (lazy-loaded, admin-only)
const SecurityAdminPage = lazy(() => import('@/modules/security/SecurityAdminPage'))

// CalDAV admin page
const CalDAVAdminPage = lazy(() => import('@/modules/admin/CalDAVAdminPage'))

// New module pages from design integration (mock data, Zustand stores)
const KontaktePage = lazy(() => import('@/modules/kontakte/KontaktePage'))
const DokumentePage = lazy(() => import('@/modules/dokumente/DokumentePage'))
const MailsPage = lazy(() => import('@/modules/mails/MailsPage'))
const TeamPage = lazy(() => import('@/modules/team/TeamPage'))
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

// Persist query cache to localStorage for offline support.
// localStorage has a ~5-10MB limit; handle QuotaExceededError gracefully.
const persister = createSyncStoragePersister({
  storage: window.localStorage,
  key: 'kmuhub-query-cache',
  serialize: (data) => {
    try {
      return JSON.stringify(data)
    } catch {
      return '{}'
    }
  },
  deserialize: (data) => {
    try {
      return JSON.parse(data)
    } catch {
      return {}
    }
  },
})

/**
 * Redirect to login if user is not authenticated.
 *
 * DEV_BYPASS_AUTH: Set to true to skip auth for design work
 * without a running backend. REMOVE before merging to main.
 */
const DEV_BYPASS_AUTH = true

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
      { path: 'crm/*', element: lazyRoute(CRMLayout) },
      { path: 'chat/*', element: lazyRoute(ChatLayout) },
      { path: 'work/*', element: lazyRoute(WorkLayout) },
      { path: 'kalender', element: lazyRoute(KalenderPage) },
      { path: 'video/*', element: lazyRoute(VideoPage) },
      { path: 'meetings/*', element: lazyRoute(MeetingsPage) },
      { path: 'notifications', element: lazyRoute(NotificationCenter) },
      { path: 'settings/dashboard', element: lazyRoute(DashboardSettings) },
      { path: 'settings', element: lazyRoute(SettingsPage) },

      // Admin security hub
      { path: 'admin/security', element: lazyRoute(SecurityAdminPage) },

      // CalDAV admin
      { path: 'admin/caldav', element: lazyRoute(CalDAVAdminPage) },

      // New modules from design integration
      { path: 'kontakte', element: lazyRoute(KontaktePage) },
      { path: 'dokumente', element: lazyRoute(DokumentePage) },
      { path: 'mails', element: lazyRoute(MailsPage) },
      { path: 'kommunikation', element: lazyRoute(KommunikationPage) },
      { path: 'automatisierung', element: lazyRoute(AutomatisierungPage) },
      { path: 'team', element: lazyRoute(TeamPage) },
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
          {DEV_BYPASS_AUTH && <ProfileSwitcher />}
        </TooltipProvider>
      </I18nProvider>
    </PersistQueryClientProvider>
  )
}
