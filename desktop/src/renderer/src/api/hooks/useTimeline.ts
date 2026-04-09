/**
 * TanStack Query hook for Contact Timeline.
 *
 * Fetches timeline events (activities + deal links) for a specific contact
 * from GET /api/v1/crm/contacts/{id}/timeline.
 */
import { useQuery } from '@tanstack/react-query'
import { API_BASE_URL } from '@/lib/constants'
import { useAuthStore } from '@/stores/auth'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface TimelineEvent {
  id: string
  event_type: string // "activity" | "deal_linked"
  occurred_at: string // ISO 8601
  title: string
  description?: string
  created_by_name?: string
  metadata?: Record<string, unknown>
}

export interface TimelineResult {
  events: TimelineEvent[]
  total: number
}

// ---------------------------------------------------------------------------
// Fetch helper
// ---------------------------------------------------------------------------

async function fetchTimeline(
  contactId: string,
  offset: number,
  limit: number,
): Promise<TimelineResult> {
  const token = useAuthStore.getState().accessToken
  const url = `${API_BASE_URL}/api/v1/crm/contacts/${contactId}/timeline?offset=${offset}&limit=${limit}`

  const res = await fetch(url, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  })

  if (!res.ok) {
    throw new Error(`Timeline fetch failed: ${res.status}`)
  }

  return res.json()
}

// ---------------------------------------------------------------------------
// Query keys
// ---------------------------------------------------------------------------

export const timelineKeys = {
  contact: (id: string, offset?: number) =>
    ['crm', 'timeline', 'contact', id, offset] as const,
}

// ---------------------------------------------------------------------------
// Hook
// ---------------------------------------------------------------------------

export function useContactTimeline(
  contactId: string | undefined,
  page = 1,
  pageSize = 20,
) {
  const offset = (page - 1) * pageSize

  return useQuery({
    queryKey: timelineKeys.contact(contactId ?? '', offset),
    queryFn: () => fetchTimeline(contactId!, offset, pageSize),
    enabled: !!contactId,
    staleTime: 30_000,
  })
}
