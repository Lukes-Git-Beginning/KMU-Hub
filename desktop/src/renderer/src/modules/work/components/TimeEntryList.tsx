/**
 * Time entry list for a task.
 *
 * Shows all time entries with user, date, duration, description,
 * and edit/delete actions for entries owned by the current user.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Clock, Pencil, Trash2, User } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib'
import { useTimeEntries, useDeleteTimeEntry } from '@/api/hooks/useTimeEntries'
import { useAuthStore } from '@/stores/auth'
import ManualTimeEntryDialog from './ManualTimeEntryDialog'

interface TimeEntryListProps {
  taskId: string
}

/** Format seconds into "Xh Ymin" */
function formatDuration(seconds: number | undefined | null): string {
  if (!seconds || seconds <= 0) return '0min'
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  if (hours > 0 && minutes > 0) return `${hours}h ${minutes}min`
  if (hours > 0) return `${hours}h`
  if (minutes > 0) return `${minutes}min`
  return '<1min'
}

function formatDate(dateStr: string | undefined): string {
  if (!dateStr) return ''
  try {
    return new Date(dateStr).toLocaleDateString('de-DE', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
    })
  } catch {
    return ''
  }
}

function formatTime(dateStr: string | undefined): string {
  if (!dateStr) return ''
  try {
    return new Date(dateStr).toLocaleTimeString('de-DE', {
      hour: '2-digit',
      minute: '2-digit',
    })
  } catch {
    return ''
  }
}

export default function TimeEntryList({ taskId }: TimeEntryListProps) {
  const { t } = useTranslation()
  const { data, isLoading } = useTimeEntries(taskId)
  const deleteEntry = useDeleteTimeEntry()
  const currentUserId = useAuthStore((s) => s.user?.id)

  const [editEntry, setEditEntry] = useState<{
    id: string
    started_at: string
    duration_seconds: number
    description: string
  } | null>(null)

  const entries = data?.entries ?? []

  if (isLoading) {
    return (
      <div className="text-xs text-muted-foreground py-2">
        {t('work.timeEntries.loading')}
      </div>
    )
  }

  if (entries.length === 0) {
    return (
      <div className="text-xs text-muted-foreground py-2">
        {t('work.timeEntries.empty')}
      </div>
    )
  }

  return (
    <>
      <div className="space-y-0.5 rounded-md border border-border">
        {entries.map((entry) => {
          const isOwner = entry.user_id === currentUserId
          const isRunning = !entry.ended_at

          return (
            <div
              key={entry.id}
              className={cn(
                'flex items-start gap-2 px-3 py-2 text-xs border-b border-border last:border-b-0',
                isRunning && 'bg-green-50 dark:bg-green-950/30'
              )}
            >
              <div className="flex-1 min-w-0 space-y-0.5">
                <div className="flex items-center gap-2">
                  <User className="h-3 w-3 text-muted-foreground shrink-0" />
                  <span className="font-medium truncate">
                    {entry.user_name || t('work.tasks.user')}
                  </span>
                  <span className="text-muted-foreground">
                    {formatDate(entry.started_at)}
                  </span>
                  <span className="text-muted-foreground">
                    {formatTime(entry.started_at)}
                  </span>
                </div>
                <div className="flex items-center gap-2">
                  <Clock className="h-3 w-3 text-muted-foreground shrink-0" />
                  <span className="font-mono font-medium">
                    {isRunning ? (
                      <span className="text-green-600 dark:text-green-400">
                        {t('work.timeEntries.running')}
                      </span>
                    ) : (
                      formatDuration(entry.duration_seconds)
                    )}
                  </span>
                  {entry.is_manual && (
                    <span className="text-muted-foreground">({t('work.timeEntries.manual')})</span>
                  )}
                </div>
                {entry.description && (
                  <p className="text-muted-foreground mt-0.5 line-clamp-2">
                    {entry.description}
                  </p>
                )}
              </div>

              {isOwner && !isRunning && (
                <div className="flex items-center gap-0.5 shrink-0">
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-6 w-6 p-0"
                    title={t('common.edit')}
                    onClick={() =>
                      setEditEntry({
                        id: entry.id ?? '',
                        started_at: entry.started_at ?? '',
                        duration_seconds: entry.duration_seconds ?? 0,
                        description: entry.description ?? '',
                      })
                    }
                  >
                    <Pencil className="h-3 w-3" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-6 w-6 p-0 text-destructive hover:text-destructive"
                    title={t('common.delete')}
                    onClick={() => {
                      if (entry.id) {
                        deleteEntry.mutate({
                          entryId: entry.id,
                          taskId,
                        })
                      }
                    }}
                    disabled={deleteEntry.isPending}
                  >
                    <Trash2 className="h-3 w-3" />
                  </Button>
                </div>
              )}
            </div>
          )
        })}
      </div>

      {/* Edit dialog */}
      {editEntry && (
        <ManualTimeEntryDialog
          open={true}
          onOpenChange={(open) => {
            if (!open) setEditEntry(null)
          }}
          taskId={taskId}
          editMode
          editEntryId={editEntry.id}
          initialStartedAt={editEntry.started_at}
          initialDurationSeconds={editEntry.duration_seconds}
          initialDescription={editEntry.description}
        />
      )}
    </>
  )
}
