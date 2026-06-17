/**
 * Team weekly worktime hook — per-member weekly minutes (MSW-backed, swap-ready).
 *
 * Replaces the dashboard widget's client-side seeding: the handler
 * (GET /api/v1/dashboard/team-worktime) returns deterministic, per-employee
 * weekly minutes so the swap to a real endpoint is handler-only.
 */
import { useQuery } from '@tanstack/react-query'
import { API_BASE_URL } from '@/lib/constants'

export interface TeamWorktimeMember {
  id: string
  name: string
  minutes: number
}

export function useTeamWorktime() {
  return useQuery({
    queryKey: ['dashboard', 'team-worktime'],
    queryFn: async (): Promise<TeamWorktimeMember[]> => {
      const res = await fetch(`${API_BASE_URL}/api/v1/dashboard/team-worktime`)
      if (!res.ok) throw new Error('Failed to load team worktime')
      const json = (await res.json()) as { members?: TeamWorktimeMember[] }
      return json.members ?? []
    },
    staleTime: 5 * 60 * 1000,
  })
}
