/**
 * Resolves a user id to a display name for wiki authors and version editors.
 *
 * Sources the team directory (useEmployees) plus the current user. Falls back
 * to an empty string for unknown ids so callers can decide how to render.
 */
import { useMemo } from 'react'
import { useEmployees } from '@/api/hooks/hr-hooks'
import { CURRENT_USER } from '@/mocks/data/shared-ids'

export function useWikiUsers() {
  const { data } = useEmployees()
  return useMemo(() => {
    const map = new Map<string, string>()
    map.set(CURRENT_USER.id, CURRENT_USER.name)
    for (const e of data?.employees ?? []) {
      if (e.userId) map.set(e.userId, e.userName ?? '')
    }
    return (id: string | null | undefined): string => (id ? map.get(id) ?? '' : '')
  }, [data])
}
