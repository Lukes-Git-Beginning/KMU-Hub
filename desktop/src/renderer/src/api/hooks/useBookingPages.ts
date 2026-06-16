/**
 * TanStack Query hooks for Booking Pages.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listBookingPages,
  getBookingPage,
  createBookingPage,
  updateBookingPage,
  deleteBookingPage,
} from '../booking-client'
import type { CreateBookingPageRequest, UpdateBookingPageRequest } from '../booking-client'

const QUERY_KEY = ['booking-pages'] as const

// ---------------------------------------------------------------------------
// Queries
// ---------------------------------------------------------------------------

/** List all booking pages. Pass includeInactive=true to include inactive ones. */
export function useBookingPages(includeInactive?: boolean) {
  return useQuery({
    queryKey: [...QUERY_KEY, { includeInactive }],
    queryFn: () => listBookingPages(includeInactive),
  })
}

/** Get a single booking page by ID. Only runs when id is defined. */
export function useBookingPage(id: string | undefined) {
  return useQuery({
    queryKey: [...QUERY_KEY, id],
    queryFn: () => getBookingPage(id!),
    enabled: !!id,
  })
}

// ---------------------------------------------------------------------------
// Mutations
// ---------------------------------------------------------------------------

/** Create a new booking page. Invalidates the list on success. */
export function useCreateBookingPage() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (req: CreateBookingPageRequest) => createBookingPage(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEY })
    },
  })
}

/** Update an existing booking page. Invalidates the list on success. */
export function useUpdateBookingPage() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, req }: { id: string; req: UpdateBookingPageRequest }) =>
      updateBookingPage(id, req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEY })
    },
  })
}

/** Delete a booking page. Invalidates the list on success. */
export function useDeleteBookingPage() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteBookingPage(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEY })
    },
  })
}
