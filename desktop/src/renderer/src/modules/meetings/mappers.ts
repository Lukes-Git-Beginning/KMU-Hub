/**
 * Mapping between the backend meeting model (api/video-types `Meeting`) and the
 * richer client-side UI model (stores/meetings `Meeting`).
 *
 * The backend model is intentionally thinner than the UI model: it has no
 * project/color/files/reminder, a single free-text agenda string (vs. an array
 * of checkable items), and no per-meeting recurrence cadence. UI-only fields are
 * therefore derived deterministically (color) or defaulted (project, files,
 * reminder). This is a read-only projection — edits to these fields are not
 * persisted back to the backend in the current pragmatic scope (R3-E3).
 */
import type { Meeting as ApiMeeting } from '@/api/video-types'
import type { Meeting as UIMeeting } from '@/stores/meetings'

const STATUS_MAP: Record<ApiMeeting['status'], UIMeeting['status']> = {
  scheduled: 'scheduled',
  in_progress: 'live',
  completed: 'past',
  cancelled: 'cancelled',
}

// Stable accent palette (mirrors the colours used by the mock seed data).
const PALETTE = ['#3B82F6', '#8B5CF6', '#10B981', '#F59E0B', '#EF4444', '#6B7280']

/** Deterministic colour from a meeting id — same id always yields same colour. */
function colorFor(id: string): string {
  let hash = 0
  for (let i = 0; i < id.length; i++) {
    hash = (hash * 31 + id.charCodeAt(i)) >>> 0
  }
  return PALETTE[hash % PALETTE.length]
}

function initialsFor(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) return '?'
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase()
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
}

function pad(n: number): string {
  return String(n).padStart(2, '0')
}

function toDateStr(d: Date): string {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

function toTimeStr(d: Date): string {
  return `${pad(d.getHours())}:${pad(d.getMinutes())}`
}

/** Project a backend meeting onto the UI meeting shape for display. */
export function backendMeetingToUI(m: ApiMeeting): UIMeeting {
  const start = new Date(m.scheduled_start)
  const end = new Date(m.scheduled_end)
  const durationMin = Math.max(
    0,
    Math.round((end.getTime() - start.getTime()) / 60_000),
  )

  return {
    id: m.id,
    title: m.title,
    status: STATUS_MAP[m.status] ?? 'scheduled',
    project: '',
    color: colorFor(m.id),
    date: toDateStr(start),
    startTime: toTimeStr(start),
    duration: durationMin,
    room: m.room_name ?? '',
    isVideoCall: true,
    // The backend stores a recurrence link but not the cadence, so we cannot
    // faithfully reconstruct daily/weekly/monthly here.
    recurrence: 'none',
    reminder: 'none',
    description: m.description ?? '',
    participants: (m.attendees ?? []).map((a) => ({
      id: a.user_id,
      name: a.user_name ?? a.user_id,
      initials: initialsFor(a.user_name ?? '?'),
    })),
    organizerId: m.organizer_id,
    agenda: m.agenda
      ? [{ id: `${m.id}-agenda`, text: m.agenda, done: false }]
      : [],
    notes: '',
    files: [],
    whiteboardLink: '',
    projectLink: '',
    calendarEventId: m.calendar_event_id ?? undefined,
    invitationsSent: (m.attendees?.length ?? 0) > 0,
  }
}
