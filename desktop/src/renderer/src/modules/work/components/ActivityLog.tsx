/**
 * Chronological activity feed for a task.
 *
 * Shows task history: status changes, assignments, priority changes,
 * creation events, comments, and file attachments. Status changes are
 * visually prominent with color-coded indicators.
 */
import {
  ArrowRight,
  UserPlus,
  Flag,
  Plus,
  MessageSquare,
  Paperclip,
  Link2,
  Unlink,
  GitBranch,
} from 'lucide-react'
import { cn } from '@/lib'
import { useTaskActivities, type TaskActivity } from '@/api/hooks/useTaskActivities'

interface ActivityLogProps {
  taskId: string
}

function formatRelativeTime(dateStr?: string): string {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMin = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMs / 3600000)
  const diffDays = Math.floor(diffMs / 86400000)

  if (diffMin < 1) return 'gerade eben'
  if (diffMin < 60) return `vor ${diffMin} Min.`
  if (diffHours < 24) return `vor ${diffHours} Std.`
  if (diffDays < 7) return `vor ${diffDays} Tag${diffDays === 1 ? '' : 'en'}`
  return date.toLocaleDateString('de-DE', {
    day: '2-digit',
    month: '2-digit',
    year: '2-digit',
  })
}

function getActionIcon(action?: string) {
  switch (action) {
    case 'status_changed':
      return ArrowRight
    case 'assigned':
    case 'unassigned':
      return UserPlus
    case 'priority_changed':
      return Flag
    case 'created':
      return Plus
    case 'commented':
      return MessageSquare
    case 'attachment_added':
    case 'attachment_removed':
      return Paperclip
    case 'linked':
      return Link2
    case 'unlinked':
      return Unlink
    case 'dependency_added':
    case 'dependency_removed':
      return GitBranch
    default:
      return ArrowRight
  }
}

function getActionColor(action?: string): string {
  switch (action) {
    case 'status_changed':
      return 'text-blue-500 bg-blue-500/10'
    case 'assigned':
    case 'unassigned':
      return 'text-purple-500 bg-purple-500/10'
    case 'priority_changed':
      return 'text-orange-500 bg-orange-500/10'
    case 'created':
      return 'text-green-500 bg-green-500/10'
    case 'commented':
      return 'text-indigo-500 bg-indigo-500/10'
    case 'attachment_added':
    case 'attachment_removed':
      return 'text-teal-500 bg-teal-500/10'
    default:
      return 'text-muted-foreground bg-muted'
  }
}

function formatActivityDescription(activity: TaskActivity): string {
  const actor = activity.actor_name ?? 'Jemand'
  const oldVal = activity.old_value
  const newVal = activity.new_value

  switch (activity.action) {
    case 'status_changed':
      return `${actor} hat den Status von "${oldVal}" zu "${newVal}" geändert`
    case 'assigned':
      if (newVal && !oldVal) return `${actor} hat ${newVal} zugewiesen`
      if (!newVal && oldVal) return `${actor} hat die Zuweisung von ${oldVal} entfernt`
      return `${actor} hat die Zuweisung von ${oldVal} zu ${newVal} geändert`
    case 'unassigned':
      return `${actor} hat die Zuweisung entfernt`
    case 'priority_changed':
      return `${actor} hat die Priorität von "${oldVal}" auf "${newVal}" geändert`
    case 'created':
      return `${actor} hat die Aufgabe erstellt`
    case 'commented':
      return `${actor} hat kommentiert`
    case 'attachment_added':
      return `${actor} hat "${newVal}" angehängt`
    case 'attachment_removed':
      return `${actor} hat "${oldVal}" entfernt`
    case 'linked':
      return `${actor} hat ${newVal} verknüpft`
    case 'unlinked':
      return `${actor} hat ${oldVal} entfernt`
    case 'dependency_added':
      return `${actor} hat eine Abhängigkeit hinzugefügt: ${newVal}`
    case 'dependency_removed':
      return `${actor} hat eine Abhängigkeit entfernt: ${oldVal}`
    default:
      if (activity.field_name) {
        return `${actor} hat ${activity.field_name} von "${oldVal}" zu "${newVal}" geändert`
      }
      return `${actor} hat eine Änderung vorgenommen`
  }
}

export default function ActivityLog({ taskId }: ActivityLogProps) {
  const { data, isLoading } = useTaskActivities(taskId)
  const activities = data?.activities ?? []

  if (isLoading) {
    return (
      <div className="py-4 text-center text-sm text-muted-foreground">
        Aktivitäten werden geladen...
      </div>
    )
  }

  if (activities.length === 0) {
    return (
      <div className="py-6 text-center text-sm text-muted-foreground">
        Keine Aktivitäten vorhanden
      </div>
    )
  }

  return (
    <div className="space-y-1">
      {activities.map((activity) => {
        const Icon = getActionIcon(activity.action)
        const colorClass = getActionColor(activity.action)
        const isStatusChange = activity.action === 'status_changed'

        return (
          <div
            key={activity.id}
            className={cn(
              'flex items-start gap-3 rounded-md px-2 py-2 transition-colors',
              isStatusChange && 'bg-accent/30'
            )}
          >
            <div
              className={cn(
                'mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-full',
                colorClass
              )}
            >
              <Icon className="h-3.5 w-3.5" />
            </div>
            <div className="flex-1 min-w-0">
              <p
                className={cn(
                  'text-sm',
                  isStatusChange ? 'font-medium' : 'text-muted-foreground'
                )}
              >
                {formatActivityDescription(activity)}
              </p>
              <p className="text-xs text-muted-foreground">
                {formatRelativeTime(activity.created_at)}
              </p>
            </div>
          </div>
        )
      })}
    </div>
  )
}
