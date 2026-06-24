/**
 * TaskLabelPicker — popover to assign/unassign labels on a task.
 *
 * Taxonomy source: real backend via useWorkLabels (GET /api/v1/work/labels).
 * Assignment persistence: stores/taskLabels (local Zustand store).
 *
 * Assignment-to-backend gap:
 *   PUT /api/v1/tasks/{id}/labels expects a real UUID for {id}.
 *   The current task board still uses synthetic IDs (tsk-001 etc.) from the
 *   MSW mock layer. Once tasks are wired to the real backend (useTask hook
 *   returns real UUIDs), replace the toggleLabel store call with
 *   useSetTaskLabels().mutate({ taskId, labelIds: nextIds }).
 */
import { useTranslation } from 'react-i18next'
import { Tag, Plus, Check } from 'lucide-react'
import { cn } from '@/lib'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { useTaskLabelsStore } from '@/stores/taskLabels'
import { useWorkLabels } from '@/api/hooks/useWorkLabels'

const EMPTY: string[] = []

interface TaskLabelPickerProps {
  taskId: string
  /** Custom trigger; defaults to a small "+ Label" button. */
  children?: React.ReactNode
}

export default function TaskLabelPicker({ taskId, children }: TaskLabelPickerProps) {
  const { t } = useTranslation()
  const assigned = useTaskLabelsStore((s) => s.byTask[taskId]) ?? EMPTY
  const toggleLabel = useTaskLabelsStore((s) => s.toggleLabel)
  const { data } = useWorkLabels()
  const labels = data?.labels ?? []

  return (
    <Popover>
      <PopoverTrigger asChild>
        {children ?? (
          <button
            type="button"
            className="inline-flex items-center gap-1 rounded-full border border-dashed border-border px-2 py-0.5 text-[11px] font-medium text-muted-foreground transition-colors hover:border-primary/40 hover:text-foreground"
          >
            <Plus className="h-3 w-3" />
            {t('work.labels.add')}
          </button>
        )}
      </PopoverTrigger>
      <PopoverContent className="w-56 p-1" align="start">
        {labels.length === 0 ? (
          <div className="px-2 py-3 text-center">
            <Tag className="mx-auto h-5 w-5 text-muted-foreground/40" />
            <p className="mt-1.5 text-xs text-muted-foreground">{t('work.labels.noneDefined')}</p>
            <p className="text-[11px] text-muted-foreground/70">{t('work.labels.manageHint')}</p>
          </div>
        ) : (
          <div className="max-h-60 space-y-0.5 overflow-y-auto">
            {labels.map((label) => {
              const active = assigned.includes(label.id)
              return (
                <button
                  key={label.id}
                  type="button"
                  onClick={() => toggleLabel(taskId, label.id)}
                  className={cn(
                    'flex w-full items-center gap-2 rounded px-2 py-1.5 text-xs transition-colors hover:bg-accent',
                    active && 'bg-accent/60',
                  )}
                >
                  <span className="h-2.5 w-2.5 shrink-0 rounded-full" style={{ backgroundColor: label.color }} />
                  <span className="flex-1 truncate text-left text-foreground">{label.name}</span>
                  {active && <Check className="h-3.5 w-3.5 shrink-0 text-primary" />}
                </button>
              )
            })}
          </div>
        )}
      </PopoverContent>
    </Popover>
  )
}
