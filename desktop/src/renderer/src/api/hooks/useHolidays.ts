/**
 * TanStack Query hooks for Public Holiday management.
 *
 * Includes holiday list queries and admin seed mutation
 * for fetching holidays from Nager.Date API.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { calendarApi } from '../calendar-client'
import type { HolidayListResponse } from '../calendar-types'

// ---------------------------------------------------------------------------
// Holiday queries
// ---------------------------------------------------------------------------

/**
 * Fetch public holidays for a given year and country.
 * Optionally filter by subdivision (e.g., 'BY' for Bavaria).
 */
export function useHolidays(
  year: number,
  countryCode: string,
  subdivisionCode?: string,
) {
  return useQuery({
    queryKey: ['holidays', { year, country: countryCode, subdivision: subdivisionCode }],
    queryFn: () =>
      calendarApi.GET<HolidayListResponse>('/api/v1/calendar/holidays', {
        year,
        country_code: countryCode,
        subdivision_code: subdivisionCode,
      }),
    enabled: !!year && !!countryCode,
  })
}

// ---------------------------------------------------------------------------
// Holiday mutations
// ---------------------------------------------------------------------------

/**
 * Admin action: seed holidays for a year/country from the Nager.Date API.
 * Triggers a backend fetch-and-store operation.
 */
export function useSeedHolidays() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      year,
      country_code,
    }: {
      year: number
      country_code: string
    }) =>
      calendarApi.POST('/api/v1/calendar/holidays/seed', {
        year,
        country_code,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['holidays'] })
    },
  })
}
