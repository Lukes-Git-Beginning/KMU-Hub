/**
 * Root application component.
 *
 * Sets up React Query with localStorage persistence, hash-based routing
 * (required for Electron's file:// protocol), and authentication guards.
 */
import { lazy, Suspense } from 'react'
import { createHashRouter, Navigate, RouterProvider } from 'react-router-dom'
import { QueryClient } from '@tanstack/react-query'
import { PersistQueryClientProvider } from '@tanstack/react-query-persist-client'
import { createSyncStoragePersister } from '@tanstack/query-sync-storage-persister'
import { TooltipProvider } from '@/components/ui/tooltip'
import { Toaster } from 'sonner'
import { I18nProvider } from '@/i18n'
import { useAuthStore } from '@/stores/auth'
import { STALE_TIME, GC_TIME } from '@/lib/constants'
import { AppShell } from '@/components/layout/AppShell'
import { ModuleLoadingFallback } from '@/components/layout/ModuleShell'
import LoginPage from '@/modules/auth/LoginPage'

// Lazy-loaded module pages
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

// Security admin pages (lazy-loaded, admin-only)
const AuditLogPage = lazy(() => import('@/modules/security/AuditLogPage'))
const SessionsPage = lazy(() => import('@/modules/security/SessionsPage'))
const VaultPage = lazy(() => import('@/modules/security/VaultPage'))
const PasswordPolicyPage = lazy(() => import('@/modules/security/PasswordPolicyPage'))
const IPAccessPage = lazy(() => import('@/modules/security/IPAccessPage'))
const GDPRExportPage = lazy(() => import('@/modules/security/GDPRExportPage'))
const GDPRErasurePage = lazy(() => import('@/modules/security/GDPRErasurePage'))

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
 */
function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const isLoading = useAuthStore((s) => s.isLoading)

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
        <AppShell />
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
        path: 'video/*',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <VideoPage />
          </Suspense>
        ),
      },
      {
        path: 'meetings/*',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <MeetingsPage />
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
      // Admin security routes
      {
        path: 'admin/security/audit',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <AuditLogPage />
          </Suspense>
        ),
      },
      {
        path: 'admin/security/sessions',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <SessionsPage />
          </Suspense>
        ),
      },
      {
        path: 'admin/security/vault',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <VaultPage />
          </Suspense>
        ),
      },
      {
        path: 'admin/security/password-policy',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <PasswordPolicyPage />
          </Suspense>
        ),
      },
      {
        path: 'admin/security/ip-access',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <IPAccessPage />
          </Suspense>
        ),
      },
      {
        path: 'admin/security/gdpr/exports',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <GDPRExportPage />
          </Suspense>
        ),
      },
      {
        path: 'admin/security/gdpr/erasure',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <GDPRErasurePage />
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
])

export default function App() {
  return (
    <PersistQueryClientProvider
      client={queryClient}
      persistOptions={{ persister, maxAge: GC_TIME }}
    >
      <I18nProvider>
        <TooltipProvider>
          <RouterProvider router={router} />
          <Toaster richColors position="bottom-right" />
        </TooltipProvider>
      </I18nProvider>
    </PersistQueryClientProvider>
  )
}
