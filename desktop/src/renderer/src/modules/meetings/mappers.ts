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
 *
 * Wire-shape robustness: the meeting list/get handlers serialize via
 * encoding/json (not protojson), so on the wire `scheduled_start`/`scheduled_end`
 * arrive as protobuf `{seconds, nanos}` objects and `status` arrives as the
 * numeric enum value — NOT the RFC3339 strings / string unions the FE types
 * declare. The parsers below accept BOTH the raw proto shape and the declared
 * string shape so this mapper stays correct whether or not the backend handlers
 * are later migrated to response.Proto. (Backend serialization alignment is a
 * tracked follow-up.)
 */
import type { Meeting as ApiMeeting } from '@/api/video-types'
import type { Meeting as UIMeeting } from '@/stores/meetings'

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

interface ProtoTimestamp {
  seconds?: number | string
  nanos?: number
}

/** Parse a timestamp that may arrive as RFC3339 string or protobuf {seconds,nanos}. */
function parseWireDate(value: unknown): Date {
  if (value == null) return new Date(NaN)
  if (typeof value === 'string' || typeof value === 'number') return new Date(value)
  if (typeof value === 'object' && 'seconds' in value) {
    const ts = value as ProtoTimestamp
    const secs = typeof ts.seconds === 'string' ? Number(ts.seconds) : ts.seconds ?? 0
    return new Date(secs * 1000 + Math.floor((ts.nanos ?? 0) / 1_000_000))
  }
  return new Date(NaN)
}

function isValidDate(d: Date): boolean {
  return !Number.isNaN(d.getTime())
}

function toDateStr(d: Date): string {
  if (!isValidDate(d)) return ''
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

function toTimeStr(d: Date): string {
  if (!isValidDate(d)) return ''
  return `${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// MeetingStatus can arrive as the numeric proto enum (1..4), the proto enum name
// (MEETING_STATUS_*), or the lowercase string union the FE type declares.
const STATUS_NUM_MAP: Record<number, UIMeeting['status']> = {
  1: 'scheduled',
  2: 'live',
  3: 'past',
  4: 'cancelled',
}
const STATUS_STR_MAP: Record<string, UIMeeting['status']> = {
  scheduled: 'scheduled',
  in_progress: 'live',
  completed: 'past',
  cancelled: 'cancelled',
  MEETING_STATUS_SCHEDULED: 'scheduled',
  MEETING_STATUS_IN_PROGRESS: 'live',
  MEETING_STATUS_COMPLETED: 'past',
  MEETING_STATUS_CANCELLED: 'cancelled',
}

function mapStatus(value: unknown): UIMeeting['status'] {
  if (typeof value === 'number') return STATUS_NUM_MAP[value] ?? 'scheduled'
  if (typeof value === 'string') return STATUS_STR_MAP[value] ?? 'scheduled'
  return 'scheduled'
}

/** Project a backend meeting onto the UI meeting shape for display. */
export function backendMeetingToUI(m: ApiMeeting): UIMeeting {
  const start = parseWireDate(m.scheduled_start)
  const end = parseWireDate(m.scheduled_end)
  const durationMin =
    isValidDate(start) && isValidDate(end)
      ? Math.max(0, Math.round((end.getTime() - start.getTime()) / 60_000))
      : 0

  return {
    id: m.id,
    title: m.title,
    status: mapStatus(m.status),
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
