/**
 * Hooks for chat message bookmarks (saved messages).
 *
 * The bookmark endpoints are demo-only (not in the generated OpenAPI types),
 * so they use a raw fetch — MSW intercepts the requests in demo mode.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { API_BASE_URL } from '@/lib/constants'
import type { components } from '@/api/types'
import { messageKeys } from './useMessages'

type MessageInfo = components['schemas']['MessageInfo']

export const bookmarkKeys = {
  all: ['bookmarks'] as const,
}

/** List all bookmarked messages for the current user. */
export function useBookmarks() {
  return useQuery({
    queryKey: bookmarkKeys.all,
    queryFn: async () => {
      const res = await fetch(`${API_BASE_URL}/api/v1/messages/bookmarks`)
      if (!res.ok) throw new Error('Failed to load bookmarks')
      return (await res.json()) as { messages: MessageInfo[] }
    },
  })
}

/** Toggle the bookmark state of a message. */
export function useToggleBookmark() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (messageId: string) => {
      const res = await fetch(`${API_BASE_URL}/api/v1/messages/${messageId}/bookmark`, {
        method: 'POST',
      })
      if (!res.ok) throw new Error('Failed to toggle bookmark')
      return (await res.json()) as { bookmarked: boolean }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: bookmarkKeys.all })
      queryClient.invalidateQueries({ queryKey: messageKeys.all })
    },
  })
}
