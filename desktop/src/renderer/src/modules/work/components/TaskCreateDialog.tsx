/**
 * Task creation dialog with title, status, priority, assignee, and due date.
 *
 * Uses Radix Dialog via shadcn pattern. On submit calls useCreateTask mutation
 * with the project_id. Supports optional parent task for subtask creation.
 */
import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
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
import { memberDisplayName } from '../lib/task-helpers'
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
  const { t } = useTranslation()
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

   
  useEffect(() => {
    if (statuses.length > 0 && !statusId) {
      const defaultStatus =
        statuses.find((s) => s.is_default) ??
        statuses.find((s) => !s.is_closed) ??
        statuses[0]
      if (defaultStatus?.id) {
        // eslint-disable-next-line react-hooks/set-state-in-effect -- sync form fields from prop/API data
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
      setError(t('work.tasks.titleRequired'))
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
          : t('work.tasks.createError')
      )
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>
            {parentTaskId ? t('work.tasks.newSubtask') : t('work.tasks.newTask')}
          </DialogTitle>
          <DialogDescription>
            {parentTaskId
              ? t('work.tasks.newSubtaskDescription')
              : t('work.tasks.newTaskDescription')}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Title */}
          <div className="space-y-2">
            <Label htmlFor="task-title">{t('work.tasks.titleLabel')} *</Label>
            <Input
              id="task-title"
              placeholder={t('work.tasks.titlePlaceholder')}
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              autoFocus
            />
          </div>

          {/* Status + Priority row */}
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>{t('common.status')}</Label>
              <Select value={statusId} onValueChange={setStatusId}>
                <SelectTrigger>
                  <SelectValue placeholder={t('work.tasks.selectStatus')} />
                </SelectTrigger>
                <SelectContent>
                  {statuses
                    .filter((s) => s.id)
                    .map((s) => (
                    <SelectItem key={s.id} value={s.id!}>
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
              <Label>{t('work.tasks.priority')}</Label>
              <Select
                value={priority}
                onValueChange={(v) => setPriority(v as Priority)}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="urgent">{t('work.priority.urgent')}</SelectItem>
                  <SelectItem value="high">{t('work.priority.high')}</SelectItem>
                  <SelectItem value="normal">{t('work.priority.normal')}</SelectItem>
                  <SelectItem value="low">{t('work.priority.low')}</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Assignee + Due date row */}
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>{t('work.tasks.assignee')}</Label>
              <Select
                value={assigneeId || '__none__'}
                onValueChange={(v) => setAssigneeId(v === '__none__' ? '' : v)}
              >
                <SelectTrigger>
                  <SelectValue placeholder={t('work.tasks.unassigned')} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="__none__">{t('work.tasks.unassigned')}</SelectItem>
                  {members
                    .filter((m) => m.user_id)
                    .map((m) => (
                      <SelectItem key={m.user_id} value={m.user_id!}>
                        {memberDisplayName(m) || t('work.tasks.user')}
                      </SelectItem>
                    ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="task-due-date">{t('work.tasks.dueDate')}</Label>
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
              <Label htmlFor="task-description">{t('work.tasks.description')}</Label>
              <Textarea
                id="task-description"
                placeholder={t('work.tasks.descriptionPlaceholder')}
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
              {t('work.tasks.addDescription')}
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
              {t('common.cancel')}
            </Button>
            <Button type="submit" disabled={createTask.isPending}>
              {createTask.isPending ? t('work.tasks.creating') : t('work.tasks.createTask')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
