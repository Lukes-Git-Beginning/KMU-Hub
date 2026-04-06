/**
 * MeetingSummaryView -- Post-meeting summary display.
 *
 * Shows meeting duration, attendees, notes, action items, and recordings.
 * Organizer can edit summary notes and finalize.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Clock,
  Users,
  FileText,
  ListTodo,
  PlayCircle,
  CheckCircle2,
  ExternalLink,
  Save,
  Lock,
} from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { Textarea } from '@/components/ui/textarea'
import {
  useMeeting,
  useMeetingNotes,
  useActionItems,
  useSaveMeetingNotes,
} from '@/api/hooks/useMeetings'
import { useRecordings } from '@/api/hooks/useVideo'
import { useAuthStore } from '@/stores/auth'
import type { MeetingSummary } from '@/api/video-types'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface MeetingSummaryViewProps {
  meetingId: string
  summary?: MeetingSummary | null
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function formatDuration(minutes: number): string {
  const h = Math.floor(minutes / 60)
  const m = minutes % 60
  if (h > 0 && m > 0) return `${h}h ${m}min`
  if (h > 0) return `${h}h`
  return `${m}min`
}

function formatDate(isoDate: string): string {
  return new Date(isoDate).toLocaleDateString('de-DE', {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  })
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function MeetingSummaryView({ meetingId, summary }: MeetingSummaryViewProps) {
  const { t } = useTranslation()
  const { data: meeting } = useMeeting(meetingId)
  const { data: notes } = useMeetingNotes(meetingId)
  const { data: actionItems = [] } = useActionItems(meetingId)
  const { data: recordings = [] } = useRecordings({ meetingId })
  const saveMutation = useSaveMeetingNotes()
  const currentUserId = useAuthStore((s) => s.user?.id ?? null)

  const [isEditing, setIsEditing] = useState(false)
  const [editContent, setEditContent] = useState('')
  const [isFinalized, setIsFinalized] = useState(false)

  if (!meeting) {
    return (
      <div className="flex items-center justify-center p-8">
        <div className="animate-spin h-6 w-6 border-2 border-primary border-t-transparent rounded-full" />
      </div>
    )
  }

  const isOrganizer = meeting.organizer_id === currentUserId
  const attendees = meeting.attendees ?? []
  const durationMin =
    summary?.duration_minutes ??
    (meeting.actual_start && meeting.actual_end
      ? Math.round(
          (new Date(meeting.actual_end).getTime() -
            new Date(meeting.actual_start).getTime()) /
            60000,
        )
      : null)
  const completedCount = actionItems.filter((i) => i.is_completed).length
  const convertedCount = actionItems.filter((i) => i.task_id).length

  const handleSaveNotes = () => {
    saveMutation.mutate(
      {
        meetingId,
        data: { content: editContent, is_private: notes?.is_private ?? false },
      },
      {
        onSuccess: () => {
          toast.success(t('meetings.notes.notesSaved'))
          setIsEditing(false)
        },
        onError: () => toast.error(t('meetings.notes.saveError')),
      },
    )
  }

  const handleFinalize = () => {
    setIsFinalized(true)
    toast.success(t('meetings.summary.finalized'))
  }

  return (
    <div className="flex-1 overflow-y-auto p-6 max-w-3xl mx-auto">
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-xl font-semibold text-foreground mb-1">{meeting.title}</h1>
        <p className="text-sm text-muted-foreground">
          {formatDate(meeting.scheduled_start)}
        </p>
        {meeting.status === 'completed' && (
          <Badge variant="secondary" className="mt-2">
            {t('meetings.summary.completed')}
          </Badge>
        )}
      </div>

      {/* Duration Card */}
      {durationMin !== null && (
        <div className="rounded-lg border border-border bg-card p-4 mb-6">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
              <Clock className="h-5 w-5" />
            </div>
            <div>
              <p className="text-2xl font-semibold text-foreground">
                {formatDuration(durationMin)}
              </p>
              <p className="text-xs text-muted-foreground">{t('meetings.summary.duration')}</p>
            </div>
          </div>
        </div>
      )}

      {/* Attendees */}
      <section className="mb-6">
        <h2 className="flex items-center gap-1.5 text-sm font-medium text-foreground mb-3">
          <Users className="h-4 w-4" />
          {t('meetings.summary.attendees')} ({summary?.attendee_count ?? attendees.length})
        </h2>
        <div className="flex flex-wrap gap-2">
          {attendees.map((a) => (
            <div
              key={a.user_id}
              className="flex items-center gap-2 rounded-full border border-border bg-card px-3 py-1.5"
            >
              <span className="flex h-6 w-6 items-center justify-center rounded-full bg-primary/10 text-[10px] font-medium text-primary">
                {(a.user_name ?? a.user_id).slice(0, 2).toUpperCase()}
              </span>
              <span className="text-sm text-foreground">
                {a.user_name ?? a.user_id.slice(0, 8)}
              </span>
              <Badge
                variant="outline"
                className={`text-[9px] ${
                  a.rsvp_status === 'accepted'
                    ? 'bg-green-50 text-green-700'
                    : 'text-muted-foreground'
                }`}
              >
                {a.rsvp_status === 'accepted' ? t('meetings.summary.present') : a.rsvp_status}
              </Badge>
            </div>
          ))}
        </div>
      </section>

      <Separator className="my-6" />

      {/* Notes */}
      <section className="mb-6">
        <div className="flex items-center justify-between mb-3">
          <h2 className="flex items-center gap-1.5 text-sm font-medium text-foreground">
            <FileText className="h-4 w-4" />
            {t('meetings.notes.title')}
          </h2>
          {isOrganizer && !isFinalized && !isEditing && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setEditContent(notes?.content ?? summary?.notes_summary ?? '')
                setIsEditing(true)
              }}
            >
              {t('common.edit')}
            </Button>
          )}
        </div>

        {isEditing ? (
          <div className="space-y-2">
            <Textarea
              value={editContent}
              onChange={(e) => setEditContent(e.target.value)}
              rows={6}
              className="text-sm"
            />
            <div className="flex gap-2">
              <Button size="sm" onClick={handleSaveNotes} disabled={saveMutation.isPending}>
                <Save className="mr-1.5 h-3.5 w-3.5" />
                {saveMutation.isPending ? t('meetings.notes.saving') : t('common.save')}
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setIsEditing(false)}
              >
                {t('common.cancel')}
              </Button>
            </div>
          </div>
        ) : (
          <div className="rounded-lg border border-border bg-card p-4 text-sm text-foreground whitespace-pre-wrap">
            {notes?.content ??
              summary?.notes_summary ??
              t('meetings.notes.noNotes')}
            {notes?.is_private && (
              <div className="mt-2 flex items-center gap-1 text-xs text-muted-foreground">
                <Lock className="h-3 w-3" />
                {t('meetings.notes.private')}
              </div>
            )}
          </div>
        )}
      </section>

      <Separator className="my-6" />

      {/* Action Items */}
      <section className="mb-6">
        <h2 className="flex items-center gap-1.5 text-sm font-medium text-foreground mb-3">
          <ListTodo className="h-4 w-4" />
          {t('meetings.summary.actionItems')} ({actionItems.length})
          {actionItems.length > 0 && (
            <span className="text-xs text-muted-foreground ml-1">
              {completedCount}/{actionItems.length} {t('meetings.summary.done')}
              {convertedCount > 0 && ` | ${convertedCount} ${t('meetings.summary.asTask')}`}
            </span>
          )}
        </h2>
        <div className="space-y-2">
          {actionItems.map((item) => (
            <div
              key={item.id}
              className={`flex items-center gap-2 rounded-md border border-border p-2.5 ${
                item.is_completed ? 'bg-green-50/50' : 'bg-card'
              }`}
            >
              <CheckCircle2
                className={`h-4 w-4 shrink-0 ${
                  item.is_completed ? 'text-green-500' : 'text-muted-foreground'
                }`}
              />
              <span
                className={`text-sm flex-1 ${
                  item.is_completed ? 'line-through text-muted-foreground' : 'text-foreground'
                }`}
              >
                {item.description}
              </span>
              {item.assignee_id && (
                <span className="flex h-5 w-5 items-center justify-center rounded-full bg-primary/10 text-[8px] font-medium text-primary shrink-0">
                  {item.assignee_id.slice(0, 2).toUpperCase()}
                </span>
              )}
              {item.task_id && (
                <Badge variant="outline" className="text-[10px] shrink-0">
                  <ExternalLink className="mr-0.5 h-2.5 w-2.5" />
                  Task
                </Badge>
              )}
            </div>
          ))}
          {actionItems.length === 0 && (
            <p className="text-sm text-muted-foreground">{t('meetings.summary.noActionItems')}</p>
          )}
        </div>
      </section>

      {/* Recordings */}
      {recordings.length > 0 && (
        <>
          <Separator className="my-6" />
          <section className="mb-6">
            <h2 className="flex items-center gap-1.5 text-sm font-medium text-foreground mb-3">
              <PlayCircle className="h-4 w-4" />
              {t('meetings.summary.recordings')} ({recordings.length})
            </h2>
            <div className="space-y-2">
              {recordings.map((rec) => (
                <div
                  key={rec.id}
                  className="flex items-center justify-between rounded-md border border-border bg-card p-3 hover:bg-secondary/30 cursor-pointer transition-colors"
                >
                  <div className="flex items-center gap-2">
                    <PlayCircle className="h-5 w-5 text-primary" />
                    <div>
                      <p className="text-sm text-foreground">
                        {rec.duration_seconds
                          ? formatDuration(Math.round(rec.duration_seconds / 60))
                          : t('meetings.summary.recording')}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {new Date(rec.created_at).toLocaleString('de-DE')}
                      </p>
                    </div>
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
          </section>
        </>
      )}

      {/* Finalize Button (organizer only) */}
      {isOrganizer && !isFinalized && (
        <>
          <Separator className="my-6" />
          <div className="flex justify-center">
            <Button onClick={handleFinalize} className="px-8">
              {t('meetings.summary.finalize')}
            </Button>
          </div>
        </>
      )}

      {isFinalized && (
        <div className="text-center py-4">
          <Badge variant="secondary" className="text-sm">
            {t('meetings.summary.finalized')}
          </Badge>
        </div>
      )}
    </div>
  )
}
