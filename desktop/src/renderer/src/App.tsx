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
import { useAuthStore } from '@/stores/auth'
import { STALE_TIME, GC_TIME } from '@/lib/constants'
import { AppShell } from '@/components/layout/AppShell'
import { ModuleLoadingFallback } from '@/components/layout/ModuleShell'
import LoginPage from '@/modules/auth/LoginPage'

// Lazy-loaded module pages
const DashboardPage = lazy(() => import('@/modules/dashboard/DashboardPage'))
const CRMLayout = lazy(() => import('@/modules/crm/CRMLayout'))
const ChatLayout = lazy(() => import('@/modules/chat/ChatLayout'))
const NotificationCenter = lazy(() => import('@/modules/notifications/NotificationCenter'))

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

// Persist query cache to localStorage for offline support
const persister = createSyncStoragePersister({
  storage: window.localStorage,
  key: 'kmuhub-query-cache',
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
        path: 'notifications',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <NotificationCenter />
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
      persistOptions={{ persister }}
    >
      <TooltipProvider>
        <RouterProvider router={router} />
      </TooltipProvider>
    </PersistQueryClientProvider>
  )
}
