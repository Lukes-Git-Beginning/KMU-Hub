/**
 * Task creation dialog with title, status, priority, assignee, and due date.
 *
 * Uses Radix Dialog via shadcn pattern. On submit calls useCreateTask mutation
 * with the project_id. Supports optional parent task for subtask creation.
 */
import { useState, useEffect } from 'react'
import { ChevronDown } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useCreateTask } from '@/api/hooks/useTasks'
import { useProjectMembers } from '@/api/hooks/useProjects'
import type { Priority } from './PriorityBadge'

interface StatusOption {
  id?: string
  name?: string
  color?: string
  is_default?: boolean
  is_closed?: boolean
}

interface TaskCreateDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectId: string
  statuses: StatusOption[]
  parentTaskId?: string
  onCreated?: () => void
}

export default function TaskCreateDialog({
  open,
  onOpenChange,
  projectId,
  statuses,
  parentTaskId,
  onCreated,
}: TaskCreateDialogProps) {
  const [title, setTitle] = useState('')
  const [statusId, setStatusId] = useState('')
  const [priority, setPriority] = useState<Priority>('normal')
  const [assigneeId, setAssigneeId] = useState('')
  const [dueDate, setDueDate] = useState('')
  const [description, setDescription] = useState('')
  const [showDescription, setShowDescription] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const createTask = useCreateTask()
  const { data: membersData } = useProjectMembers(projectId)
  const members = membersData?.members ?? []

  // Set default status to first non-closed or first status
  useEffect(() => {
    if (statuses.length > 0 && !statusId) {
      const defaultStatus =
        statuses.find((s) => s.is_default) ??
        statuses.find((s) => !s.is_closed) ??
        statuses[0]
      if (defaultStatus?.id) {
        setStatusId(defaultStatus.id)
      }
    }
  }, [statuses, statusId])

  function resetForm() {
    setTitle('')
    setStatusId('')
    setPriority('normal')
    setAssigneeId('')
    setDueDate('')
    setDescription('')
    setShowDescription(false)
    setError(null)
  }

  function handleOpenChange(nextOpen: boolean) {
    if (!nextOpen) resetForm()
    onOpenChange(nextOpen)
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)

    if (!title.trim()) {
      setError('Titel ist erforderlich.')
      return
    }

    try {
      await createTask.mutateAsync({
        title: title.trim(),
        project_id: projectId,
        status_id: statusId || undefined,
        priority,
        assignee_id: assigneeId || undefined,
        due_date: dueDate || undefined,
        description: description.trim() || undefined,
        parent_task_id: parentTaskId,
      })
      resetForm()
      onOpenChange(false)
      onCreated?.()
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : 'Aufgabe konnte nicht erstellt werden.'
      )
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>
            {parentTaskId ? 'Neue Unteraufgabe' : 'Neue Aufgabe'}
          </DialogTitle>
          <DialogDescription>
            {parentTaskId
              ? 'Erstelle eine Unteraufgabe fuer die ausgewaehlte Aufgabe.'
              : 'Erstelle eine neue Aufgabe in diesem Projekt.'}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Title */}
          <div className="space-y-2">
            <Label htmlFor="task-title">Titel *</Label>
            <Input
              id="task-title"
              placeholder="Was muss erledigt werden?"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              autoFocus
            />
          </div>

          {/* Status + Priority row */}
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>Status</Label>
              <Select value={statusId} onValueChange={setStatusId}>
                <SelectTrigger>
                  <SelectValue placeholder="Status waehlen" />
                </SelectTrigger>
                <SelectContent>
                  {statuses.map((s) => (
                    <SelectItem key={s.id} value={s.id ?? ''}>
                      <span className="flex items-center gap-2">
                        <span
                          className="h-2 w-2 rounded-full inline-block"
                          style={{ backgroundColor: s.color || '#6b7280' }}
                        />
                        {s.name}
                      </span>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label>Prioritaet</Label>
              <Select
                value={priority}
                onValueChange={(v) => setPriority(v as Priority)}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="urgent">Dringend</SelectItem>
                  <SelectItem value="high">Hoch</SelectItem>
                  <SelectItem value="normal">Normal</SelectItem>
                  <SelectItem value="low">Niedrig</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Assignee + Due date row */}
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>Zustaendig</Label>
              <Select value={assigneeId} onValueChange={setAssigneeId}>
                <SelectTrigger>
                  <SelectValue placeholder="Nicht zugewiesen" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="">Nicht zugewiesen</SelectItem>
                  {members.map((m) => (
                    <SelectItem key={m.user_id} value={m.user_id ?? ''}>
                      {m.display_name || m.email || 'Benutzer'}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="task-due-date">Faellig am</Label>
              <Input
                id="task-due-date"
                type="date"
                value={dueDate}
                onChange={(e) => setDueDate(e.target.value)}
              />
            </div>
          </div>

          {/* Description (collapsible) */}
          {showDescription ? (
            <div className="space-y-2">
              <Label htmlFor="task-description">Beschreibung</Label>
              <Textarea
                id="task-description"
                placeholder="Optionale Beschreibung..."
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                rows={3}
              />
            </div>
          ) : (
            <button
              type="button"
              className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors"
              onClick={() => setShowDescription(true)}
            >
              <ChevronDown className="h-3 w-3" />
              Beschreibung hinzufuegen
            </button>
          )}

          {/* Error */}
          {error && <p className="text-sm text-destructive">{error}</p>}

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => handleOpenChange(false)}
            >
              Abbrechen
            </Button>
            <Button type="submit" disabled={createTask.isPending}>
              {createTask.isPending ? 'Erstelle...' : 'Aufgabe erstellen'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
