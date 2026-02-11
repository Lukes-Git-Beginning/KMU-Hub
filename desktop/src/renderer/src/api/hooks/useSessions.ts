/**
 * TanStack Query hooks for session management.
 *
 * Provides queries for listing active sessions (own + admin all-users)
 * and mutations for session termination.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listMySessions,
  listAllSessions,
  terminateSession,
  terminateAllSessions,
} from '../security-client'

// ---------------------------------------------------------------------------
// Queries
// ---------------------------------------------------------------------------

/** List the current user's active sessions. */
export function useMySessions() {
  return useQuery({
    queryKey: ['sessions', 'mine'],
    queryFn: listMySessions,
  })
}

/** Admin: list all active sessions across users. */
export function useAllSessions(offset = 0, limit = 50) {
  return useQuery({
    queryKey: ['sessions', 'all', offset, limit],
    queryFn: () => listAllSessions({ offset, limit }),
  })
}

// ---------------------------------------------------------------------------
// Mutations
// ---------------------------------------------------------------------------

/** Terminate a specific session by ID. */
export function useTerminateSession() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (sessionId: string) => terminateSession(sessionId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['sessions'] })
    },
  })
}

/** Terminate all other sessions (keep current). */
export function useTerminateAllSessions() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => terminateAllSessions(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['sessions'] })
    },
  })
}
