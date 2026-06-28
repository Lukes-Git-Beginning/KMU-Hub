import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { IDS } from '../data/shared-ids'
import { daysAgo, daysFromNow, hoursAgo } from '../data/date-helpers'
import type { BreakoutRoom } from '@/api/video-types'

const API = API_BASE_URL

// ---------------------------------------------------------------------------
// Meetings (5)
// ---------------------------------------------------------------------------

const meetings = [
  {
    id: 'mtg-001',
    title: 'Sprint Review',
    description: 'Review der abgeschlossenen Stories aus Sprint 42.',
    organizer_id: IDS.users.elena,
    organizer_name: 'Sarah Beck',
    start_time: daysFromNow(1) + 'T14:00:00Z',
    end_time: daysFromNow(1) + 'T15:00:00Z',
    status: 'scheduled',
    participants: [
      { user_id: IDS.users.stefan, name: 'Stefan Vogel', status: 'accepted' },
      { user_id: IDS.users.markus, name: 'Markus Weber', status: 'accepted' },
      { user_id: IDS.users.elena, name: 'Sarah Beck', status: 'accepted' },
      { user_id: IDS.users.laura, name: 'Laura Neumann', status: 'accepted' },
      { user_id: IDS.users.felix, name: 'Felix Krause', status: 'tentative' },
    ],
    recording_enabled: true,
    created_at: daysAgo(3),
  },
  {
    id: 'mtg-002',
    title: 'Vertrieb Weekly',
    description: 'Woechentliche Pipeline-Besprechung.',
    organizer_id: IDS.users.thomas,
    organizer_name: 'Thomas Meier',
    start_time: daysFromNow(2) + 'T10:00:00Z',
    end_time: daysFromNow(2) + 'T10:45:00Z',
    status: 'scheduled',
    participants: [
      { user_id: IDS.users.thomas, name: 'Thomas Meier', status: 'accepted' },
      { user_id: IDS.users.laura, name: 'Sabine Fischer', status: 'accepted' },
      { user_id: IDS.users.david, name: 'Kevin Baumann', status: 'accepted' },
      { user_id: IDS.users.stefan, name: 'Stefan Vogel', status: 'tentative' },
    ],
    recording_enabled: false,
    created_at: daysAgo(5),
  },
  {
    id: 'mtg-003',
    title: 'Design Review: Dashboard v2',
    description: 'Feedback zum neuen Dashboard-Design.',
    organizer_id: IDS.users.nina,
    organizer_name: 'Nina Richter',
    start_time: daysFromNow(3) + 'T11:00:00Z',
    end_time: daysFromNow(3) + 'T12:00:00Z',
    status: 'scheduled',
    participants: [
      { user_id: IDS.users.nina, name: 'Nina Richter', status: 'accepted' },
      { user_id: IDS.users.sophie, name: 'Sophie Lang', status: 'accepted' },
      { user_id: IDS.users.lena, name: 'Lena Braun', status: 'accepted' },
      { user_id: IDS.users.markus, name: 'Markus Weber', status: 'accepted' },
    ],
    recording_enabled: false,
    created_at: daysAgo(1),
  },
  {
    id: 'mtg-004',
    title: 'Demo: Gruber Maschinenbau',
    description: 'Produkt-Demo für Neukunden — CRM und Projektmanagement.',
    organizer_id: IDS.users.thomas,
    organizer_name: 'Thomas Meier',
    start_time: daysFromNow(5) + 'T14:00:00Z',
    end_time: daysFromNow(5) + 'T15:30:00Z',
    status: 'scheduled',
    participants: [
      { user_id: IDS.users.thomas, name: 'Thomas Meier', status: 'accepted' },
      { user_id: IDS.users.stefan, name: 'Stefan Vogel', status: 'accepted' },
    ],
    recording_enabled: true,
    created_at: hoursAgo(1),
  },
  {
    id: 'mtg-005',
    title: 'All-Hands März',
    description: 'Monatliches Company-Meeting — Updates aus allen Abteilungen.',
    organizer_id: IDS.users.stefan,
    organizer_name: 'Stefan Vogel',
    start_time: daysAgo(5) + 'T09:00:00Z',
    end_time: daysAgo(5) + 'T10:00:00Z',
    status: 'completed',
    participants: EMPLOYEES_SHORT(),
    recording_enabled: true,
    created_at: daysAgo(14),
  },
]

/** Short participant list for all-hands */
function EMPLOYEES_SHORT() {
  return [
    { user_id: IDS.users.stefan, name: 'Stefan Vogel', status: 'accepted' },
    { user_id: IDS.users.markus, name: 'Markus Weber', status: 'accepted' },
    { user_id: IDS.users.thomas, name: 'Thomas Meier', status: 'accepted' },
    { user_id: IDS.users.julia, name: 'Julia Hofmann', status: 'accepted' },
    { user_id: IDS.users.elena, name: 'Sarah Beck', status: 'accepted' },
    { user_id: IDS.users.nina, name: 'Nina Richter', status: 'accepted' },
    { user_id: IDS.users.felix, name: 'Felix Krause', status: 'accepted' },
    { user_id: IDS.users.lena, name: 'Lena Braun', status: 'declined' },
  ]
}

