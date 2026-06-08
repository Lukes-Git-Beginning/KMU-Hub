/**
 * TaskLabelChips — renders the labels assigned to a task as tinted colour chips.
 *
 * Resolves the task's label ids (stores/taskLabels) against the tenant label
 * taxonomy (stores/workSettings). Renders nothing when the task has no labels.
 */
import { cn } from '@/lib'
import { useTaskLabelsStore } from '@/stores/taskLabels'
import { useWorkSettingsStore } from '@/stores/workSettings'

const EMPTY: string[] = []

interface TaskLabelChipsProps {
  taskId: string
  /** Cap the number of chips shown; extras collapse into a "+N" pill. */
  max?: number
  size?: 'xs' | 'sm'
  className?: string
}

export default function TaskLabelChips({ taskId, max, size = 'sm', className }: TaskLabelChipsProps) {
  const ids = useTaskLabelsStore((s) => s.byTask[taskId]) ?? EMPTY
  const labels = useWorkSettingsStore((s) => s.labels)

  if (ids.length === 0) return null
  const resolved = ids
    .map((id) => labels.find((l) => l.id === id))
    .filter((l): l is NonNullable<typeof l> => Boolean(l))
  if (resolved.length === 0) return null

  const shown = max ? resolved.slice(0, max) : resolved
  const overflow = resolved.length - shown.length

  const pad = size === 'xs' ? 'px-1.5 py-0 text-[10px]' : 'px-2 py-0.5 text-[11px]'

  return (
    <div className={cn('flex flex-wrap items-center gap-1', className)}>
      {shown.map((label) => (
        <span
          key={label.id}
          className={cn('inline-flex items-center rounded-full font-medium leading-tight', pad)}
          style={{
            backgroundColor: `${label.color}1a`,
            color: label.color,
            border: `1px solid ${label.color}40`,
          }}
        >
          {label.name}
        </span>
      ))}
      {overflow > 0 && (
        <span className={cn('inline-flex items-center rounded-full bg-secondary font-medium leading-tight text-muted-foreground', pad)}>
          +{overflow}
        </span>
      )}
    </div>
  )
}
