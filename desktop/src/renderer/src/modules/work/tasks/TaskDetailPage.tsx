/**
 * Full task detail page with two-column layout for deep editing.
 *
 * Route: /work/projects/:projectId/tasks/:taskId
 * Left column: breadcrumb, title, description, subtasks, activity+comments feed.
 * Right column: status, priority, assignee, due date, dependencies, files, metadata.
 */
import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  ArrowLeft,
  Calendar,
  User,
  ListTree,
  Plus,
  Clock,
} from 'lucide-react'
import { cn } from '@/lib'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import {
  useTask,
  useUpdateTask,
  useSubtasks,
} from '@/api/hooks/useTasks'
import {
  useProject,
  useProjectStatuses,
  useProjectMembers,
} from '@/api/hooks/useProjects'
import StatusBadge from '../components/StatusBadge'
import PriorityBadge from '../components/PriorityBadge'
import type { Priority } from '../components/PriorityBadge'
import CommentThread from '../components/CommentThread'
import ActivityLog from '../components/ActivityLog'
import TaskFileAttachments from '../components/TaskFileAttachments'
import DependencyList from '../components/DependencyList'
import TaskCreateDialog from '../components/TaskCreateDialog'
import TaskLinkField from '../components/TaskLinkField'
import CustomFieldsSection from '../components/CustomFieldsSection'
import TaskTimer from '../components/TaskTimer'
import TimeEntryList from '../components/TimeEntryList'
import ManualTimeEntryDialog from '../components/ManualTimeEntryDialog'

