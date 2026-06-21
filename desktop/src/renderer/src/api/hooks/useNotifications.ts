/**
 * React Query hooks for the notification system.
 *
 * Provides notification CRUD, unread count tracking, preference management,
 * and real-time WebSocket subscription that triggers native desktop
 * notifications when the app is not focused.
 */
import { useEffect, useMemo } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/api/client'
import { wsManager } from '@/api/websocket'
import i18next from 'i18next'
import type { components } from '@/api/types'

type Notification = components['schemas']['Notification']
type NotificationPreference = components['schemas']['NotificationPreference']
type UpdateNotificationPreferenceRequest = components['schemas']['UpdateNotificationPreferenceRequest']

/** Query key factory for notification-related queries. */
export const notificationKeys = {
  all: ['notifications'] as const,
  lists: () => [...notificationKeys.all, 'list'] as const,
  list: (params?: { page?: number; isRead?: boolean; moduleId?: string }) =>
    [...notificationKeys.lists(), params] as const,
  unreadCount: () => [...notificationKeys.all, 'unread-count'] as const,
  preferences: () => [...notificationKeys.all, 'preferences'] as const,
  eventTypes: () => [...notificationKeys.all, 'event-types'] as const,
}

/** Fetch paginated list of notifications. */
export function useNotifications(params?: {
  page?: number
  pageSize?: number
  isRead?: boolean
  moduleId?: string
  /** Poll interval (ms) — used by the live toast surface to let new notifications "arrive". */
  refetchInterval?: number
}) {
  return useQuery({
    queryKey: notificationKeys.list({
      page: params?.page,
      isRead: params?.isRead,
      moduleId: params?.moduleId,
    }),
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/notifications', {
        params: {
          query: {
            page: params?.page,
            page_size: params?.pageSize,
            is_read: params?.isRead,
            module_id: params?.moduleId,
          },
        },
      })
      if (error) throw new Error('Failed to load notifications')
      return data
    },
    ...(params?.refetchInterval ? { refetchInterval: params.refetchInterval } : {}),
  })
}

/** Fetch unread notification count. */
export function useUnreadNotificationCount() {
  return useQuery({
    queryKey: notificationKeys.unreadCount(),
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/notifications/unread-count')
      if (error) throw new Error('Failed to load unread count')
      return data?.count ?? 0
    },
    // Poll every 30 seconds as fallback to WebSocket
    refetchInterval: 30_000,
  })
}

/**
 * Unread notification counts grouped by module_id — the source for the sidebar
 * per-module badges. Reuses the notifications list query (unread-only).
 */
export function useModuleUnreadCounts(): Record<string, number> {
  const { data } = useNotifications({ isRead: false, pageSize: 100 })
  return useMemo(() => {
    const counts: Record<string, number> = {}
    for (const n of data?.notifications ?? []) {
      if (n.module_id && !n.is_read) counts[n.module_id] = (counts[n.module_id] ?? 0) + 1
    }
    return counts
  }, [data])
}

/** Mark a single notification as read. */
export function useMarkNotificationRead() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (notificationId: string) => {
      const { data, error } = await apiClient.POST('/api/v1/notifications/{id}/read', {
        params: { path: { id: notificationId } },
      })
      if (error) throw new Error('Failed to mark notification as read')
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: notificationKeys.lists() })
      queryClient.invalidateQueries({ queryKey: notificationKeys.unreadCount() })
    },
  })
}

/** Mark all notifications as read, optionally filtered by module. */
export function useMarkAllNotificationsRead() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (moduleId?: string) => {
      const { data, error } = await apiClient.POST('/api/v1/notifications/read-all', {
        body: moduleId ? { module_id: moduleId } : {},
      })
      if (error) throw new Error('Failed to mark all as read')
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: notificationKeys.lists() })
      queryClient.invalidateQueries({ queryKey: notificationKeys.unreadCount() })
    },
  })
}

/** Fetch notification preferences. */
export function useNotificationPreferences(moduleId?: string) {
  return useQuery({
    queryKey: [...notificationKeys.preferences(), moduleId],
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/notifications/preferences', {
        params: {
          query: { module_id: moduleId },
        },
      })
      if (error) throw new Error('Failed to load preferences')
      return data?.preferences ?? []
    },
  })
}

