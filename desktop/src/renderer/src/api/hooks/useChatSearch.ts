/**
 * Chat full-text search hook (messages + files merged by relevance).
 *
 * Wraps GET /api/v1/chat/search. Real backend exists (chat SearchChat RPC);
 * a demo MSW handler searches the mock messages so it works without a backend.
 */
import { useQuery } from '@tanstack/react-query'
import { apiClient } from '@/api/client'
import type { components } from '@/api/types'

export type ChatSearchResult = components['schemas']['ChatSearchResult']

export function useChatSearch(q: string, channelId?: string) {
  const query = q.trim()
  return useQuery({
    queryKey: ['chat', 'search', query, channelId ?? null],
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/chat/search', {
        params: { query: { q: query, channel_id: channelId } },
      })
      if (error) throw error
      return data
    },
    // Backend requires min 2 chars; avoid firing for shorter queries.
    enabled: query.length >= 2,
  })
}
