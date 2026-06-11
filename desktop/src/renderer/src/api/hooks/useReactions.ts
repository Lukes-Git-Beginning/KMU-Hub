/**
 * TanStack Query hooks for message reactions (emoji).
 *
 * Toggle uses optimistic update: invalidates the per-message reaction cache
 * so the next render re-fetches fresh data from the chat service.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listReactions,
  getReactionSummary,
  toggleReaction,
} from '../video-client'
import type { ToggleReactionRequest } from '../video-types'

// ---------------------------------------------------------------------------
// Queries
// ---------------------------------------------------------------------------

/** Get individual reactions for a single message. */
export function useReactions(messageId: string) {
  return useQuery({
    queryKey: ['reactions', messageId],
    queryFn: () => listReactions(messageId),
    enabled: !!messageId,
  })
}

/** Get reaction summaries for multiple messages (batch). */
export function useReactionSummary(messageIds: string[]) {
  return useQuery({
    queryKey: ['reactions', 'summary', messageIds],
    queryFn: () => getReactionSummary(messageIds),
    enabled: messageIds.length > 0,
  })
}

// ---------------------------------------------------------------------------
// Mutations
// ---------------------------------------------------------------------------

/** Toggle a reaction on a message (add or remove). */
export function useToggleReaction() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (req: ToggleReactionRequest) => toggleReaction(req),
    onSuccess: (_data, variables) => {
      // Invalidate the per-message reaction cache so the UI re-fetches
      void queryClient.invalidateQueries({ queryKey: ['reactions', variables.message_id] })
    },
  })
}
