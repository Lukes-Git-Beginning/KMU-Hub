/**
 * React Query hooks for dashboard layout management.
 *
 * Provides server-side layout persistence: user overrides, role defaults,
 * and reset-to-defaults. Admin hooks for configuring role default layouts.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/api/client'

/** Query key factory for dashboard-related queries. */
export const dashboardKeys = {
  all: ['dashboard'] as const,
  layout: () => [...dashboardKeys.all, 'layout'] as const,
  defaults: (role: string) => [...dashboardKeys.all, 'defaults', role] as const,
}

/** Fetch the effective dashboard layout for the authenticated user. */
export function useDashboardLayout() {
  return useQuery({
    queryKey: dashboardKeys.layout(),
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/dashboard/layout')
      if (error) throw new Error('Failed to load dashboard layout')
      return data
    },
    // Do not refetch too aggressively -- layout changes come from user actions
    staleTime: 60_000,
  })
}

/** Layout item type matching the OpenAPI schema (object with string keys). */
type LayoutItem = { [key: string]: unknown }

/** Save the user's personal dashboard layout. */
export function useSaveDashboardLayout() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (body: {
      layout: LayoutItem[]
      active_widgets: string[]
    }) => {
      const { data, error } = await apiClient.PUT('/api/v1/dashboard/layout', {
        body,
      })
      if (error) throw new Error('Failed to save dashboard layout')
      return data
    },
    onSuccess: (data) => {
      // Optimistically update the cache
      queryClient.setQueryData(dashboardKeys.layout(), data)
    },
  })
}

/** Reset the user's dashboard to role defaults. */
export function useResetDashboard() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async () => {
      const { data, error } = await apiClient.DELETE('/api/v1/dashboard/layout')
      if (error) throw new Error('Failed to reset dashboard')
      return data
    },
    onSuccess: () => {
      // Invalidate to refetch the new effective layout (role default)
      queryClient.invalidateQueries({ queryKey: dashboardKeys.layout() })
    },
  })
}

/** Fetch the default layout for a specific role (admin only). */
export function useDashboardDefaults(role: string) {
  return useQuery({
    queryKey: dashboardKeys.defaults(role),
    queryFn: async () => {
      const { data, error } = await apiClient.GET(
        '/api/v1/dashboard/defaults/{role}',
        { params: { path: { role: role as 'admin' | 'manager' | 'member' } } }
      )
      if (error) throw new Error('Failed to load dashboard defaults')
      return data
    },
    enabled: !!role,
  })
}

/** Save the default layout for a role (admin only). */
export function useSaveDashboardDefaults() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({
      role,
      layout,
      active_widgets,
    }: {
      role: string
      layout: LayoutItem[]
      active_widgets: string[]
    }) => {
      const { data, error } = await apiClient.PUT(
        '/api/v1/dashboard/defaults/{role}',
        {
          params: { path: { role: role as 'admin' | 'manager' | 'member' } },
          body: { layout, active_widgets },
        }
      )
      if (error) throw new Error('Failed to save dashboard defaults')
      return data
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: dashboardKeys.defaults(variables.role),
      })
    },
  })
}
