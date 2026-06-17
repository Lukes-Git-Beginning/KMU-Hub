/**
 * Birthdays query hook — upcoming team birthdays (MSW-backed, swap-ready).
 *
 * Backend gap: EmployeeProfile has no `birthday` column yet. Until then the MSW
 * handler (GET /api/v1/dashboard/birthdays) serves mock data; the widget reads it
 * via this query hook so swapping to a real endpoint is a handler-only change.
 */
import { useQuery } from '@tanstack/react-query'
import { API_BASE_URL } from '@/lib/constants'

export interface Birthday {
  employeeId: string
  name: string
  initials: string
  department: string
  displayDate: string
  daysUntil: number
}

export function useBirthdays() {
  return useQuery({
    queryKey: ['dashboard', 'birthdays'],
    queryFn: async (): Promise<Birthday[]> => {
      const res = await fetch(`${API_BASE_URL}/api/v1/dashboard/birthdays`)
      if (!res.ok) throw new Error('Failed to load birthdays')
      const json = (await res.json()) as { birthdays?: Birthday[] }
      return json.birthdays ?? []
    },
    staleTime: 60 * 60 * 1000,
  })
}
