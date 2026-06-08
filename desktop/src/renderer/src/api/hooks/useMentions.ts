/**
 * Hook for the current user's @mentions across all channels.
 *
 * Wraps GET /api/v1/messages/mentions (chat GetUserMentions RPC).
 * Real backend exists; a demo MSW handler returns mock mentions so it
 * works without a backend.
 */
import { useQuery } from '@tanstack/react-query'
import { apiClient } from '@/api/client'
import type { components } from '@/api/types'

export type UserMentionsResponse = components['schemas']['UserMentionsResponse']
export type MentionDetail = NonNullable<UserMentionsResponse['mentions']>[number]

export function useUserMentions(page = 1, pageSize = 20) {
  return useQuery({
    queryKey: ['chat', 'mentions', page, pageSize],
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/messages/mentions', {
        params: { query: { page, page_size: pageSize } },
      })
      if (error) throw error
      return data
    },
  })
}
