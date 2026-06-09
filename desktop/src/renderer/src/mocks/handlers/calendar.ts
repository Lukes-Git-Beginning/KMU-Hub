import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { mockCalendars, mockEvents, mockResources, buildMockHolidays } from '../data/events'

const API = API_BASE_URL

/** Derive an RRULE from the legacy mock `recurring` field so recurring events
 *  are detectable by the UI (enables the recurring-edit scope dialog). */
function recurringToRrule(recurring: unknown): string | null {
  switch (recurring) {
    case 'daily':
      return 'FREQ=DAILY'
    case 'weekly':
      return 'FREQ=WEEKLY'
    case 'biweekly':
      return 'FREQ=WEEKLY;INTERVAL=2'
    case 'monthly':
      return 'FREQ=MONTHLY'
    default:
      return null
  }
}

export const calendarHandlers = [
  // List calendars
  http.get(`${API}/api/v1/calendars`, () => {
    return HttpResponse.json(mockCalendars)
  }),

  // List events (supports ?start=...&end=... date range filter)
  http.get(`${API}/api/v1/calendar/events`, ({ request }) => {
    const url = new URL(request.url)
    const start = url.searchParams.get('start')
    const end = url.searchParams.get('end')
    // calendar_ids is sent as repeated params — honour all of them (not just
    // the first), otherwise only the first calendar's events ever show.
    const calendarIds = url.searchParams.getAll('calendar_ids')
    const singleId = url.searchParams.get('calendar_id')
    const ids = calendarIds.length ? calendarIds : singleId ? [singleId] : []

    let filtered = [...mockEvents.events]

    if (start) {
      filtered = filtered.filter((e) => e.start_time >= start)
    }
    if (end) {
      filtered = filtered.filter((e) => e.start_time <= end)
    }
    if (ids.length) {
      filtered = filtered.filter((e) => ids.includes(e.calendar_id))
    }

    const withRrule = filtered.map((e) => ({
      ...e,
      rrule: (e as { rrule?: string }).rrule ?? recurringToRrule((e as { recurring?: unknown }).recurring),
    }))
    return HttpResponse.json({ events: withRrule, total: withRrule.length })
  }),

  // Event detail
  http.get(`${API}/api/v1/calendar/events/:id`, ({ params }) => {
    const event = mockEvents.events.find((e) => e.id === params.id)
    if (!event) {
      return HttpResponse.json({ error: 'Event not found' }, { status: 404 })
    }
    return HttpResponse.json({ event })
  }),

  // Create event
  http.post(`${API}/api/v1/calendar/events`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const newEvent = {
      id: `evt-${Date.now()}`,
      ...body,
      created_at: new Date().toISOString(),
    }
    return HttpResponse.json({ event: newEvent }, { status: 201 })
  }),

  // Update event
  http.put(`${API}/api/v1/calendar/events/:id`, async ({ params, request }) => {
    const existing = mockEvents.events.find((e) => e.id === params.id)
    if (!existing) {
      return HttpResponse.json({ error: 'Event not found' }, { status: 404 })
    }
    const body = (await request.json()) as Record<string, unknown>
    const updated = { ...existing, ...body }
    return HttpResponse.json({ event: updated })
  }),

  // Update recurring event (scope: this | this_and_future | all)
  http.put(`${API}/api/v1/calendar/events/:id/recurring`, async ({ params, request }) => {
    const existing = mockEvents.events.find((e) => e.id === params.id)
    if (!existing) {
      return HttpResponse.json({ error: 'Event not found' }, { status: 404 })
    }
    const body = (await request.json()) as Record<string, unknown>
    return HttpResponse.json({ event: { ...existing, ...body } })
  }),

  // Delete event
  http.delete(`${API}/api/v1/calendar/events/:id`, ({ params }) => {
    const exists = mockEvents.events.some((e) => e.id === params.id)
    if (!exists) {
      return HttpResponse.json({ error: 'Event not found' }, { status: 404 })
    }
    return new HttpResponse(null, { status: 204 })
  }),

  // List resources (rooms)
  http.get(`${API}/api/v1/resources`, () => {
    return HttpResponse.json(mockResources)
  }),

  // Public holidays (?year=&country_code=&subdivision_code=)
  http.get(`${API}/api/v1/calendar/holidays`, ({ request }) => {
    const url = new URL(request.url)
    const year = Number(url.searchParams.get('year')) || new Date().getFullYear()
    const country = url.searchParams.get('country_code') || 'DE'
    const subdivision = url.searchParams.get('subdivision_code') || ''
    return HttpResponse.json({ holidays: buildMockHolidays(year, country, subdivision) })
  }),
]