// ---------------------------------------------------------------------------
// Breakout room demo state (stateful module-level store)
// ---------------------------------------------------------------------------

// lean: single shared store for all breakout calls; resets between live-reload cycles.
let _breakoutRooms: BreakoutRoom[] = []

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

export const videoHandlers = [
  // Active calls — empty (no live calls in demo)
  http.get(`${API}/api/v1/video/calls`, () => {
    return HttpResponse.json({ calls: [], total: 0 })
  }),

  // Recordings — empty
  http.get(`${API}/api/v1/video/recordings`, () => {
    return HttpResponse.json({ recordings: [], total: 0 })
  }),

  // Meeting list
  http.get(`${API}/api/v1/meetings`, ({ request }) => {
    const url = new URL(request.url)
    const status = url.searchParams.get('status')

    let filtered = [...meetings]
    if (status) {
      filtered = filtered.filter((m) => m.status === status)
    }

    // Match the real wire shape: a bare array (gateway uses response.ProtoList),
    // not a {meetings,total} envelope — the client maps the response directly.
    return HttpResponse.json(filtered)
  }),

  // Meeting detail
  http.get(`${API}/api/v1/meetings/:id`, ({ params }) => {
    const meeting = meetings.find((m) => m.id === params.id)
    if (!meeting) {
      return HttpResponse.json({ error: 'Meeting not found' }, { status: 404 })
    }
    // Bare Meeting object — getMeeting() expects Meeting, not { meeting }.
    return HttpResponse.json(meeting)
  }),

  // -------------------------------------------------------------------------
  // Breakout rooms (Wave 6A demo handlers)
  // -------------------------------------------------------------------------

  // Create breakout rooms
  http.post(`${API}/api/v1/meetings/:meetingId/breakout`, async ({ params, request }) => {
    const body = await request.json() as { count: number }
    const count = Math.max(1, Math.min(20, body.count ?? 2))
    _breakoutRooms = Array.from({ length: count }, (_, i) => ({
      id: `br-${String(params.meetingId)}-${i + 1}`,
      meeting_id: String(params.meetingId),
      label: `Raum ${i + 1}`,
      status: 'open' as const,
      participant_ids: [],
    }))
    return HttpResponse.json({ rooms: _breakoutRooms })
  }),

  // List breakout rooms
  http.get(`${API}/api/v1/meetings/:meetingId/breakout`, () => {
    return HttpResponse.json({ rooms: _breakoutRooms })
  }),

  // Assign participant to a room (or back to main room)
  http.post(`${API}/api/v1/meetings/:meetingId/breakout/assign`, async ({ request }) => {
    const body = await request.json() as { target_user_id: string; breakout_room_id: string | null }
    _breakoutRooms = _breakoutRooms.map((room) => {
      // Remove from current room first
      const withoutTarget = room.participant_ids.filter((id) => id !== body.target_user_id)
      // Add to target room
      const participantIds =
        room.id === body.breakout_room_id
          ? [...withoutTarget, body.target_user_id]
          : withoutTarget
      return { ...room, participant_ids: participantIds }
    })
    return HttpResponse.json({})
  }),

  // Join breakout room (returns a demo token)
  http.post(`${API}/api/v1/meetings/:meetingId/breakout/join`, ({ params }) => {
    const room = _breakoutRooms[0] ?? {
      id: `br-${String(params.meetingId)}-1`,
      meeting_id: String(params.meetingId),
      label: 'Raum 1',
      status: 'open' as const,
      participant_ids: [],
    }
    return HttpResponse.json({
      token: `demo-breakout-token-${Date.now()}`,
      ws_url: 'wss://demo.livekit.example/breakout',
      room,
    })
  }),

  // Get caller's current breakout assignment (demo: always main room)
  http.get(`${API}/api/v1/meetings/:meetingId/breakout/assignment`, () => {
    return HttpResponse.json({ room: null })
  }),

  // Return participant to main room
  http.post(`${API}/api/v1/meetings/:meetingId/breakout/return`, async ({ request }) => {
    const body = await request.json() as { target_user_id?: string } | null
    const targetId = body?.target_user_id
    if (targetId) {
      _breakoutRooms = _breakoutRooms.map((room) => ({
        ...room,
        participant_ids: room.participant_ids.filter((id) => id !== targetId),
      }))
    }
    return HttpResponse.json({})
  }),

  // Close all breakout rooms
  http.post(`${API}/api/v1/meetings/:meetingId/breakout/close`, () => {
    _breakoutRooms = []
    return HttpResponse.json({})
  }),
]
