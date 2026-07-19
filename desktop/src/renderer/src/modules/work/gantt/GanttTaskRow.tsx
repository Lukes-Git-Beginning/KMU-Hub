/**
 * Individual task row with horizontal bar on the timeline.
 *
 * Left side (fixed): task info with key, title, assignee.
 * Right side (scrollable): colored bar at computed position with
 * tooltip, critical path highlight, completed styling, and interactive
 * drag-to-move / drag-to-resize functionality.
 */
import { useState, useRef, useCallback, useMemo, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { format } from 'date-fns'
import { de } from 'date-fns/locale'
import type { GanttTask, TimelineConfig } from './gantt-utils'
import {
  taskToBar,
  getBarColor,
  xToDate,
  ROW_HEIGHT,
  BAR_HEIGHT,
  TASK_INFO_WIDTH,
} from './gantt-utils'
import { useTaskCan } from '../useWorkPermissions'

// ---------------------------------------------------------------------------
// Drag state types
// ---------------------------------------------------------------------------

type DragMode = 'move' | 'resize-left' | 'resize-right'

interface DragState {
  mode: DragMode
  /** X pixel where the drag started (relative to the timeline area). */
  startMouseX: number
  /** Original bar X at drag start. */
  originalX: number
  /** Original bar width at drag start. */
  originalWidth: number
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

/** Pixel zone at each edge of the bar that triggers resize cursors. */
const RESIZE_HANDLE_WIDTH = 6

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface GanttTaskRowProps {
  task: GanttTask
  config: TimelineConfig
  rowIndex: number
  onTaskClick: (taskId: string) => void
  onTaskDateChange?: (taskId: string, newStart: Date, newEnd: Date) => void
  timelineWidth: number
}

export default function GanttTaskRow({
  task,
  config,
  rowIndex,
  onTaskClick,
  onTaskDateChange,
  timelineWidth,
}: GanttTaskRowProps) {
  const { t } = useTranslation()
  const can = useTaskCan(task)
  const bar = useMemo(() => taskToBar(task, config), [task, config])
  const barColor = useMemo(() => getBarColor(task), [task])
  const isCompleted = !!task.completed_at
  // Drag is only enabled when the user may edit this task
  const dragEnabled = can.edit && !isCompleted && !!onTaskDateChange

  // -----------------------------------------------------------------------
  // Drag state
  // -----------------------------------------------------------------------
  const [drag, setDrag] = useState<DragState | null>(null)
  const [ghostX, setGhostX] = useState<number>(0)
  const [ghostWidth, setGhostWidth] = useState<number>(0)
  const dragRef = useRef<DragState | null>(null)
  const barAreaRef = useRef<HTMLDivElement>(null)

  // Sync ref for event handlers
  useEffect(() => {
    dragRef.current = drag
  }, [drag])

  // -----------------------------------------------------------------------
  // Hover cursor state (resize handles)
  // -----------------------------------------------------------------------
  const [cursorStyle, setCursorStyle] = useState<string>('cursor-pointer')

  const handleBarMouseMove = useCallback(
    (e: React.MouseEvent<HTMLDivElement>) => {
      if (drag) return
      if (!bar || !dragEnabled) return
      const rect = e.currentTarget.getBoundingClientRect()
      const localX = e.clientX - rect.left
      const barWidth = e.currentTarget.offsetWidth

      if (localX <= RESIZE_HANDLE_WIDTH) {
        setCursorStyle('cursor-col-resize')
      } else if (localX >= barWidth - RESIZE_HANDLE_WIDTH) {
        setCursorStyle('cursor-col-resize')
      } else {
        setCursorStyle('cursor-grab')
      }
    },
    [bar, drag, dragEnabled]
  )

  // -----------------------------------------------------------------------
  // Drag start
  // -----------------------------------------------------------------------
  const handleBarMouseDown = useCallback(
    (e: React.MouseEvent<HTMLDivElement>) => {
      if (!bar || !barAreaRef.current || !dragEnabled) return
      e.preventDefault()
      e.stopPropagation()

      const rect = e.currentTarget.getBoundingClientRect()
      const localX = e.clientX - rect.left
      const barWidth = e.currentTarget.offsetWidth

      let mode: DragMode = 'move'
      if (localX <= RESIZE_HANDLE_WIDTH) {
        mode = 'resize-left'
      } else if (localX >= barWidth - RESIZE_HANDLE_WIDTH) {
        mode = 'resize-right'
      }

      // Use the timeline area container to compute mouse position
      const areaRect = barAreaRef.current.getBoundingClientRect()
      const mouseX = e.clientX - areaRect.left

      const state: DragState = {
        mode,
        startMouseX: mouseX,
        originalX: bar.x,
        originalWidth: bar.width,
      }

      setDrag(state)
      setGhostX(bar.x)
      setGhostWidth(bar.width)
    },
    [bar, isCompleted]
  )

  // -----------------------------------------------------------------------
  // Global mousemove / mouseup during drag
  // -----------------------------------------------------------------------
   
  useEffect(() => {
    if (!drag) return

    function handleMouseMove(e: MouseEvent) {
      const d = dragRef.current
      if (!d || !barAreaRef.current) return

      const areaRect = barAreaRef.current.getBoundingClientRect()
      const mouseX = e.clientX - areaRect.left
      const deltaX = mouseX - d.startMouseX

      if (d.mode === 'move') {
        const newX = Math.max(0, d.originalX + deltaX)
        setGhostX(newX)
        setGhostWidth(d.originalWidth)
      } else if (d.mode === 'resize-left') {
        const newX = Math.max(0, d.originalX + deltaX)
        const newWidth = Math.max(12, d.originalWidth - deltaX)
        // Don't let left edge go past right edge
        if (newX < d.originalX + d.originalWidth - 12) {
          setGhostX(newX)
          setGhostWidth(newWidth)
        }
      } else if (d.mode === 'resize-right') {
        const newWidth = Math.max(12, d.originalWidth + deltaX)
        setGhostX(d.originalX)
        setGhostWidth(newWidth)
      }
    }

    function handleMouseUp() {
      const d = dragRef.current
      if (!d) {
        setDrag(null)
        return
      }

      // Convert ghost pixel positions back to dates
      const newStart = xToDate(ghostX, config)
      const newEnd = xToDate(ghostX + ghostWidth, config)

      // Only fire callback if dates actually changed
      if (onTaskDateChange && (newStart.getTime() !== task.start?.getTime() || newEnd.getTime() !== task.end?.getTime())) {
        onTaskDateChange(task.id, newStart, newEnd)
      }

      setDrag(null)
    }

    window.addEventListener('mousemove', handleMouseMove)
    window.addEventListener('mouseup', handleMouseUp)
    return () => {
      window.removeEventListener('mousemove', handleMouseMove)
      window.removeEventListener('mouseup', handleMouseUp)
    }
   
  }, [drag, ghostX, ghostWidth, config, onTaskDateChange, task.id, task.start, task.end])

  // -----------------------------------------------------------------------
  // Task key display
  // -----------------------------------------------------------------------
  const taskKey =
    task.project_key && task.task_number
      ? `${task.project_key}-${task.task_number}`
      : task.task_number
        ? `#${task.task_number}`
        : ''

  // -----------------------------------------------------------------------
  // Tooltip text
  // -----------------------------------------------------------------------
  const tooltipText = [
    task.title,
    task.start ? `${t('work.gantt.tooltipStart')}: ${format(task.start, 'dd.MM.yyyy', { locale: de })}` : null,
    task.end ? `${t('work.gantt.tooltipDue')}: ${format(task.end, 'dd.MM.yyyy', { locale: de })}` : null,
    task.assignee_name ? `${t('work.gantt.tooltipAssignee')}: ${task.assignee_name}` : null,
    task.status_name ? `${t('common.status')}: ${task.status_name}` : null,
  ]
    .filter(Boolean)
    .join('\n')

  // -----------------------------------------------------------------------
  // Date tooltip during drag
  // -----------------------------------------------------------------------
  const dragTooltip = useMemo(() => {
    if (!drag) return null
    const newStart = xToDate(ghostX, config)
    const newEnd = xToDate(ghostX + ghostWidth, config)
    return `${format(newStart, 'dd.MM.yyyy', { locale: de })} - ${format(newEnd, 'dd.MM.yyyy', { locale: de })}`
  }, [drag, ghostX, ghostWidth, config])

  // -----------------------------------------------------------------------
  // Bar render position: use ghost during drag, original otherwise
  // -----------------------------------------------------------------------
  const renderX = drag ? ghostX : bar?.x ?? 0
  const renderWidth = drag ? ghostWidth : bar?.width ?? 0

  return (
    <div
      className={cn(
        'flex border-b border-border hover:bg-muted/20',
        rowIndex % 2 === 0 ? 'bg-background' : 'bg-muted/5'
      )}
      style={{ height: ROW_HEIGHT }}
    >
      {/* Fixed left: task info */}
      <div
        className="flex-shrink-0 flex items-center gap-2 border-r border-border px-3 overflow-hidden cursor-pointer hover:bg-muted/30"
        style={{ width: TASK_INFO_WIDTH }}
        onClick={() => onTaskClick(task.id)}
      >
        <span className="text-[10px] font-mono text-muted-foreground flex-shrink-0">
          {taskKey}
        </span>
        <span
          className={cn(
            'text-sm truncate flex-1',
            isCompleted && 'line-through text-muted-foreground'
          )}
        >
          {task.title}
        </span>
        {task.assignee_name && (
          <span
            className="flex-shrink-0 flex items-center justify-center rounded-full bg-muted text-[10px] font-medium"
            style={{ width: 20, height: 20 }}
            title={task.assignee_name}
          >
            {task.assignee_name.charAt(0).toUpperCase()}
          </span>
        )}
      </div>

      {/* Scrollable right: bar area */}
      <div
        ref={barAreaRef}
        className="relative flex-1 overflow-hidden"
        style={{ minWidth: timelineWidth }}
      >
        {bar ? (
          <>
            {/* Ghost bar (visible during drag) */}
            {drag && (
              <div
                className="absolute rounded opacity-30 pointer-events-none"
                style={{
                  left: bar.x,
                  top: (ROW_HEIGHT - BAR_HEIGHT) / 2,
                  width: bar.width,
                  height: BAR_HEIGHT,
                  backgroundColor: barColor,
                }}
              />
            )}

            {/* Main bar (or active drag bar) */}
            <Tooltip>
              <TooltipTrigger asChild>
                <div
                  className={cn(
                    'absolute rounded transition-shadow select-none',
                    drag ? 'cursor-grabbing shadow-lg ring-2 ring-primary/40 z-10' : cursorStyle,
                    !drag && 'hover:shadow-md',
                    task.is_on_critical_path && 'ring-2 ring-red-500',
                    isCompleted && 'opacity-50'
                  )}
                  style={{
                    left: renderX,
                    top: (ROW_HEIGHT - BAR_HEIGHT) / 2,
                    width: renderWidth,
                    height: BAR_HEIGHT,
                    backgroundColor: barColor,
                  }}
                  onMouseMove={handleBarMouseMove}
                  onMouseDown={handleBarMouseDown}
                  onClick={(e) => {
                    if (!drag) {
                      e.stopPropagation()
                      onTaskClick(task.id)
                    }
                  }}
                >
                  {/* Resize handles visual indicators */}
                  {dragEnabled && !drag && (
                    <>
                      <div
                        className="absolute left-0 top-0 bottom-0 w-1.5 rounded-l opacity-0 hover:opacity-100 bg-white/30"
                      />
                      <div
                        className="absolute right-0 top-0 bottom-0 w-1.5 rounded-r opacity-0 hover:opacity-100 bg-white/30"
                      />
                    </>
                  )}

                  {/* Title inside bar if wide enough */}
                  {renderWidth > 60 && (
                    <span className="absolute inset-0 flex items-center px-2 text-[10px] text-white truncate font-medium pointer-events-none">
                      {task.title}
                    </span>
                  )}
                  {isCompleted && (
                    <div
                      className="absolute top-1/2 left-0 right-0 border-t border-white/60 pointer-events-none"
                      style={{ transform: 'translateY(-50%)' }}
                    />
                  )}
                </div>
              </TooltipTrigger>
              <TooltipContent side="top" className="max-w-xs whitespace-pre-line text-xs">
                {drag ? dragTooltip : tooltipText}
              </TooltipContent>
            </Tooltip>

            {/* Snap date label during drag */}
            {drag && dragTooltip && (
              <div
                className="absolute z-20 rounded bg-foreground/90 text-background px-2 py-0.5 text-[10px] font-mono whitespace-nowrap pointer-events-none"
                style={{
                  left: renderX + renderWidth / 2,
                  top: (ROW_HEIGHT - BAR_HEIGHT) / 2 - 20,
                  transform: 'translateX(-50%)',
                }}
              >
                {dragTooltip}
              </div>
            )}
          </>
        ) : (
          <div
            className="absolute flex items-center text-xs text-muted-foreground italic px-2"
            style={{ top: (ROW_HEIGHT - BAR_HEIGHT) / 2, height: BAR_HEIGHT }}
          >
            {t('work.gantt.noDueDate')}
          </div>
        )}
      </div>
    </div>
  )
}
