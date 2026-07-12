/**
 * TanStack Query hooks for session management.
 *
 * Provides queries for listing active sessions (own + admin all-users)
 * and mutations for session termination.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listMySessions,
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

// lean: Admin-Cross-User-Sessions-Ansicht (useAllSessions) entfernt — BE `/sessions/all?user_id=X`
// listet nur die Sessions EINES Users, kein globales Aggregat, und es gibt keinen User-Picker.
// Wieder einbauen, sobald ein Admin-User-Picker existiert oder ein Cross-User-Aggregat-Endpoint
// bereitsteht. Client-Fn `listAllSessions(userId)` bleibt in security-client.ts vorbereitet.

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
