/**
 * TanStack Query hooks for Resource management.
 *
 * Includes resource list/detail, availability checks, booking/cancellation.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { calendarApi } from '../calendar-client'
import type {
  ResourceListResponse,
  ResourceResponse,
  ResourceBookingsResponse,
  CreateResourceRequest,
  UpdateResourceRequest,
  BookResourceRequest,
  ResourceType,
} from '../calendar-types'

// ---------------------------------------------------------------------------
// Query params
// ---------------------------------------------------------------------------

export interface ResourceListParams {
  resource_type?: ResourceType
  min_capacity?: number
  tag?: string
  is_active?: boolean
}

// ---------------------------------------------------------------------------
// Resource queries
// ---------------------------------------------------------------------------

/** List resources with optional filters (type, capacity, tag). */
export function useResources(params?: ResourceListParams) {
  return useQuery({
    queryKey: ['resources', params],
    queryFn: () =>
      calendarApi.GET<ResourceListResponse>('/api/v1/calendar/resources', {
        resource_type: params?.resource_type,
        min_capacity: params?.min_capacity,
        tag: params?.tag,
        is_active: params?.is_active,
      }),
  })
}

/** Fetch a single resource by ID. */
export function useResource(resourceId: string) {
  return useQuery({
    queryKey: ['resources', resourceId],
    queryFn: () =>
      calendarApi.GET<ResourceResponse>(
        `/api/v1/calendar/resources/${resourceId}`,
      ),
    enabled: !!resourceId,
  })
}

/** Fetch bookings for a resource on a specific date. */
export function useResourceAvailability(resourceId: string, date: string) {
  return useQuery({
    queryKey: ['resources', resourceId, 'availability', date],
    queryFn: () =>
      calendarApi.GET<ResourceBookingsResponse>(
        `/api/v1/calendar/resources/${resourceId}/availability`,
        { date },
      ),
    enabled: !!resourceId && !!date,
  })
}

// ---------------------------------------------------------------------------
// Resource mutations
// ---------------------------------------------------------------------------

export function useCreateResource() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: CreateResourceRequest) =>
      calendarApi.POST<ResourceResponse>('/api/v1/calendar/resources', body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['resources'] })
    },
  })
}

export function useUpdateResource() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: UpdateResourceRequest & { id: string }) =>
      calendarApi.PUT<ResourceResponse>(
        `/api/v1/calendar/resources/${id}`,
        body,
      ),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ['resources'] })
      queryClient.invalidateQueries({
        queryKey: ['resources', variables.id],
      })
    },
  })
}

export function useDeleteResource() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) =>
      calendarApi.DELETE(`/api/v1/calendar/resources/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['resources'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Booking mutations
// ---------------------------------------------------------------------------

export function useBookResource() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: BookResourceRequest) =>
      calendarApi.POST('/api/v1/calendar/resources/bookings', body),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ['resources'] })
      queryClient.invalidateQueries({
        queryKey: ['resources', variables.resource_id],
      })
      queryClient.invalidateQueries({ queryKey: ['events'] })
    },
  })
}

export function useCancelBooking() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (bookingId: string) =>
      calendarApi.DELETE(`/api/v1/calendar/resources/bookings/${bookingId}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['resources'] })
      queryClient.invalidateQueries({ queryKey: ['events'] })
    },
  })
}
