import { describe, it, expect } from 'vitest'
import { backendMeetingToUI } from '../mappers'
import type { Meeting as ApiMeeting } from '@/api/video-types'

// The meeting list/get handlers serialize via encoding/json, so on the wire
// timestamps arrive as protobuf {seconds,nanos} and status as the numeric enum.
// The mapper must also accept the RFC3339 / string-union shape the FE type
// declares (and the proto enum names). These tests lock in both.

const BASE = {
  id: 'mtg-1',
  title: 'Sprint Planning',
  description: 'desc',
  agenda: 'Topic A',
  organizer_id: 'user-1',
  room_name: 'Room A',
  calendar_event_id: null,
  recurring_meeting_id: null,
  actual_start: null,
  actual_end: null,
  created_at: '2026-06-20T08:00:00Z',
  updated_at: '2026-06-20T08:00:00Z',
  attendees: [{ meeting_id: 'mtg-1', user_id: 'user-2', rsvp_status: 'accepted', user_name: 'Anna Müller' }],
}

describe('backendMeetingToUI', () => {
  it('parses the raw proto wire shape ({seconds,nanos} + numeric enum)', () => {
    const wire = {
      ...BASE,
      status: 2, // MEETING_STATUS_IN_PROGRESS -> live
      scheduled_start: { seconds: 1_718_877_600, nanos: 0 },
      scheduled_end: { seconds: 1_718_881_200, nanos: 0 }, // +3600s = 60 min
    } as unknown as ApiMeeting

    const ui = backendMeetingToUI(wire)
    expect(ui.status).toBe('live')
    expect(ui.duration).toBe(60)
    expect(ui.date).not.toBe('')
    expect(ui.startTime).not.toBe('')
    expect(ui.participants).toHaveLength(1)
    expect(ui.participants[0].initials).toBe('AM')
    expect(ui.agenda).toHaveLength(1)
  })

  it('parses the declared FE shape (RFC3339 strings + string union)', () => {
    const wire = {
      ...BASE,
      status: 'scheduled',
      scheduled_start: '2026-06-20T10:00:00Z',
      scheduled_end: '2026-06-20T11:30:00Z', // 90 min
    } as unknown as ApiMeeting

    const ui = backendMeetingToUI(wire)
    expect(ui.status).toBe('scheduled')
    expect(ui.duration).toBe(90)
  })

  it('maps proto enum names and falls back safely on garbage', () => {
    const named = backendMeetingToUI({
      ...BASE,
      status: 'MEETING_STATUS_CANCELLED',
      scheduled_start: '2026-06-20T10:00:00Z',
      scheduled_end: '2026-06-20T10:30:00Z',
    } as unknown as ApiMeeting)
    expect(named.status).toBe('cancelled')

    const broken = backendMeetingToUI({
      ...BASE,
      status: 99,
      scheduled_start: undefined,
      scheduled_end: undefined,
      attendees: null,
    } as unknown as ApiMeeting)
    expect(broken.status).toBe('scheduled')
    expect(broken.duration).toBe(0)
    expect(broken.date).toBe('')
    expect(broken.participants).toEqual([])
  })

  it('derives a stable colour from the id', () => {
    const a = backendMeetingToUI({ ...BASE, status: 1, scheduled_start: '2026-06-20T10:00:00Z', scheduled_end: '2026-06-20T11:00:00Z' } as unknown as ApiMeeting)
    const b = backendMeetingToUI({ ...BASE, status: 1, scheduled_start: '2026-06-20T10:00:00Z', scheduled_end: '2026-06-20T11:00:00Z' } as unknown as ApiMeeting)
    expect(a.color).toBe(b.color)
    expect(a.color).toMatch(/^#[0-9A-F]{6}$/i)
  })
})