export default function TaskDetailPage() {
  const { id: projectId, taskId } = useParams<{
    id: string
    taskId: string
  }>()
  const navigate = useNavigate()

  const { data: taskData, isLoading, error, refetch } = useTask(taskId ?? '')
  const task = taskData?.task
  const effectiveProjectId = projectId ?? task?.project_id ?? ''

  const { data: projectData } = useProject(effectiveProjectId)
  const { data: statusesData } = useProjectStatuses(effectiveProjectId)
  const { data: membersData } = useProjectMembers(effectiveProjectId)
  const { data: subtasksData } = useSubtasks(taskId ?? '')
  const updateTask = useUpdateTask()

  const project = projectData?.project
  const statuses = statusesData?.statuses ?? []
  const members = membersData?.members ?? []
  const subtasks = subtasksData?.tasks ?? []

  const [createSubtaskOpen, setCreateSubtaskOpen] = useState(false)
  const [manualTimeEntryOpen, setManualTimeEntryOpen] = useState(false)
  const [editingTitle, setEditingTitle] = useState(false)
  const [titleValue, setTitleValue] = useState('')
  const [editingDesc, setEditingDesc] = useState(false)
  const [descValue, setDescValue] = useState('')
  const [activeTab, setActiveTab] = useState<'combined' | 'comments' | 'activity'>('combined')

  useEffect(() => {
    setTitleValue(task?.title ?? '')
    setDescValue(task?.description ?? '')
  }, [task?.title, task?.description])

  function handleTitleSave() {
    setEditingTitle(false)
    const trimmed = titleValue.trim()
    if (trimmed && trimmed !== task?.title && taskId) {
      updateTask.mutate({ id: taskId, title: trimmed })
    } else {
      setTitleValue(task?.title ?? '')
    }
  }

  function handleDescSave() {
    setEditingDesc(false)
    if (descValue !== (task?.description ?? '') && taskId) {
      updateTask.mutate({ id: taskId, description: descValue })
    }
  }

  function handleStatusChange(statusId: string) {
    if (taskId) updateTask.mutate({ id: taskId, status_id: statusId })
  }

  function handlePriorityChange(priority: string) {
    if (taskId) updateTask.mutate({ id: taskId, priority: priority as Priority })
  }

  function handleAssigneeChange(assigneeId: string) {
    if (taskId) {
      updateTask.mutate({
        id: taskId,
        assignee_id: assigneeId === '__none__' ? undefined : assigneeId,
      })
    }
  }

  function handleDueDateChange(date: string) {
    if (taskId) updateTask.mutate({ id: taskId, due_date: date || undefined })
  }

  function toDateInputValue(date: unknown): string {
    if (!date) return ''
    try {
      const d = date instanceof Date ? date : new Date(String(date))
      if (isNaN(d.getTime())) return ''
      return d.toISOString().split('T')[0]
    } catch {
      return ''
    }
  }

  const taskKey =
    task?.project_key && task?.task_number
      ? `${task.project_key}-${task.task_number}`
      : ''

  if (error) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <div className="text-center">
          <p className="text-lg font-semibold text-foreground">
            Fehler beim Laden der Aufgabe
          </p>
          <p className="mt-2 text-sm text-muted-foreground">
            {error instanceof Error ? error.message : 'Ein unerwarteter Fehler ist aufgetreten.'}
          </p>
          <Button variant="outline" className="mt-4" onClick={() => refetch()}>
            Erneut versuchen
          </Button>
        </div>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="p-6 space-y-4">
        <Skeleton className="h-6 w-64" />
        <Skeleton className="h-10 w-96" />
        <Skeleton className="h-40 w-full" />
      </div>
    )
  }

  if (!task) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <div className="text-center">
          <p className="text-lg font-semibold text-foreground">
            Aufgabe nicht gefunden
          </p>
          <Button
            variant="outline"
            className="mt-4"
            onClick={() => navigate(`/work/projects/${effectiveProjectId}`)}
          >
            Zurueck zum Projekt
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col overflow-hidden">
      {/* Top breadcrumb bar */}
      <div className="flex items-center gap-2 border-b border-border px-6 py-2">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => navigate(`/work/projects/${effectiveProjectId}`)}
        >
          <ArrowLeft className="h-4 w-4 mr-1" />
          {project?.name ?? 'Projekt'}
        </Button>
        {taskKey && (
          <>
            <span className="text-muted-foreground">/</span>
            <Badge variant="outline" className="font-mono text-xs">
              {taskKey}
            </Badge>
          </>
        )}
      </div>

      {/* Two-column layout */}
      <div className="flex flex-1 min-h-0 overflow-hidden">
        {/* Main content (left) */}
        <div className="flex-1 overflow-y-auto px-6 py-4 space-y-6 min-w-0">
          {/* Title (editable) */}
          {editingTitle ? (
            <Input
              className="text-2xl font-bold h-auto py-1"
              value={titleValue}
              onChange={(e) => setTitleValue(e.target.value)}
              onBlur={handleTitleSave}
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleTitleSave()
                if (e.key === 'Escape') {
                  setTitleValue(task.title ?? '')
                  setEditingTitle(false)
                }
              }}
              autoFocus
            />
          ) : (
            <h1
              className="text-2xl font-bold cursor-pointer hover:bg-accent/30 rounded px-2 -mx-2 py-1 transition-colors"
              onClick={() => setEditingTitle(true)}
              title="Klicken zum Bearbeiten"
            >
              {task.title}
            </h1>
          )}

          {/* Description (editable) */}
          <div className="space-y-1">
            <label className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
              Beschreibung
            </label>
            {editingDesc ? (
              <div className="space-y-2">
                <Textarea
                  value={descValue}
                  onChange={(e) => setDescValue(e.target.value)}
                  rows={6}
                  className="text-sm"
                  autoFocus
                />
                <div className="flex gap-2">
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => {
                      setDescValue(task.description ?? '')
                      setEditingDesc(false)
                    }}
                  >
                    Abbrechen
                  </Button>
                  <Button size="sm" onClick={handleDescSave}>
                    Speichern
                  </Button>
                </div>
              </div>
            ) : (
              <div
                className={cn(
                  'min-h-[60px] rounded-md border border-transparent px-3 py-2 text-sm cursor-pointer hover:bg-accent/30 hover:border-border transition-colors whitespace-pre-wrap',
                  !task.description && 'text-muted-foreground italic'
                )}
                onClick={() => setEditingDesc(true)}
                title="Klicken zum Bearbeiten"
              >
                {task.description || 'Beschreibung hinzufuegen...'}
              </div>
            )}
          </div>

          {/* Subtasks */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <label className="text-xs font-medium text-muted-foreground uppercase tracking-wide flex items-center gap-1.5">
                <ListTree className="h-3.5 w-3.5" />
                Unteraufgaben
                {subtasks.length > 0 && (
                  <span>({subtasks.length})</span>
                )}
              </label>
              <Button
                variant="ghost"
                size="sm"
                className="h-7 gap-1 text-xs"
                onClick={() => setCreateSubtaskOpen(true)}
              >
                <Plus className="h-3.5 w-3.5" />
                Hinzufuegen
              </Button>
            </div>
            {subtasks.length > 0 ? (
              <div className="space-y-0.5 rounded-md border border-border">
                {subtasks.map((st) => (
                  <div
                    key={st.id}
                    className="flex items-center gap-3 px-3 py-2 text-sm hover:bg-accent/50 transition-colors cursor-pointer border-b border-border last:border-b-0"
                    onClick={() => {
                      if (st.id) {
                        navigate(
                          `/work/projects/${effectiveProjectId}/tasks/${st.id}`
                        )
                      }
                    }}
                  >
                    <span
                      className={cn(
                        'h-2.5 w-2.5 rounded-full shrink-0',
                        st.is_closed
                          ? 'bg-green-500'
                          : 'bg-muted-foreground/30'
                      )}
                    />
                    <span className="flex-1 truncate">
                      {st.project_key && st.task_number && (
                        <span className="font-mono text-xs text-muted-foreground mr-2">
                          {st.project_key}-{st.task_number}
                        </span>
                      )}
                      <span
                        className={cn(
                          st.is_closed && 'line-through text-muted-foreground'
                        )}
                      >
                        {st.title}
                      </span>
                    </span>
                    {st.assignee_name && (
                      <span className="text-xs text-muted-foreground flex items-center gap-1">
                        <User className="h-3 w-3" />
                        {st.assignee_name}
                      </span>
                    )}
                    <PriorityBadge
                      priority={(st.priority as Priority) ?? 'medium'}
                      compact
                    />
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm text-muted-foreground py-2">
                Keine Unteraufgaben vorhanden.
              </p>
            )}
          </div>

          <Separator />

          {/* Activity + Comments feed */}
          <div className="space-y-3">
            <div className="flex items-center gap-2">
              <button
                type="button"
                className={cn(
                  'text-xs font-medium px-2 py-1 rounded transition-colors',
                  activeTab === 'combined'
                    ? 'bg-secondary text-secondary-foreground'
                    : 'text-muted-foreground hover:text-foreground'
                )}
                onClick={() => setActiveTab('combined')}
              >
                Alle
              </button>
              <button
                type="button"
                className={cn(
                  'text-xs font-medium px-2 py-1 rounded transition-colors',
                  activeTab === 'comments'
                    ? 'bg-secondary text-secondary-foreground'
                    : 'text-muted-foreground hover:text-foreground'
                )}
                onClick={() => setActiveTab('comments')}
              >
                Kommentare
              </button>
              <button
                type="button"
                className={cn(
                  'text-xs font-medium px-2 py-1 rounded transition-colors',
                  activeTab === 'activity'
                    ? 'bg-secondary text-secondary-foreground'
                    : 'text-muted-foreground hover:text-foreground'
                )}
                onClick={() => setActiveTab('activity')}
              >
                Aktivitaet
              </button>
            </div>

            {(activeTab === 'combined' || activeTab === 'activity') && (
              <ActivityLog taskId={taskId ?? ''} />
            )}
            {(activeTab === 'combined' || activeTab === 'comments') && (
              <CommentThread
                taskId={taskId ?? ''}
                projectId={effectiveProjectId}
              />
            )}
          </div>
        </div>

        {/* Sidebar (right) */}
        <div className="w-[280px] shrink-0 border-l border-border overflow-y-auto px-4 py-4 space-y-5">
          {/* Status */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">
              Status
            </label>
            <Popover>
              <PopoverTrigger asChild>
                <button type="button" className="cursor-pointer">
                  <StatusBadge
                    name={task.status_name ?? 'Offen'}
                    color={task.status_color}
                    isClosed={task.is_closed}
                  />
                </button>
              </PopoverTrigger>
              <PopoverContent className="w-48 p-1" align="start">
                <div className="space-y-0.5">
                  {statuses.map((s) => (
                    <button
                      key={s.id}
                      type="button"
                      className={cn(
                        'flex w-full items-center gap-2 rounded px-2 py-1.5 text-xs hover:bg-accent transition-colors',
                        s.id === task.status_id && 'bg-accent'
                      )}
                      onClick={() => s.id && handleStatusChange(s.id)}
                    >
                      <span
                        className="h-2.5 w-2.5 rounded-full"
                        style={{ backgroundColor: s.color || '#6b7280' }}
                      />
                      {s.name}
                    </button>
                  ))}
                </div>
              </PopoverContent>
            </Popover>
          </div>

          {/* Priority */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">
              Prioritaet
            </label>
            <Popover>
              <PopoverTrigger asChild>
                <button type="button" className="cursor-pointer">
                  <PriorityBadge
                    priority={(task.priority as Priority) ?? 'medium'}
                  />
                </button>
              </PopoverTrigger>
              <PopoverContent className="w-40 p-1" align="start">
                <div className="space-y-0.5">
                  {(['urgent', 'high', 'medium', 'low'] as const).map((p) => (
                    <button
                      key={p}
                      type="button"
                      className={cn(
                        'flex w-full items-center gap-2 rounded px-2 py-1.5 text-xs hover:bg-accent transition-colors',
                        p === task.priority && 'bg-accent'
                      )}
                      onClick={() => handlePriorityChange(p)}
                    >
                      <PriorityBadge priority={p} compact />
                      <span>
                        {p === 'urgent'
                          ? 'Dringend'
                          : p === 'high'
                            ? 'Hoch'
                            : p === 'medium'
                              ? 'Normal'
                              : 'Niedrig'}
                      </span>
                    </button>
                  ))}
                </div>
              </PopoverContent>
            </Popover>
          </div>

          {/* Assignee */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">
              Zustaendig
            </label>
            <Popover>
              <PopoverTrigger asChild>
                <button
                  type="button"
                  className="flex items-center gap-1.5 text-sm cursor-pointer hover:bg-accent/50 rounded px-1 py-0.5 transition-colors"
                >
                  <User className="h-3.5 w-3.5 text-muted-foreground" />
                  <span>{task.assignee_name || 'Nicht zugewiesen'}</span>
                </button>
              </PopoverTrigger>
              <PopoverContent className="w-48 p-1" align="start">
                <div className="space-y-0.5">
                  <button
                    type="button"
                    className={cn(
                      'flex w-full items-center gap-2 rounded px-2 py-1.5 text-xs hover:bg-accent transition-colors',
                      !task.assignee_id && 'bg-accent'
                    )}
                    onClick={() => handleAssigneeChange('__none__')}
                  >
                    Nicht zugewiesen
                  </button>
                  {members.map((m) => (
                    <button
                      key={m.user_id}
                      type="button"
                      className={cn(
                        'flex w-full items-center gap-2 rounded px-2 py-1.5 text-xs hover:bg-accent transition-colors',
                        m.user_id === task.assignee_id && 'bg-accent'
                      )}
                      onClick={() =>
                        m.user_id && handleAssigneeChange(m.user_id)
                      }
                    >
                      {m.display_name || m.email || 'Benutzer'}
                    </button>
                  ))}
                </div>
              </PopoverContent>
            </Popover>
          </div>

          {/* Due date */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">
              Faellig am
            </label>
            <Popover>
              <PopoverTrigger asChild>
                <button
                  type="button"
                  className="flex items-center gap-1.5 text-sm cursor-pointer hover:bg-accent/50 rounded px-1 py-0.5 transition-colors"
                >
                  <Calendar className="h-3.5 w-3.5 text-muted-foreground" />
                  <span>
                    {task.due_date
                      ? new Date(task.due_date).toLocaleDateString('de-DE')
                      : 'Kein Datum'}
                  </span>
                </button>
              </PopoverTrigger>
              <PopoverContent className="w-auto p-3" align="start">
                <div className="space-y-2">
                  <Input
                    type="date"
                    className="h-8 text-xs"
                    value={toDateInputValue(task.due_date)}
                    onChange={(e) => handleDueDateChange(e.target.value)}
                  />
                  {task.due_date && (
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-7 w-full text-xs"
                      onClick={() => handleDueDateChange('')}
                    >
                      Datum entfernen
                    </Button>
                  )}
                </div>
              </PopoverContent>
            </Popover>
          </div>

          <Separator />

          {/* Dependencies */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">
              Abhaengigkeiten
            </label>
            <DependencyList
              taskId={taskId ?? ''}
              projectId={effectiveProjectId}
            />
          </div>

          <Separator />

          {/* Entity links */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">
              Verknuepfungen
            </label>
            <TaskLinkField
              taskId={taskId ?? ''}
              taskTitle={task.title}
              taskDescription={task.description}
            />
          </div>

          {/* Custom fields */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">
              Benutzerdefinierte Felder
            </label>
            <CustomFieldsSection
              taskId={taskId ?? ''}
              projectId={effectiveProjectId}
            />
          </div>

          <Separator />

          {/* File attachments */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">
              Dateien
            </label>
            <TaskFileAttachments taskId={taskId ?? ''} />
          </div>

          <Separator />

          {/* Time tracking */}
          <div className="space-y-1.5">
            <div className="flex items-center justify-between">
              <label className="text-xs font-medium text-muted-foreground">
                Zeiterfassung
              </label>
              <Button
                variant="ghost"
                size="sm"
                className="h-6 text-xs px-2"
                onClick={() => setManualTimeEntryOpen(true)}
              >
                + Manuell
              </Button>
            </div>
            <TaskTimer taskId={taskId ?? ''} />
          </div>

          {/* Time entries */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">
              Zeiteintraege
            </label>
            <TimeEntryList taskId={taskId ?? ''} />
          </div>

          <Separator />

          {/* Metadata */}
          <div className="space-y-2 text-xs text-muted-foreground">
            {task.created_by_name && (
              <div className="flex items-center gap-1.5">
                <User className="h-3 w-3" />
                <span>Erstellt von {task.created_by_name}</span>
              </div>
            )}
            {task.created_at && (
              <div className="flex items-center gap-1.5">
                <Clock className="h-3 w-3" />
                <span>
                  {new Date(task.created_at).toLocaleDateString('de-DE', {
                    day: '2-digit',
                    month: '2-digit',
                    year: 'numeric',
                    hour: '2-digit',
                    minute: '2-digit',
                  })}
                </span>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Subtask create dialog */}
      <TaskCreateDialog
        open={createSubtaskOpen}
        onOpenChange={setCreateSubtaskOpen}
        projectId={effectiveProjectId}
        statuses={statuses}
        parentTaskId={taskId}
      />

      {/* Manual time entry dialog */}
      <ManualTimeEntryDialog
        open={manualTimeEntryOpen}
        onOpenChange={setManualTimeEntryOpen}
        taskId={taskId ?? ''}
      />
    </div>
  )
}
