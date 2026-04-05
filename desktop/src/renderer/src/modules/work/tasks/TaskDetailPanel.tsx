/**
 * Slide-over side panel for quick task viewing and editing.
 *
 * Opens from the right when a task is clicked in list or Kanban views.
 * Shows task metadata (editable), description, subtasks, and quick
 * comment input. Has an "Erweitern" button to navigate to full detail page.
 */
import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import {
  X,
  Maximize2,
  Calendar,
  User,
  ListTree,
  Send,
} from 'lucide-react'
import { cn } from '@/lib'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { useTask, useUpdateTask, useSubtasks } from '@/api/hooks/useTasks'
import { useProjectStatuses, useProjectMembers } from '@/api/hooks/useProjects'
import { useCreateComment } from '@/api/hooks/useTaskComments'
import { useWorkStore } from '@/stores/work'
import StatusBadge from '../components/StatusBadge'
import PriorityBadge from '../components/PriorityBadge'
import type { Priority } from '../components/PriorityBadge'

export default function TaskDetailPanel() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const {
    activeTaskId,
    taskPanelOpen,
    closeTaskPanel,
  } = useWorkStore()

  const { data: taskData, isLoading } = useTask(activeTaskId ?? '')
  const task = taskData?.task
  const projectId = task?.project_id ?? ''

  const { data: statusesData } = useProjectStatuses(projectId)
  const { data: membersData } = useProjectMembers(projectId)
  const { data: subtasksData } = useSubtasks(activeTaskId ?? '')
  const updateTask = useUpdateTask()
  const createComment = useCreateComment()

  const statuses = statusesData?.statuses ?? []
  const members = membersData?.members ?? []
  const subtasks = subtasksData?.tasks ?? []

  // Editable fields
  const [editingTitle, setEditingTitle] = useState(false)
  const [titleValue, setTitleValue] = useState('')
  const [editingDesc, setEditingDesc] = useState(false)
  const [descValue, setDescValue] = useState('')
  const [commentValue, setCommentValue] = useState('')

   
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- sync local editable state from prop
    setTitleValue(task?.title ?? '')
    setDescValue(task?.description ?? '')
  }, [task?.title, task?.description])

  // Close on Escape
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === 'Escape' && taskPanelOpen) {
        closeTaskPanel()
      }
    },
    [taskPanelOpen, closeTaskPanel]
  )

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])

  function handleTitleSave() {
    setEditingTitle(false)
    const trimmed = titleValue.trim()
    if (trimmed && trimmed !== task?.title && activeTaskId) {
      updateTask.mutate({ id: activeTaskId, title: trimmed })
    } else {
      setTitleValue(task?.title ?? '')
    }
  }

  function handleDescSave() {
    setEditingDesc(false)
    if (descValue !== (task?.description ?? '') && activeTaskId) {
      updateTask.mutate({ id: activeTaskId, description: descValue })
    }
  }

  function handleStatusChange(statusId: string) {
    if (activeTaskId) {
      updateTask.mutate({ id: activeTaskId, status_id: statusId })
    }
  }

  function handlePriorityChange(priority: string) {
    if (activeTaskId) {
      updateTask.mutate({
        id: activeTaskId,
        priority: priority as Priority,
      })
    }
  }

  function handleAssigneeChange(assigneeId: string) {
    if (activeTaskId) {
      updateTask.mutate({
        id: activeTaskId,
        assignee_id: assigneeId === '__none__' ? undefined : assigneeId,
      })
    }
  }

  function handleDueDateChange(date: string) {
    if (activeTaskId) {
      updateTask.mutate({
        id: activeTaskId,
        due_date: date || undefined,
      })
    }
  }

  async function handleQuickComment() {
    const content = commentValue.trim()
    if (!content || !activeTaskId) return
    try {
      await createComment.mutateAsync({ taskId: activeTaskId, content })
      setCommentValue('')
    } catch {
      // Error handled by mutation
    }
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

  function handleExpand() {
    if (activeTaskId && projectId) {
      closeTaskPanel()
      navigate(`/work/projects/${projectId}/tasks/${activeTaskId}`)
    }
  }

  if (!taskPanelOpen) return null

  const taskKey =
    task?.project_key && task?.task_number
      ? `${task.project_key}-${task.task_number}`
      : ''

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/20 z-40"
        onClick={closeTaskPanel}
      />

      {/* Panel */}
      <div
        className={cn(
          'fixed right-0 top-0 z-50 h-full w-[400px] bg-background border-l border-border shadow-xl',
          'transform transition-transform duration-200 ease-out',
          taskPanelOpen ? 'translate-x-0' : 'translate-x-full'
        )}
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-border px-4 py-3">
          <div className="flex items-center gap-2">
            {taskKey && (
              <Badge variant="outline" className="font-mono text-xs">
                {taskKey}
              </Badge>
            )}
          </div>
          <div className="flex items-center gap-1">
            <Button
              variant="ghost"
              size="sm"
              onClick={handleExpand}
              title={t('work.panel.expand')}
            >
              <Maximize2 className="h-4 w-4" />
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={closeTaskPanel}
              title={t('common.close')}
            >
              <X className="h-4 w-4" />
            </Button>
          </div>
        </div>

        {/* Content */}
        <div className="flex flex-col h-[calc(100%-57px)] overflow-y-auto">
          {isLoading ? (
            <div className="p-4 text-sm text-muted-foreground">
              {t('work.panel.loading')}
            </div>
          ) : !task ? (
            <div className="p-4 text-sm text-muted-foreground">
              {t('work.panel.notFound')}
            </div>
          ) : (
            <>
              <div className="flex-1 overflow-y-auto px-4 py-3 space-y-4">
                {/* Title (editable) */}
                {editingTitle ? (
                  <Input
                    className="text-lg font-semibold"
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
                  <h2
                    className="text-lg font-semibold cursor-pointer hover:bg-accent/30 rounded px-1 -mx-1 py-0.5 transition-colors"
                    onClick={() => setEditingTitle(true)}
                    title={t('work.panel.clickToEdit')}
                  >
                    {task.title}
                  </h2>
                )}

                {/* Metadata row */}
                <div className="grid grid-cols-2 gap-3">
                  {/* Status */}
                  <div className="space-y-1">
                    <label className="text-xs font-medium text-muted-foreground">
                      {t('common.status')}
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
                                style={{
                                  backgroundColor: s.color || '#6b7280',
                                }}
                              />
                              {s.name}
                            </button>
                          ))}
                        </div>
                      </PopoverContent>
                    </Popover>
                  </div>

                  {/* Priority */}
                  <div className="space-y-1">
                    <label className="text-xs font-medium text-muted-foreground">
                      {t('work.tasks.priority')}
                    </label>
                    <Popover>
                      <PopoverTrigger asChild>
                        <button type="button" className="cursor-pointer">
                          <PriorityBadge
                            priority={
                              (task.priority as Priority) ?? 'medium'
                            }
                          />
                        </button>
                      </PopoverTrigger>
                      <PopoverContent className="w-40 p-1" align="start">
                        <div className="space-y-0.5">
                          {(
                            ['urgent', 'high', 'medium', 'low'] as const
                          ).map((p) => (
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
                                  ? t('work.priority.urgent')
                                  : p === 'high'
                                    ? t('work.priority.high')
                                    : p === 'medium'
                                      ? t('work.priority.normal')
                                      : t('work.priority.low')}
                              </span>
                            </button>
                          ))}
                        </div>
                      </PopoverContent>
                    </Popover>
                  </div>

                  {/* Assignee */}
                  <div className="space-y-1">
                    <label className="text-xs font-medium text-muted-foreground">
                      {t('work.tasks.assignee')}
                    </label>
                    <Popover>
                      <PopoverTrigger asChild>
                        <button
                          type="button"
                          className="flex items-center gap-1.5 text-xs cursor-pointer hover:bg-accent/50 rounded px-1 py-0.5 transition-colors"
                        >
                          <User className="h-3.5 w-3.5 text-muted-foreground" />
                          <span>
                            {task.assignee_name || t('work.tasks.unassigned')}
                          </span>
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
                            {t('work.tasks.unassigned')}
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
                              {m.display_name || m.email || t('work.tasks.user')}
                            </button>
                          ))}
                        </div>
                      </PopoverContent>
                    </Popover>
                  </div>

                  {/* Due date */}
                  <div className="space-y-1">
                    <label className="text-xs font-medium text-muted-foreground">
                      {t('work.tasks.dueAt')}
                    </label>
                    <Popover>
                      <PopoverTrigger asChild>
                        <button
                          type="button"
                          className="flex items-center gap-1.5 text-xs cursor-pointer hover:bg-accent/50 rounded px-1 py-0.5 transition-colors"
                        >
                          <Calendar className="h-3.5 w-3.5 text-muted-foreground" />
                          <span>
                            {task.due_date
                              ? new Date(task.due_date).toLocaleDateString(
                                  'de-DE'
                                )
                              : t('work.tasks.noDate')}
                          </span>
                        </button>
                      </PopoverTrigger>
                      <PopoverContent className="w-auto p-3" align="start">
                        <div className="space-y-2">
                          <Input
                            type="date"
                            className="h-8 text-xs"
                            value={toDateInputValue(task.due_date)}
                            onChange={(e) =>
                              handleDueDateChange(e.target.value)
                            }
                          />
                          {task.due_date && (
                            <Button
                              variant="ghost"
                              size="sm"
                              className="h-7 w-full text-xs"
                              onClick={() => handleDueDateChange('')}
                            >
                              {t('work.tasks.removeDate')}
                            </Button>
                          )}
                        </div>
                      </PopoverContent>
                    </Popover>
                  </div>
                </div>

                {/* Description */}
                <div className="space-y-1">
                  <label className="text-xs font-medium text-muted-foreground">
                    {t('work.tasks.description')}
                  </label>
                  {editingDesc ? (
                    <div className="space-y-2">
                      <Textarea
                        value={descValue}
                        onChange={(e) => setDescValue(e.target.value)}
                        rows={4}
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
                          {t('common.cancel')}
                        </Button>
                        <Button size="sm" onClick={handleDescSave}>
                          {t('common.save')}
                        </Button>
                      </div>
                    </div>
                  ) : (
                    <div
                      className={cn(
                        'min-h-[40px] rounded-md border border-transparent px-2 py-1.5 text-sm cursor-pointer hover:bg-accent/30 hover:border-border transition-colors whitespace-pre-wrap',
                        !task.description && 'text-muted-foreground italic'
                      )}
                      onClick={() => setEditingDesc(true)}
                      title={t('work.panel.clickToEdit')}
                    >
                      {task.description || t('work.panel.addDescription')}
                    </div>
                  )}
                </div>

                {/* Subtasks */}
                <div className="space-y-1">
                  <div className="flex items-center justify-between">
                    <label className="text-xs font-medium text-muted-foreground flex items-center gap-1">
                      <ListTree className="h-3.5 w-3.5" />
                      {t('work.panel.subtasks')}
                      {subtasks.length > 0 && (
                        <span className="text-muted-foreground/70">
                          ({subtasks.length})
                        </span>
                      )}
                    </label>
                  </div>
                  {subtasks.length > 0 ? (
                    <div className="space-y-0.5">
                      {subtasks.map((st) => (
                        <div
                          key={st.id}
                          className="flex items-center gap-2 rounded px-2 py-1 text-sm hover:bg-accent/50 transition-colors cursor-pointer"
                          onClick={() => {
                            if (st.id) {
                              useWorkStore.getState().openTaskPanel(st.id)
                            }
                          }}
                        >
                          <span
                            className={cn(
                              'h-2 w-2 rounded-full shrink-0',
                              st.is_closed
                                ? 'bg-green-500'
                                : 'bg-muted-foreground/30'
                            )}
                          />
                          <span
                            className={cn(
                              'truncate flex-1',
                              st.is_closed && 'line-through text-muted-foreground'
                            )}
                          >
                            {st.title}
                          </span>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p className="text-xs text-muted-foreground py-1">
                      {t('work.panel.noSubtasks')}
                    </p>
                  )}
                </div>
              </div>

              {/* Quick comment input */}
              <div className="border-t border-border px-4 py-3">
                <div className="relative">
                  <Input
                    className="pr-10 text-sm"
                    placeholder={t('work.panel.commentPlaceholder')}
                    value={commentValue}
                    onChange={(e) => setCommentValue(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter' && !e.shiftKey) {
                        e.preventDefault()
                        handleQuickComment()
                      }
                    }}
                  />
                  <button
                    type="button"
                    className={cn(
                      'absolute right-2 top-1/2 -translate-y-1/2 rounded p-1 transition-colors',
                      commentValue.trim()
                        ? 'text-primary hover:bg-primary/10'
                        : 'text-muted-foreground/40 cursor-not-allowed'
                    )}
                    onClick={handleQuickComment}
                    disabled={
                      !commentValue.trim() || createComment.isPending
                    }
                  >
                    <Send className="h-4 w-4" />
                  </button>
                </div>
              </div>
            </>
          )}
        </div>
      </div>
    </>
  )
}
