/**
 * ActivityFormDialog — Create/Edit Activity.
 *
 * Lightweight dialog with 4 fields: type, subject, description, due date.
 * Entity linking (contact/company/deal) is handled from entity detail pages.
 */
import { useState, useEffect } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { ACTIVITY_TYPES } from './activityUtils'

export interface ActivityFormData {
  activity_type: 'call' | 'meeting' | 'note' | 'email' | 'task'
  subject: string
  description: string
  due_date: string
}

interface ActivityFormDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  initialData?: Partial<ActivityFormData>
  onSubmit: (data: ActivityFormData) => void
  isEdit?: boolean
}

export function ActivityFormDialog({
  open,
  onOpenChange,
  initialData,
  onSubmit,
  isEdit = false,
}: ActivityFormDialogProps) {
  const [activityType, setActivityType] = useState<ActivityFormData['activity_type']>('call')
  const [subject, setSubject] = useState('')
  const [description, setDescription] = useState('')
  const [dueDate, setDueDate] = useState('')

  useEffect(() => {
    if (open && initialData) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- sync form fields from prop/API data
      setActivityType(initialData.activity_type ?? 'call')
      setSubject(initialData.subject ?? '')
      setDescription(initialData.description ?? '')
      setDueDate(initialData.due_date ?? '')
    } else if (open) {
      setActivityType('call')
      setSubject('')
      setDescription('')
      setDueDate('')
    }
  }, [open, initialData])

  const handleSubmit = () => {
    if (!subject.trim()) return
    onSubmit({
      activity_type: activityType,
      subject: subject.trim(),
      description: description.trim(),
      due_date: dueDate,
    })
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>
            {isEdit ? 'Aktivitaet bearbeiten' : 'Neue Aktivitaet'}
          </DialogTitle>
          <DialogDescription>
            {isEdit
              ? 'Aktivitaet aktualisieren.'
              : 'Erstelle eine neue Aktivitaet.'}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 pt-2">
          {/* Activity Type */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">Typ *</label>
            <select
              value={activityType}
              onChange={(e) =>
                setActivityType(e.target.value as ActivityFormData['activity_type'])
              }
              disabled={isEdit}
              className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none focus:border-primary disabled:opacity-50"
            >
              {ACTIVITY_TYPES.map((t) => (
                <option key={t.value} value={t.value}>
                  {t.label}
                </option>
              ))}
            </select>
          </div>

          {/* Subject */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">
              Betreff *
            </label>
            <input
              type="text"
              value={subject}
              onChange={(e) => setSubject(e.target.value)}
              placeholder="z.B. Rueckruf vereinbaren"
              className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none placeholder:text-muted-foreground focus:border-primary"
              autoFocus
            />
          </div>

          {/* Description */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">
              Beschreibung
            </label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Optionale Details..."
              rows={3}
              className="w-full rounded-md border border-border bg-transparent px-3 py-2 text-sm outline-none placeholder:text-muted-foreground focus:border-primary resize-none"
            />
          </div>

          {/* Due Date */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">
              Fällig am
            </label>
            <input
              type="date"
              value={dueDate}
              onChange={(e) => setDueDate(e.target.value)}
              className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none focus:border-primary"
            />
          </div>

          {/* Actions */}
          <div className="flex justify-end gap-2 pt-1">
            <button
              onClick={() => onOpenChange(false)}
              className="h-9 rounded-md border border-border px-4 text-sm text-foreground hover:bg-accent transition-colors"
            >
              Abbrechen
            </button>
            <button
              onClick={handleSubmit}
              disabled={!subject.trim()}
              className="h-9 rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              {isEdit ? 'Speichern' : 'Erstellen'}
            </button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
