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
import { useAuthStore } from '@/stores/auth'
import { STALE_TIME, GC_TIME } from '@/lib/constants'
import { DeskEnvironment } from '@/components/layout/DeskEnvironment'
import { ModuleLoadingFallback } from '@/components/layout/ModuleShell'
import LoginPage from '@/modules/auth/LoginPage'
import { DEV_PROFILES } from '@/config/roles'
import { ProfileSwitcher } from '@/components/dev/ProfileSwitcher'

// Lazy-loaded module pages
const DashboardPage = lazy(() => import('@/modules/dashboard/DashboardPage'))
const CRMLayout = lazy(() => import('@/modules/crm/CRMLayout'))
const ChatLayout = lazy(() => import('@/modules/chat/ChatLayout'))
const WorkLayout = lazy(() => import('@/modules/work/WorkLayout'))
const KalenderPage = lazy(() => import('@/modules/kalender/KalenderPage'))
const NotificationCenter = lazy(() => import('@/modules/notifications/NotificationCenter'))
const DashboardSettings = lazy(() => import('@/modules/settings/DashboardSettings'))
const SettingsPage = lazy(() => import('@/modules/settings/SettingsPage'))
const MeetingsPage = lazy(() => import('@/modules/meetings/MeetingsPage'))
const KontaktePage = lazy(() => import('@/modules/kontakte/KontaktePage'))
const DokumentePage = lazy(() => import('@/modules/dokumente/DokumentePage'))
const MailsPage = lazy(() => import('@/modules/mails/MailsPage'))
const TeamPage = lazy(() => import('@/modules/team/TeamPage'))
const BuchhaltungPage = lazy(() => import('@/modules/buchhaltung/BuchhaltungPage'))
const InfrastrukturPage = lazy(() => import('@/modules/admin/InfrastrukturPage'))
const ProfilPage = lazy(() => import('@/modules/profil/ProfilPage'))
const ComposeWindowPage = lazy(() => import('@/modules/mails/ComposeWindowPage'))

// Industry-specific module pages
const InventarPage = lazy(() => import('@/modules/inventar/InventarPage'))
const SchichtenPage = lazy(() => import('@/modules/schichten/SchichtenPage'))
const EinkaufPage = lazy(() => import('@/modules/einkauf/EinkaufPage'))
const HelpdeskPage = lazy(() => import('@/modules/helpdesk/HelpdeskPage'))
const FuhrparkPage = lazy(() => import('@/modules/fuhrpark/FuhrparkPage'))
const ProduktionPage = lazy(() => import('@/modules/produktion/ProduktionPage'))
const BerichtePage = lazy(() => import('@/modules/berichte/BerichtePage'))

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
 * Used as a route element wrapper for protected routes.
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
 * Used on the login page to prevent seeing login when already logged in.
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

// Hash router -- Electron loads from file:// in production, which
// breaks HTML5 history API. Hash routing (#/crm, #/chat) works everywhere.
const router = createHashRouter([
  {
    path: '/',
    element: (
      <ProtectedRoute>
        <DeskEnvironment />
      </ProtectedRoute>
    ),
    children: [
      {
        index: true,
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <DashboardPage />
          </Suspense>
        ),
      },
      {
        path: 'crm/*',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <CRMLayout />
          </Suspense>
        ),
      },
      {
        path: 'chat/*',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <ChatLayout />
          </Suspense>
        ),
      },
      {
        path: 'work/*',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <WorkLayout />
          </Suspense>
        ),
      },
      {
        path: 'calendar',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <KalenderPage />
          </Suspense>
        ),
      },
      {
        path: 'notifications',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <NotificationCenter />
          </Suspense>
        ),
      },
      {
        path: 'settings/dashboard',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <DashboardSettings />
          </Suspense>
        ),
      },
      {
        path: 'settings',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <SettingsPage />
          </Suspense>
        ),
      },
      {
        path: 'meetings',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <MeetingsPage />
          </Suspense>
        ),
      },
      {
        path: 'kontakte',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <KontaktePage />
          </Suspense>
        ),
      },
      {
        path: 'dokumente',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <DokumentePage />
          </Suspense>
        ),
      },
      {
        path: 'mails',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <MailsPage />
          </Suspense>
        ),
      },
      {
        path: 'kalender',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <KalenderPage />
          </Suspense>
        ),
      },
      {
        path: 'team',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <TeamPage />
          </Suspense>
        ),
      },
      {
        path: 'buchhaltung',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <BuchhaltungPage />
          </Suspense>
        ),
      },
      {
        path: 'infrastruktur',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <InfrastrukturPage />
          </Suspense>
        ),
      },
      {
        path: 'profil',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <ProfilPage />
          </Suspense>
        ),
      },
      // Industry-specific modules
      {
        path: 'inventar',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <InventarPage />
          </Suspense>
        ),
      },
      {
        path: 'schichten',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <SchichtenPage />
          </Suspense>
        ),
      },
      {
        path: 'einkauf',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <EinkaufPage />
          </Suspense>
        ),
      },
      {
        path: 'helpdesk',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <HelpdeskPage />
          </Suspense>
        ),
      },
      {
        path: 'fuhrpark',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <FuhrparkPage />
          </Suspense>
        ),
      },
      {
        path: 'produktion',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <ProduktionPage />
          </Suspense>
        ),
      },
      {
        path: 'berichte',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <BerichtePage />
          </Suspense>
        ),
      },
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
    element: (
      <Suspense fallback={<ModuleLoadingFallback />}>
        <ComposeWindowPage />
      </Suspense>
    ),
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
      <TooltipProvider>
        <RouterProvider router={router} />
        <Toaster position="bottom-right" richColors closeButton />
        {DEV_BYPASS_AUTH && <ProfileSwitcher />}
      </TooltipProvider>
    </PersistQueryClientProvider>
  )
}
