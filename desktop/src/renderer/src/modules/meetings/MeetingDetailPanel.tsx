import {
  Calendar,
  Clock,
  MapPin,
  Video,
  Users,
  FileText,
  FolderOpen,
  Presentation,
  Pencil,
  Trash2,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { DetailPanel } from '@/components/shared'
import type { Meeting } from '@/stores/meetings'

interface MeetingDetailPanelProps {
  meeting: Meeting | null
  open: boolean
  onClose: () => void
  onEdit: (meeting: Meeting) => void
  onDelete: (meetingId: string) => void
  onJoin: (meetingId: string) => void
}

const statusConfig = {
  live: { label: 'Live', variant: 'destructive' as const, dot: 'bg-red-500' },
  scheduled: { label: 'Geplant', variant: 'secondary' as const, dot: 'bg-blue-500' },
  past: { label: 'Vergangen', variant: 'outline' as const, dot: 'bg-gray-400' },
  cancelled: { label: 'Abgesagt', variant: 'outline' as const, dot: 'bg-gray-400' },
}

const recurrenceLabels: Record<string, string> = {
  none: 'Einmalig',
  daily: 'Taeglich',
  weekly: 'Woechentlich',
  monthly: 'Monatlich',
}

const reminderLabels: Record<string, string> = {
  none: 'Keine',
  '15min': '15 Min vorher',
  '30min': '30 Min vorher',
  '1h': '1 Std vorher',
}

export function MeetingDetailPanel({
  meeting,
  open,
  onClose,
  onEdit,
  onDelete,
  onJoin,
}: MeetingDetailPanelProps) {
  if (!meeting) return null

  const status = statusConfig[meeting.status]

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
          {meeting.status === 'live' && (
            <Button className="flex-1" onClick={() => onJoin(meeting.id)}>
              <Video className="mr-1.5 h-4 w-4" />
              Beitreten
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
          <Calendar className="h-4 w-4 text-[var(--muted)]" />
          <span className="text-[var(--body)]">
            {new Date(meeting.date).toLocaleDateString('de-CH', {
              weekday: 'long',
              day: 'numeric',
              month: 'long',
              year: 'numeric',
            })}
          </span>
        </div>
        <div className="flex items-center gap-2 text-sm">
          <Clock className="h-4 w-4 text-[var(--muted)]" />
          <span className="text-[var(--body)]">
            {meeting.startTime} Uhr &middot; {meeting.duration >= 60 ? `${meeting.duration / 60} Std` : `${meeting.duration} Min`}
          </span>
        </div>
        <div className="flex items-center gap-2 text-sm">
          <MapPin className="h-4 w-4 text-[var(--muted)]" />
          <span className="text-[var(--body)]">{meeting.room}</span>
          {meeting.isVideoCall && (
            <Badge variant="outline" className="ml-1 text-xs">
              <Video className="mr-1 h-3 w-3" />
              Video
            </Badge>
          )}
        </div>
        {meeting.recurrence !== 'none' && (
          <div className="flex items-center gap-2 text-sm">
            <Clock className="h-4 w-4 text-[var(--muted)]" />
            <span className="text-[var(--body)]">{recurrenceLabels[meeting.recurrence]}</span>
          </div>
        )}
        <div className="flex items-center gap-2 text-sm">
          <Clock className="h-4 w-4 text-[var(--muted)]" />
          <span className="text-[var(--body)]">Erinnerung: {reminderLabels[meeting.reminder]}</span>
        </div>
      </div>

      <Separator className="my-4" />

      {/* Project */}
      {meeting.project && (
        <>
          <div className="mb-4">
            <h4 className="mb-1.5 text-xs font-medium uppercase text-[var(--muted)]">Projekt</h4>
            <div className="flex items-center gap-2 text-sm text-primary">
              <FolderOpen className="h-4 w-4" />
              {meeting.project}
            </div>
          </div>
        </>
      )}

      {/* Participants */}
      <div className="mb-4">
        <h4 className="mb-2 text-xs font-medium uppercase text-[var(--muted)]">
          <Users className="mr-1 inline h-3.5 w-3.5" />
          Teilnehmer ({meeting.participants.length})
        </h4>
        <div className="space-y-1.5">
          {meeting.participants.map((p) => (
            <div key={p.id} className="flex items-center gap-2">
              <span className="flex h-7 w-7 items-center justify-center rounded-full bg-primary/10 text-[10px] font-medium text-primary">
                {p.initials}
              </span>
              <span className="text-sm text-[var(--body)]">{p.name}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Description */}
      {meeting.description && (
        <>
          <Separator className="my-4" />
          <div className="mb-4">
            <h4 className="mb-1.5 text-xs font-medium uppercase text-[var(--muted)]">Beschreibung</h4>
            <p className="text-sm text-[var(--body)] whitespace-pre-wrap">{meeting.description}</p>
          </div>
        </>
      )}

      {/* Files */}
      {meeting.files.length > 0 && (
        <>
          <Separator className="my-4" />
          <div className="mb-4">
            <h4 className="mb-2 text-xs font-medium uppercase text-[var(--muted)]">
              <FileText className="mr-1 inline h-3.5 w-3.5" />
              Dateien ({meeting.files.length})
            </h4>
            <div className="space-y-1.5">
              {meeting.files.map((f) => (
                <div
                  key={f.id}
                  className="flex items-center justify-between rounded-md border border-border px-3 py-2 text-sm hover:bg-secondary/50 cursor-pointer"
                >
                  <span className="text-[var(--body)]">{f.name}</span>
                  <span className="text-xs text-[var(--muted)]">{f.size}</span>
                </div>
              ))}
            </div>
          </div>
        </>
      )}

      {/* Whiteboard */}
      <Separator className="my-4" />
      <div>
        <h4 className="mb-2 text-xs font-medium uppercase text-[var(--muted)]">
          <Presentation className="mr-1 inline h-3.5 w-3.5" />
          Whiteboard
        </h4>
        <Button variant="outline" size="sm" className="w-full">
          <Presentation className="mr-1.5 h-4 w-4" />
          Whiteboard starten
        </Button>
      </div>
    </DetailPanel>
  )
}
