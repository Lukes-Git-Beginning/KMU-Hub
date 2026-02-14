/**
 * TanStack Query hooks for Calendar CRUD operations.
 *
 * Includes calendar list/detail, members, subscriptions, preferences,
 * and event category management.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { calendarApi } from '../calendar-client'
import type {
  CalendarListResponse,
  CalendarResponse,
  CalendarMembersResponse,
  CalendarPreferencesResponse,
  EventCategoriesResponse,
  CreateCalendarRequest,
  UpdateCalendarRequest,
  CalendarPermission,
} from '../calendar-types'

// ---------------------------------------------------------------------------
// Calendar queries
// ---------------------------------------------------------------------------

/** List the current user's calendars (personal + subscribed). */
export function useCalendars() {
  return useQuery({
    queryKey: ['calendars'],
    queryFn: () => calendarApi.GET<CalendarListResponse>('/api/v1/calendars'),
  })
}

/** List shared calendars available for subscription. */
export function useBrowsableCalendars() {
  return useQuery({
    queryKey: ['calendars', 'browsable'],
    queryFn: () =>
      calendarApi.GET<CalendarListResponse>('/api/v1/calendars/browse'),
  })
}

/** List members of a specific calendar. */
export function useCalendarMembers(calendarId: string) {
  return useQuery({
    queryKey: ['calendars', calendarId, 'members'],
    queryFn: () =>
      calendarApi.GET<CalendarMembersResponse>(
        `/api/v1/calendars/${calendarId}/members`,
      ),
    enabled: !!calendarId,
  })
}

/** Get the user's calendar preferences. */
export function useCalendarPreferences() {
  return useQuery({
    queryKey: ['calendar-preferences'],
    queryFn: () =>
      calendarApi.GET<CalendarPreferencesResponse>(
        '/api/v1/calendars/preferences',
      ),
  })
}

/** List the user's event categories. */
export function useEventCategories() {
  return useQuery({
    queryKey: ['event-categories'],
    queryFn: () =>
      calendarApi.GET<EventCategoriesResponse>('/api/v1/calendars/categories'),
  })
}

// ---------------------------------------------------------------------------
// Calendar mutations
// ---------------------------------------------------------------------------

export function useCreateCalendar() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: CreateCalendarRequest) =>
      calendarApi.POST<CalendarResponse>('/api/v1/calendars', body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['calendars'] })
    },
  })
}

export function useUpdateCalendar() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: UpdateCalendarRequest & { id: string }) =>
      calendarApi.PUT<CalendarResponse>(`/api/v1/calendars/${id}`, body),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ['calendars'] })
      queryClient.invalidateQueries({
        queryKey: ['calendars', variables.id],
      })
    },
  })
}

export function useDeleteCalendar() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => calendarApi.DELETE(`/api/v1/calendars/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['calendars'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Member mutations
// ---------------------------------------------------------------------------

export function useAddCalendarMember() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      calendarId,
      user_id,
      permission,
    }: {
      calendarId: string
      user_id: string
      permission: CalendarPermission
    }) =>
      calendarApi.POST(`/api/v1/calendars/${calendarId}/members`, {
        user_id,
        permission,
      }),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['calendars', variables.calendarId, 'members'],
      })
    },
  })
}

export function useRemoveCalendarMember() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      calendarId,
      userId,
    }: {
      calendarId: string
      userId: string
    }) =>
      calendarApi.DELETE(
        `/api/v1/calendars/${calendarId}/members/${userId}`,
      ),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['calendars', variables.calendarId, 'members'],
      })
    },
  })
}

// ---------------------------------------------------------------------------
// Subscription mutations
// ---------------------------------------------------------------------------

export function useSubscribeToCalendar() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (calendarId: string) =>
      calendarApi.POST(`/api/v1/calendars/${calendarId}/subscribe`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['calendars'] })
      queryClient.invalidateQueries({ queryKey: ['calendars', 'browsable'] })
    },
  })
}

export function useUnsubscribeFromCalendar() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (calendarId: string) =>
      calendarApi.DELETE(`/api/v1/calendars/${calendarId}/subscribe`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['calendars'] })
      queryClient.invalidateQueries({ queryKey: ['calendars', 'browsable'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Preference mutations
// ---------------------------------------------------------------------------

export function useUpdateCalendarPreferences() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: {
      default_view?: 'day' | 'week' | 'month'
      week_days?: 5 | 7
      default_reminder_minutes?: number
      default_allday_reminder_minutes?: number
      subdivision_code?: string | null
      show_task_deadlines?: boolean
    }) => calendarApi.PUT('/api/v1/calendars/preferences', body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['calendar-preferences'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Event category mutations
// ---------------------------------------------------------------------------

export function useCreateEventCategory() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: { name: string; color: string }) =>
      calendarApi.POST('/api/v1/calendars/categories', body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['event-categories'] })
    },
  })
}

export function useDeleteEventCategory() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) =>
      calendarApi.DELETE(`/api/v1/calendars/categories/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['event-categories'] })
    },
  })
}
