/**
 * MeetingFormDialog -- create or edit a meeting.
 *
 * Adapted from Darien's design (design/brainstorm) to use real backend API
 * via useCreateMeeting/useUpdateMeeting mutations. Replaces mock participants
 * with attendee_ids array for backend API.
 */
import { useState, useEffect } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Switch } from '@/components/ui/switch'
import { ChevronDown, ChevronUp, Plus, X } from 'lucide-react'
import { toast } from 'sonner'
import { useCreateMeeting, useUpdateMeeting } from '@/api/hooks/useMeetings'
import type { Meeting } from '@/api/video-types'

interface MeetingFormDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  meeting?: Meeting | null
}

export function MeetingFormDialog({ open, onOpenChange, meeting }: MeetingFormDialogProps) {
  const isEdit = !!meeting
  const createMutation = useCreateMeeting()
  const updateMutation = useUpdateMeeting()

  // Form state
  const [title, setTitle] = useState('')
  const [scheduledStart, setScheduledStart] = useState('')
  const [startTime, setStartTime] = useState('09:00')
  const [scheduledEnd, setScheduledEnd] = useState('')
  const [endTime, setEndTime] = useState('10:00')
  const [agenda, setAgenda] = useState('')
  const [description, setDescription] = useState('')
  const [attendeeInput, setAttendeeInput] = useState('')
  const [attendeeIds, setAttendeeIds] = useState<string[]>([])
  const [showExtras, setShowExtras] = useState(false)
  const [calendarEventId, setCalendarEventId] = useState('')
  const [recurringMeetingId, setRecurringMeetingId] = useState('')

  // Reset form when dialog opens/closes or meeting changes
  useEffect(() => {
    if (meeting) {
      setTitle(meeting.title)
      const start = new Date(meeting.scheduled_start)
      const end = new Date(meeting.scheduled_end)
      setScheduledStart(start.toISOString().split('T')[0])
      setStartTime(
        start.toLocaleTimeString('de-CH', { hour: '2-digit', minute: '2-digit' }),
      )
      setScheduledEnd(end.toISOString().split('T')[0])
      setEndTime(
        end.toLocaleTimeString('de-CH', { hour: '2-digit', minute: '2-digit' }),
      )
      setAgenda(meeting.agenda ?? '')
      setDescription(meeting.description ?? '')
      setAttendeeIds(meeting.attendees?.map((a) => a.user_id) ?? [])
      setCalendarEventId(meeting.calendar_event_id ?? '')
      setRecurringMeetingId(meeting.recurring_meeting_id ?? '')
    } else {
      setTitle('')
      setScheduledStart(new Date().toISOString().split('T')[0])
      setStartTime('09:00')
      setScheduledEnd(new Date().toISOString().split('T')[0])
      setEndTime('10:00')
      setAgenda('')
      setDescription('')
      setAttendeeIds([])
      setAttendeeInput('')
      setShowExtras(false)
      setCalendarEventId('')
      setRecurringMeetingId('')
    }
  }, [meeting, open])

  /** Build full ISO datetime from date + time strings. */
  function buildISO(date: string, time: string): string {
    return `${date}T${time}:00Z`
  }

  const handleSubmit = () => {
    if (!title.trim()) return

    const startISO = buildISO(scheduledStart, startTime)
    const endISO = buildISO(scheduledEnd || scheduledStart, endTime)

    // Validate: start must be before end
    if (new Date(startISO) >= new Date(endISO)) {
      toast.error('Startzeit muss vor der Endzeit liegen')
      return
    }

    if (isEdit && meeting) {
      updateMutation.mutate(
        {
          id: meeting.id,
          data: {
            title: title.trim(),
            description: description || undefined,
            agenda: agenda || undefined,
            scheduled_start: startISO,
            scheduled_end: endISO,
          },
        },
        {
          onSuccess: () => {
            toast.success('Meeting aktualisiert')
            onOpenChange(false)
          },
          onError: () => toast.error('Fehler beim Aktualisieren'),
        },
      )
    } else {
      createMutation.mutate(
        {
          title: title.trim(),
          description: description || undefined,
          agenda: agenda || undefined,
          scheduled_start: startISO,
          scheduled_end: endISO,
          attendee_ids: attendeeIds,
          calendar_event_id: calendarEventId || undefined,
          recurring_meeting_id: recurringMeetingId || undefined,
        },
        {
          onSuccess: () => {
            toast.success('Meeting erstellt')
            onOpenChange(false)
          },
          onError: () => toast.error('Fehler beim Erstellen'),
        },
      )
    }
  }

  const addAttendee = () => {
    const trimmed = attendeeInput.trim()
    if (trimmed && !attendeeIds.includes(trimmed)) {
      setAttendeeIds((prev) => [...prev, trimmed])
      setAttendeeInput('')
    }
  }

  const removeAttendee = (id: string) => {
    setAttendeeIds((prev) => prev.filter((a) => a !== id))
  }

  const isPending = createMutation.isPending || updateMutation.isPending

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{isEdit ? 'Meeting bearbeiten' : 'Neues Meeting'}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2">
          {/* Title */}
          <div className="space-y-1.5">
            <Label>Titel *</Label>
            <Input
              placeholder="z.B. Sprint Planning"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              autoFocus
            />
          </div>

          {/* Date + Time */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Startdatum</Label>
              <Input
                type="date"
                value={scheduledStart}
                onChange={(e) => setScheduledStart(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label>Startzeit</Label>
              <Input
                type="time"
                value={startTime}
                onChange={(e) => setStartTime(e.target.value)}
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Enddatum</Label>
              <Input
                type="date"
                value={scheduledEnd || scheduledStart}
                onChange={(e) => setScheduledEnd(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label>Endzeit</Label>
              <Input
                type="time"
                value={endTime}
                onChange={(e) => setEndTime(e.target.value)}
              />
            </div>
          </div>

          {/* Agenda */}
          <div className="space-y-1.5">
            <Label>Agenda</Label>
            <Textarea
              placeholder="Meeting-Agenda eingeben..."
              value={agenda}
              onChange={(e) => setAgenda(e.target.value)}
              rows={3}
            />
          </div>

          {/* Attendees */}
          <div className="space-y-1.5">
            <Label>Teilnehmer (User-IDs)</Label>
            <div className="flex flex-wrap gap-1.5 mb-2">
              {attendeeIds.map((id) => (
                <span
                  key={id}
                  className="inline-flex items-center gap-1 rounded-full bg-primary/10 px-2.5 py-1 text-xs font-medium text-primary"
                >
                  {id.slice(0, 8)}...
                  <button
                    onClick={() => removeAttendee(id)}
                    className="ml-0.5 rounded-full hover:bg-primary/20 p-0.5"
                  >
                    <X className="h-3 w-3" />
                  </button>
                </span>
              ))}
            </div>
            <div className="flex gap-2">
              <Input
                placeholder="User-ID eingeben..."
                value={attendeeInput}
                onChange={(e) => setAttendeeInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault()
                    addAttendee()
                  }
                }}
                className="text-sm"
              />
              <Button variant="outline" size="sm" onClick={addAttendee}>
                <Plus className="h-4 w-4" />
              </Button>
            </div>
          </div>

          {/* Expandable extras */}
          <button
            onClick={() => setShowExtras(!showExtras)}
            className="flex items-center gap-1 text-sm text-primary hover:underline"
          >
            {showExtras ? (
              <ChevronUp className="h-3.5 w-3.5" />
            ) : (
              <ChevronDown className="h-3.5 w-3.5" />
            )}
            {showExtras ? 'Weniger Optionen' : 'Weitere Optionen'}
          </button>

          {showExtras && (
            <div className="space-y-4 rounded-lg border border-border p-3">
              {/* Description */}
              <div className="space-y-1.5">
                <Label>Beschreibung</Label>
                <Textarea
                  placeholder="Zusaetzliche Beschreibung..."
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  rows={2}
                />
              </div>

              {/* Calendar Event Link */}
              <div className="space-y-1.5">
                <Label>Kalender-Event-ID (optional)</Label>
                <Input
                  placeholder="calendar_event_id..."
                  value={calendarEventId}
                  onChange={(e) => setCalendarEventId(e.target.value)}
                />
              </div>

              {/* Recurring Meeting Link */}
              <div className="space-y-1.5">
                <Label>Wiederkehrendes Meeting-ID (optional)</Label>
                <Input
                  placeholder="recurring_meeting_id..."
                  value={recurringMeetingId}
                  onChange={(e) => setRecurringMeetingId(e.target.value)}
                />
              </div>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button onClick={handleSubmit} disabled={!title.trim() || isPending}>
            <Plus className="mr-1.5 h-4 w-4" />
            {isPending ? 'Speichern...' : isEdit ? 'Speichern' : 'Meeting erstellen'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
