/**
 * TanStack Query hooks for Calendar Event CRUD operations.
 *
 * Includes event range queries, single event detail, attendees,
 * RSVP, reminders, recurring event edits, and task deadline overlay.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { calendarApi } from '../calendar-client'
import type {
  EventListResponse,
  EventResponse,
  EventAttendeesResponse,
  TaskDeadlinesResponse,
  CreateEventRequest,
  UpdateEventRequest,
  UpdateRecurringEventRequest,
  RSVPStatus,
} from '../calendar-types'

// ---------------------------------------------------------------------------
// Event queries
// ---------------------------------------------------------------------------

/**
 * Fetch expanded events for the given calendars within a time range.
 * Enabled only when at least one calendar ID is provided.
 */
export function useEventsInRange(
  calendarIds: string[],
  start: string,
  end: string,
) {
  return useQuery({
    queryKey: ['events', { calendars: calendarIds, start, end }],
    queryFn: () =>
      calendarApi.GET<EventListResponse>('/api/v1/calendar/events', {
        calendar_ids: calendarIds,
        start,
        end,
      }),
    enabled: calendarIds.length > 0,
  })
}

/** Fetch a single event by ID. */
export function useEvent(eventId: string) {
  return useQuery({
    queryKey: ['events', eventId],
    queryFn: () =>
      calendarApi.GET<EventResponse>(`/api/v1/calendar/events/${eventId}`),
    enabled: !!eventId,
  })
}

/** Fetch attendees for an event. */
export function useEventAttendees(eventId: string) {
  return useQuery({
    queryKey: ['events', eventId, 'attendees'],
    queryFn: () =>
      calendarApi.GET<EventAttendeesResponse>(
        `/api/v1/calendar/events/${eventId}/attendees`,
      ),
    enabled: !!eventId,
  })
}

/**
 * Fetch task deadlines within a date range for calendar overlay.
 * Uses the work service task endpoint with date filter.
 */
export function useTaskDeadlines(start: string, end: string) {
  return useQuery({
    queryKey: ['task-deadlines', { start, end }],
    queryFn: () =>
      calendarApi.GET<TaskDeadlinesResponse>(
        '/api/v1/calendar/task-deadlines',
        { start, end },
      ),
    enabled: !!start && !!end,
  })
}

// ---------------------------------------------------------------------------
// Event mutations
// ---------------------------------------------------------------------------

export function useCreateEvent() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: CreateEventRequest) =>
      calendarApi.POST<EventResponse>('/api/v1/calendar/events', body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['events'] })
    },
  })
}

export function useUpdateEvent() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: UpdateEventRequest & { id: string }) =>
      calendarApi.PUT<EventResponse>(`/api/v1/calendar/events/${id}`, body),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ['events'] })
      queryClient.invalidateQueries({ queryKey: ['events', variables.id] })
    },
  })
}

/**
 * Update a single occurrence or future occurrences of a recurring event.
 * Sends scope (this | this_and_future | all) and original_date.
 */
export function useUpdateRecurringEvent() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      id,
      ...body
    }: UpdateRecurringEventRequest & { id: string }) =>
      calendarApi.PUT<EventResponse>(
        `/api/v1/calendar/events/${id}/recurring`,
        body,
      ),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ['events'] })
      queryClient.invalidateQueries({ queryKey: ['events', variables.id] })
    },
  })
}

export function useDeleteEvent() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) =>
      calendarApi.DELETE(`/api/v1/calendar/events/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['events'] })
    },
  })
}

// ---------------------------------------------------------------------------
// RSVP mutation
// ---------------------------------------------------------------------------

export function useRSVPToEvent() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      eventId,
      status,
    }: {
      eventId: string
      status: RSVPStatus
    }) =>
      calendarApi.POST(`/api/v1/calendar/events/${eventId}/rsvp`, { status }),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ['events'] })
      queryClient.invalidateQueries({
        queryKey: ['events', variables.eventId, 'attendees'],
      })
    },
  })
}

// ---------------------------------------------------------------------------
// Reminder mutation
// ---------------------------------------------------------------------------

export function useSetEventReminders() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      eventId,
      minutes_before,
    }: {
      eventId: string
      minutes_before: number[]
    }) =>
      calendarApi.PUT(`/api/v1/calendar/events/${eventId}/reminders`, {
        minutes_before,
      }),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['events', variables.eventId],
      })
    },
  })
}
