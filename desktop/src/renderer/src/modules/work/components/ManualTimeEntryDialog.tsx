/**
 * Dialog for adding or editing a manual time entry.
 *
 * Supports date, start time, duration (hours + minutes), and optional description.
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
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import {
  useAddManualTimeEntry,
  useUpdateTimeEntry,
} from '@/api/hooks/useTimeEntries'

interface ManualTimeEntryDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  taskId: string
  editMode?: boolean
  editEntryId?: string
  initialStartedAt?: string
  initialDurationSeconds?: number
  initialDescription?: string
}

export default function ManualTimeEntryDialog({
  open,
  onOpenChange,
  taskId,
  editMode = false,
  editEntryId,
  initialStartedAt,
  initialDurationSeconds,
  initialDescription,
}: ManualTimeEntryDialogProps) {
  const addEntry = useAddManualTimeEntry()
  const updateEntry = useUpdateTimeEntry()

  // Parse initial values
  const initDate = initialStartedAt
    ? new Date(initialStartedAt).toISOString().split('T')[0]
    : new Date().toISOString().split('T')[0]
  const initTime = initialStartedAt
    ? new Date(initialStartedAt).toLocaleTimeString('de-DE', {
        hour: '2-digit',
        minute: '2-digit',
        hour12: false,
      })
    : new Date().toLocaleTimeString('de-DE', {
        hour: '2-digit',
        minute: '2-digit',
        hour12: false,
      })
  const initHours = initialDurationSeconds
    ? Math.floor(initialDurationSeconds / 3600)
    : 0
  const initMinutes = initialDurationSeconds
    ? Math.floor((initialDurationSeconds % 3600) / 60)
    : 30

  const [date, setDate] = useState(initDate)
  const [time, setTime] = useState(initTime)
  const [hours, setHours] = useState(String(initHours))
  const [minutes, setMinutes] = useState(String(initMinutes))
  const [description, setDescription] = useState(initialDescription ?? '')

  // Reset form when dialog opens with new values
  useEffect(() => {
    if (open) {
      setDate(initDate)
      setTime(initTime)
      setHours(String(initHours))
      setMinutes(String(initMinutes))
      setDescription(initialDescription ?? '')
    }
  }, [open, initDate, initTime, initHours, initMinutes, initialDescription])

  function handleSubmit() {
    const h = parseInt(hours) || 0
    const m = parseInt(minutes) || 0
    const durationSeconds = h * 3600 + m * 60

    if (durationSeconds <= 0) return

    // Build ISO started_at from date + time
    const startedAt = `${date}T${time}:00`

    if (editMode && editEntryId) {
      updateEntry.mutate(
        {
          entryId: editEntryId,
          taskId,
          started_at: new Date(startedAt).toISOString(),
          duration_seconds: durationSeconds,
          description: description.trim() || undefined,
        },
        {
          onSuccess: () => onOpenChange(false),
        }
      )
    } else {
      addEntry.mutate(
        {
          taskId,
          started_at: new Date(startedAt).toISOString(),
          duration_seconds: durationSeconds,
          description: description.trim() || undefined,
        },
        {
          onSuccess: () => onOpenChange(false),
        }
      )
    }
  }

  const isSubmitting = addEntry.isPending || updateEntry.isPending

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[380px]">
        <DialogHeader>
          <DialogTitle>
            {editMode ? 'Zeiteintrag bearbeiten' : 'Zeiteintrag hinzufügen'}
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2">
          {/* Date */}
          <div className="space-y-1.5">
            <Label htmlFor="te-date" className="text-xs">
              Datum
            </Label>
            <Input
              id="te-date"
              type="date"
              value={date}
              onChange={(e) => setDate(e.target.value)}
              className="h-8 text-sm"
            />
          </div>

          {/* Start time */}
          <div className="space-y-1.5">
            <Label htmlFor="te-time" className="text-xs">
              Startzeit
            </Label>
            <Input
              id="te-time"
              type="time"
              value={time}
              onChange={(e) => setTime(e.target.value)}
              className="h-8 text-sm"
            />
          </div>

          {/* Duration */}
          <div className="space-y-1.5">
            <Label className="text-xs">Dauer</Label>
            <div className="flex items-center gap-2">
              <div className="flex items-center gap-1">
                <Input
                  type="number"
                  min="0"
                  max="23"
                  value={hours}
                  onChange={(e) => setHours(e.target.value)}
                  className="h-8 w-16 text-sm text-center"
                />
                <span className="text-xs text-muted-foreground">h</span>
              </div>
              <div className="flex items-center gap-1">
                <Input
                  type="number"
                  min="0"
                  max="59"
                  value={minutes}
                  onChange={(e) => setMinutes(e.target.value)}
                  className="h-8 w-16 text-sm text-center"
                />
                <span className="text-xs text-muted-foreground">min</span>
              </div>
            </div>
          </div>

          {/* Description */}
          <div className="space-y-1.5">
            <Label htmlFor="te-desc" className="text-xs">
              Beschreibung (optional)
            </Label>
            <Textarea
              id="te-desc"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Was wurde gemacht..."
              rows={3}
              className="text-sm resize-none"
              maxLength={2000}
            />
          </div>
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            size="sm"
            onClick={() => onOpenChange(false)}
          >
            Abbrechen
          </Button>
          <Button
            size="sm"
            onClick={handleSubmit}
            disabled={
              isSubmitting ||
              (parseInt(hours) || 0) * 3600 + (parseInt(minutes) || 0) * 60 <=
                0
            }
          >
            {editMode ? 'Speichern' : 'Hinzufuegen'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
