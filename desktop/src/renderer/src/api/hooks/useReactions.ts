/**
 * TanStack Query hooks for message reactions (emoji).
 *
 * Toggle uses optimistic update: sets reaction data directly in the
 * query cache from the server response to avoid a refetch roundtrip.
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

/** Get reaction summaries for a single message. */
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
    onSuccess: (data, variables) => {
      // Optimistic update: set reaction data directly from server response
      queryClient.setQueryData(
        ['reactions', variables.message_id],
        data.reactions,
      )
    },
  })
}
