/**
 * Hooks for the unified unread inbox (KO-7) — every channel/DM with unread
 * messages, for focused cross-workspace triage. Demo-only endpoints, so they
 * use a raw fetch (MSW intercepts in demo mode).
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { API_BASE_URL } from '@/lib/constants'
import { channelKeys } from './useChannels'

export interface UnreadInboxMessage {
  id?: string
  content?: string
  created_at?: string
  sender_first_name?: string
  sender_last_name?: string
}

export interface UnreadInboxEntry {
  channel_id: string
  channel_name: string
  is_dm: boolean
  unread_count: number
  messages: UnreadInboxMessage[]
}

export const unreadInboxKeys = {
  all: ['unread-inbox'] as const,
}

/** Fetch the unified unread inbox. */
export function useUnreadInbox() {
  return useQuery({
    queryKey: unreadInboxKeys.all,
    queryFn: async () => {
      const res = await fetch(`${API_BASE_URL}/api/v1/messages/unread-inbox`)
      if (!res.ok) throw new Error('Failed to load unread inbox')
      return (await res.json()) as { entries: UnreadInboxEntry[] }
    },
  })
}

/** Mark every channel/DM as read. */
export function useMarkAllRead() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async () => {
      const res = await fetch(`${API_BASE_URL}/api/v1/channels/read-all`, { method: 'POST' })
      if (!res.ok) throw new Error('Failed to mark all read')
      return (await res.json()) as { success: boolean }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: unreadInboxKeys.all })
      queryClient.invalidateQueries({ queryKey: channelKeys.unread() })
    },
  })
}
