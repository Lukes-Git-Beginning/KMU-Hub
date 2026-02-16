/**
 * MeetingDetailPanel -- slide-over panel showing full meeting details.
 *
 * Adapted from Darien's design (design/brainstorm) to use real backend API.
 * Shows attendees with RSVP status, agenda, recordings, and action items.
 */
import {
  Calendar,
  Clock,
  Video,
  Users,
  FileText,
  Pencil,
  Trash2,
  PlayCircle,
  CheckCircle2,
  CircleDot,
  XCircle,
  HelpCircle,
  ListTodo,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { DetailPanel } from '@/components/shared'
import { useActionItems } from '@/api/hooks/useMeetings'
import { useRecordings } from '@/api/hooks/useVideo'
import type { Meeting, MeetingStatus, RsvpStatus } from '@/api/video-types'

interface MeetingDetailPanelProps {
  meeting: Meeting | null
  open: boolean
  onClose: () => void
  onEdit: (meeting: Meeting) => void
  onDelete: (meetingId: string) => void
}

const statusConfig: Record<
  MeetingStatus,
  { label: string; variant: 'destructive' | 'secondary' | 'outline'; dot: string }
> = {
  in_progress: { label: 'Live', variant: 'destructive', dot: 'bg-red-500' },
  scheduled: { label: 'Geplant', variant: 'secondary', dot: 'bg-blue-500' },
  completed: { label: 'Beendet', variant: 'outline', dot: 'bg-green-500' },
  cancelled: { label: 'Abgesagt', variant: 'outline', dot: 'bg-gray-400' },
}

const rsvpIcons: Record<RsvpStatus, typeof CheckCircle2> = {
  accepted: CheckCircle2,
  declined: XCircle,
  tentative: HelpCircle,
  pending: CircleDot,
}

const rsvpColors: Record<RsvpStatus, string> = {
  accepted: 'text-green-500',
  declined: 'text-red-500',
  tentative: 'text-amber-500',
  pending: 'text-muted-foreground',
}

function formatDateTime(isoDate: string): string {
  return new Date(isoDate).toLocaleDateString('de-CH', {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  })
}

function formatTime(isoDate: string): string {
  return new Date(isoDate).toLocaleTimeString('de-CH', {
    hour: '2-digit',
    minute: '2-digit',
  })
}

function durationMinutes(start: string, end: string): number {
  return Math.round((new Date(end).getTime() - new Date(start).getTime()) / 60000)
}

function formatDuration(minutes: number): string {
  if (minutes >= 60) {
    const h = Math.floor(minutes / 60)
    const m = minutes % 60
    return m > 0 ? `${h} Std ${m} Min` : `${h} Std`
  }
  return `${minutes} Min`
}

export function MeetingDetailPanel({
  meeting,
  open,
  onClose,
  onEdit,
  onDelete,
}: MeetingDetailPanelProps) {
  // Hooks must be called unconditionally
  const meetingId = meeting?.id ?? ''
  const { data: actionItems = [] } = useActionItems(meetingId)
  const { data: recordings = [] } = useRecordings({ meetingId: meetingId || undefined })

  if (!meeting) return null

  const status = statusConfig[meeting.status]
  const duration = durationMinutes(meeting.scheduled_start, meeting.scheduled_end)
  const attendees = meeting.attendees ?? []

  return (
    <DetailPanel
      open={open}
      onClose={onClose}
      title={meeting.title}
      badge={
        <Badge variant={status.variant} className="ml-2 shrink-0">
          <span className={`mr-1.5 inline-block h-1.5 w-1.5 rounded-full ${status.dot}`} />
          {status.label}
        </Badge>
      }
      footer={
        <div className="flex gap-2">
          {meeting.status === 'in_progress' && (
            <Button className="flex-1" onClick={onClose}>
              <Video className="mr-1.5 h-4 w-4" />
              Zur Lobby
            </Button>
          )}
          {meeting.status === 'scheduled' && (
            <Button variant="outline" className="flex-1" onClick={() => onEdit(meeting)}>
              <Pencil className="mr-1.5 h-4 w-4" />
              Bearbeiten
            </Button>
          )}
          <Button
            variant="outline"
            size="icon"
            className="text-red-500 hover:text-red-600 hover:bg-red-50"
            onClick={() => onDelete(meeting.id)}
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      }
    >
      {/* Info Grid */}
      <div className="space-y-3">
        <div className="flex items-center gap-2 text-sm">
          <Calendar className="h-4 w-4 text-muted-foreground" />
          <span className="text-foreground">
            {formatDateTime(meeting.scheduled_start)}
          </span>
        </div>
        <div className="flex items-center gap-2 text-sm">
          <Clock className="h-4 w-4 text-muted-foreground" />
          <span className="text-foreground">
            {formatTime(meeting.scheduled_start)} - {formatTime(meeting.scheduled_end)}{' '}
            ({formatDuration(duration)})
          </span>
        </div>
        {meeting.actual_start && meeting.actual_end && (
          <div className="flex items-center gap-2 text-sm">
            <Clock className="h-4 w-4 text-muted-foreground" />
            <span className="text-foreground">
              Tatsaechlich: {formatTime(meeting.actual_start)} -{' '}
              {formatTime(meeting.actual_end)} (
              {formatDuration(durationMinutes(meeting.actual_start, meeting.actual_end))})
            </span>
          </div>
        )}
      </div>

      <Separator className="my-4" />

      {/* Attendees */}
      <div className="mb-4">
        <h4 className="mb-2 text-xs font-medium uppercase text-muted-foreground">
          <Users className="mr-1 inline h-3.5 w-3.5" />
          Teilnehmer ({attendees.length})
        </h4>
        <div className="space-y-1.5">
          {attendees.map((a) => {
            const RsvpIcon = rsvpIcons[a.rsvp_status]
            return (
              <div key={a.user_id} className="flex items-center gap-2">
                <span className="flex h-7 w-7 items-center justify-center rounded-full bg-primary/10 text-[10px] font-medium text-primary">
                  {(a.user_name ?? a.user_id).slice(0, 2).toUpperCase()}
                </span>
                <span className="text-sm text-foreground flex-1">
                  {a.user_name ?? a.user_id.slice(0, 8)}
                </span>
                <RsvpIcon className={`h-4 w-4 ${rsvpColors[a.rsvp_status]}`} />
              </div>
            )
          })}
          {attendees.length === 0 && (
            <p className="text-sm text-muted-foreground">Keine Teilnehmer</p>
          )}
        </div>
      </div>

      {/* Agenda */}
      {meeting.agenda && (
        <>
          <Separator className="my-4" />
          <div className="mb-4">
            <h4 className="mb-1.5 text-xs font-medium uppercase text-muted-foreground">Agenda</h4>
            <p className="text-sm text-foreground whitespace-pre-wrap">{meeting.agenda}</p>
          </div>
        </>
      )}

      {/* Description */}
      {meeting.description && (
        <>
          <Separator className="my-4" />
          <div className="mb-4">
            <h4 className="mb-1.5 text-xs font-medium uppercase text-muted-foreground">
              Beschreibung
            </h4>
            <p className="text-sm text-foreground whitespace-pre-wrap">{meeting.description}</p>
          </div>
        </>
      )}

      {/* Action Items Preview */}
      {actionItems.length > 0 && (
        <>
          <Separator className="my-4" />
          <div className="mb-4">
            <h4 className="mb-2 text-xs font-medium uppercase text-muted-foreground">
              <ListTodo className="mr-1 inline h-3.5 w-3.5" />
              Action Items ({actionItems.length})
            </h4>
            <div className="space-y-1.5">
              {actionItems.slice(0, 5).map((item) => (
                <div
                  key={item.id}
                  className={`flex items-center gap-2 text-sm ${
                    item.is_completed ? 'line-through text-muted-foreground' : 'text-foreground'
                  }`}
                >
                  <CheckCircle2
                    className={`h-3.5 w-3.5 ${
                      item.is_completed ? 'text-green-500' : 'text-muted-foreground'
                    }`}
                  />
                  <span className="flex-1">{item.description}</span>
                  {item.task_id && (
                    <Badge variant="outline" className="text-[10px]">
                      Task
                    </Badge>
                  )}
                </div>
              ))}
              {actionItems.length > 5 && (
                <p className="text-xs text-muted-foreground">
                  +{actionItems.length - 5} weitere
                </p>
              )}
            </div>
          </div>
        </>
      )}

      {/* Recordings */}
      {recordings.length > 0 && (
        <>
          <Separator className="my-4" />
          <div className="mb-4">
            <h4 className="mb-2 text-xs font-medium uppercase text-muted-foreground">
              <FileText className="mr-1 inline h-3.5 w-3.5" />
              Aufnahmen ({recordings.length})
            </h4>
            <div className="space-y-1.5">
              {recordings.map((rec) => (
                <div
                  key={rec.id}
                  className="flex items-center justify-between rounded-md border border-border px-3 py-2 text-sm hover:bg-secondary/50 cursor-pointer"
                >
                  <div className="flex items-center gap-2">
                    <PlayCircle className="h-4 w-4 text-primary" />
                    <span className="text-foreground">
                      {rec.duration_seconds
                        ? formatDuration(Math.round(rec.duration_seconds / 60))
                        : 'Aufnahme'}
                    </span>
                  </div>
                  <div className="flex items-center gap-2">
                    {rec.file_size_bytes && (
                      <span className="text-xs text-muted-foreground">
                        {(rec.file_size_bytes / 1024 / 1024).toFixed(1)} MB
                      </span>
                    )}
                    <Badge
                      variant={rec.status === 'completed' ? 'secondary' : 'outline'}
                      className="text-[10px]"
                    >
                      {rec.status}
                    </Badge>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </>
      )}
    </DetailPanel>
  )
}
