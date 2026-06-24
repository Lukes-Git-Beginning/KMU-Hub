/**
 * Shared helpers for the work module that bridge gaps between the OpenAPI
 * contract and the data the UI consumes.
 */
import type { components } from '@/api/types'

type ProjectMember = components['schemas']['ProjectMemberResponse']

/**
 * Display name for a project member. The backend Member shape only carries
 * first_name/last_name (no denormalized display_name/email), so we derive the
 * label and fall back to the user id when no name is present.
 */
export function memberDisplayName(
  m: Pick<ProjectMember, 'first_name' | 'last_name' | 'user_id'>
): string {
  const full = `${m.first_name ?? ''} ${m.last_name ?? ''}`.trim()
  return full || m.user_id || ''
}

/**
 * Task response with denormalized fields the demo MSW provides but the real
 * backend proto does not emit yet:
 *   - is_closed: derived from the task's project status (status.is_closed)
 *   - created_by_name: resolved from a user join over created_by
 * Both are typed optional so real-backend responses (which omit them) stay valid
 * and the UI degrades gracefully.
 */
export type TaskWithDerived = components['schemas']['TaskResponse'] & {
  is_closed?: boolean
  created_by_name?: string
}
