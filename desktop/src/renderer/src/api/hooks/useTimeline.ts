/**
 * TanStack Query hook for Contact Timeline.
 *
 * Fetches timeline events (activities + deal links) for a specific contact
 * from GET /api/v1/crm/contacts/{id}/timeline.
 */
import { useQuery } from '@tanstack/react-query'
import { authenticatedRequest } from '@/api/utils/authenticatedFetch'

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
  return authenticatedRequest<TimelineResult>({
    method: 'GET',
    path: `/api/v1/crm/contacts/${contactId}/timeline`,
    params: { offset, limit },
  })
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