/** Update a notification preference. */
export function useUpdateNotificationPreference() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (request: UpdateNotificationPreferenceRequest) => {
      const { data, error } = await apiClient.PUT('/api/v1/notifications/preferences', {
        body: request,
      })
      if (error) throw new Error('Failed to update preference')
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: notificationKeys.preferences() })
    },
  })
}

/** Fetch available notification event types. */
export function useEventTypes(moduleId?: string) {
  return useQuery({
    queryKey: [...notificationKeys.eventTypes(), moduleId],
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/notifications/event-types', {
        params: {
          query: { module_id: moduleId },
        },
      })
      if (error) throw new Error('Failed to load event types')
      return data?.event_types ?? []
    },
  })
}

/**
 * Subscribe to real-time notification events via WebSocket.
 *
 * When a 'notification.new' event arrives:
 * 1. Invalidates the notification list and unread count caches
 * 2. If the app is not focused, triggers a native desktop notification
 *    via the Electron IPC bridge
 *
 * Call this once in the AppShell or Header to ensure notifications
 * are always active while authenticated.
 */
export function useNotificationWebSocket() {
  const queryClient = useQueryClient()

  useEffect(() => {
    const unsubNew = wsManager.on('notification.new', (data) => {
      // Invalidate caches for fresh data
      queryClient.invalidateQueries({ queryKey: notificationKeys.lists() })
      queryClient.invalidateQueries({ queryKey: notificationKeys.unreadCount() })

      // Trigger native desktop notification if app is not focused
      if (!document.hasFocus()) {
        const title = (data.title as string) || i18next.t('api.notifications.newNotification')
        const body = (data.body as string) || ''
        try {
          window.electronAPI.notifications.show(title, body)
        } catch {
          // Gracefully handle if electronAPI is not available (e.g., in browser dev)
        }
      }
    })

    const unsubCount = wsManager.on('notification.unread_count', (data) => {
      const count = data.count as number | undefined
      if (typeof count === 'number') {
        queryClient.setQueryData(notificationKeys.unreadCount(), count)
      }
    })

    return () => {
      unsubNew()
      unsubCount()
    }
  }, [queryClient])
}

// ---------------------------------------------------------------------------
// Quiet Hours hooks
// ---------------------------------------------------------------------------

import { quietHoursApi, dndApi, mutingApi } from '../notification-client'
import type { QuietHours, MutedResource } from '../notification-client'

export const quietHoursKeys = {
  quietHours: () => ['notifications', 'quiet-hours'] as const,
  dnd: () => ['notifications', 'dnd'] as const,
  mutes: () => ['notifications', 'mutes'] as const,
}

export function useQuietHours() {
  return useQuery({
    queryKey: quietHoursKeys.quietHours(),
    queryFn: () => quietHoursApi.get(),
    staleTime: 5 * 60 * 1000,
    select: (data) => data.quiet_hours,
  })
}

export function useUpdateQuietHours() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: Partial<QuietHours>) => quietHoursApi.update(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: quietHoursKeys.quietHours() })
    },
  })
}

export function useDNDStatus() {
  return useQuery({
    queryKey: quietHoursKeys.dnd(),
    queryFn: () => dndApi.getStatus(),
    refetchInterval: 60_000,
  })
}

export function useEnableDND() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (expiresAt?: string) => dndApi.enable(expiresAt),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: quietHoursKeys.dnd() })
    },
  })
}

export function useDisableDND() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => dndApi.disable(),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: quietHoursKeys.dnd() })
    },
  })
}

export function useMutedResources() {
  return useQuery({
    queryKey: quietHoursKeys.mutes(),
    queryFn: () => mutingApi.list(),
    select: (data) => data.mutes,
  })
}

export function useMuteResource() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ resourceType, resourceId }: { resourceType: string; resourceId: string }) =>
      mutingApi.mute(resourceType, resourceId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: quietHoursKeys.mutes() })
    },
  })
}

export function useUnmuteResource() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (muteId: string) => mutingApi.unmute(muteId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: quietHoursKeys.mutes() })
    },
  })
}

export type { Notification, NotificationPreference, QuietHours, MutedResource }
